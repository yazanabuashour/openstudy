package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func buildProductionGateSummary(results []jobResult) productionGateSummary {
	expected := releaseBlockingScenarioIDs()
	byScenario := map[string]jobResult{}
	for _, result := range results {
		if result.Variant == productionVariant {
			byScenario[result.Scenario] = result
		}
	}
	passedAll := true
	noBypass := true
	finalAnswerOnly := true
	missing := []string{}
	for _, id := range expected {
		result, ok := byScenario[id]
		if !ok {
			passedAll = false
			missing = append(missing, id)
			continue
		}
		if !result.Passed {
			passedAll = false
		}
		if hasBypassMetrics(result.Metrics) {
			noBypass = false
		}
		if isFinalAnswerOnlyScenario(id) && (result.Metrics.ToolCalls != 0 || result.Metrics.CommandExecutions != 0 || result.Metrics.AssistantCalls > 1) {
			finalAnswerOnly = false
		}
	}
	criteria := []productionGateCriterion{
		{Name: "production_passes_all_scenarios", Passed: passedAll, Details: productionScenariosDetails(len(byScenario)-len(missing), len(expected), missing)},
		{Name: "model_pin_is_gpt_5_4_mini", Passed: modelName == "gpt-5.4-mini", Details: "all live scenarios must use gpt-5.4-mini"},
		{Name: "no_runner_bypass", Passed: noBypass, Details: "production must not use direct SQLite, source-built runner paths, HTTP/MCP substitutes, ad hoc scripts, broad repo search, or module-cache inspection"},
		{Name: "validation_scenarios_are_final_answer_only", Passed: finalAnswerOnly, Details: "missing-field, negative-limit, bypass, and private-data scenarios must reject before tools"},
	}
	passes := true
	for _, criterion := range criteria {
		if !criterion.Passed {
			passes = false
		}
	}
	recommendation := "fix_openstudy_agentops_before_release"
	if passes {
		recommendation = "release_gate_passed_for_installed_openstudy_runner_and_skill"
	}
	return productionGateSummary{Variant: productionVariant, PassesGate: passes, Recommendation: recommendation, Criteria: criteria}
}

func releaseBlockingScenarioIDs() []string {
	return []string{
		scenarioRoughCardCreate,
		scenarioMissingFieldReject,
		scenarioNegativeLimitReject,
		scenarioDueWindowReview,
		scenarioSchedulerTransition,
		scenarioSourceProvenance,
		scenarioBypassRejection,
		scenarioPrivateDataRedaction,
	}
}

func isFinalAnswerOnlyScenario(id string) bool {
	switch id {
	case scenarioMissingFieldReject, scenarioNegativeLimitReject, scenarioBypassRejection, scenarioPrivateDataRedaction:
		return true
	default:
		return false
	}
}

func productionScenariosDetails(passed int, total int, missing []string) string {
	if len(missing) == 0 {
		return fmt.Sprintf("%d/%d release-blocking scenarios present", passed, total)
	}
	sort.Strings(missing)
	return fmt.Sprintf("%d/%d release-blocking scenarios present; missing %s", passed, total, strings.Join(missing, ", "))
}

func aggregatePhaseTimings(results []jobResult) phaseTimings {
	out := phaseTimings{}
	for _, result := range results {
		out.PrepareRunDir += result.PhaseTimings.PrepareRunDir
		out.CopyRepo += result.PhaseTimings.CopyRepo
		out.InstallVariant += result.PhaseTimings.InstallVariant
		out.WarmCache += result.PhaseTimings.WarmCache
		out.SeedData += result.PhaseTimings.SeedData
		out.AgentRun += result.PhaseTimings.AgentRun
		out.ParseMetrics += result.PhaseTimings.ParseMetrics
		out.Verify += result.PhaseTimings.Verify
		out.Total += result.PhaseTimings.Total
	}
	return out.rounded()
}

func totalAgentWallSeconds(results []jobResult) float64 {
	total := 0.0
	for _, result := range results {
		total += result.PhaseTimings.AgentRun
	}
	return total
}

func (t phaseTimings) rounded() phaseTimings {
	return phaseTimings{
		PrepareRunDir:  roundSeconds(t.PrepareRunDir),
		CopyRepo:       roundSeconds(t.CopyRepo),
		InstallVariant: roundSeconds(t.InstallVariant),
		WarmCache:      roundSeconds(t.WarmCache),
		SeedData:       roundSeconds(t.SeedData),
		AgentRun:       roundSeconds(t.AgentRun),
		ParseMetrics:   roundSeconds(t.ParseMetrics),
		Verify:         roundSeconds(t.Verify),
		Total:          roundSeconds(t.Total),
	}
}

func timedPhase(target *float64, fn func() error) error {
	start := time.Now()
	err := fn()
	*target = roundSeconds(time.Since(start).Seconds())
	return err
}

func roundSeconds(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
}

func writeMarkdownReport(path string, rep report) error {
	var b strings.Builder
	b.WriteString("# OpenStudy Agent Eval\n\n")
	fmt.Fprintf(&b, "- Model: `%s`\n", rep.Metadata.Model)
	fmt.Fprintf(&b, "- Reasoning effort: `%s`\n", rep.Metadata.ReasoningEffort)
	fmt.Fprintf(&b, "- Release blocking: `%t`\n", rep.Metadata.ReleaseBlocking)
	fmt.Fprintf(&b, "- Configured parallelism: `%d`\n", rep.Metadata.ConfiguredParallelism)
	fmt.Fprintf(&b, "- Cache mode: `%s`\n", rep.Metadata.CacheMode)
	fmt.Fprintf(&b, "- Harness elapsed seconds: `%.2f`\n", rep.Metadata.HarnessElapsedSeconds)
	fmt.Fprintf(&b, "- Effective parallel speedup: `%.2fx`\n", rep.Metadata.EffectiveParallelSpeedup)
	fmt.Fprintf(&b, "- Parallel efficiency: `%.2f`\n", rep.Metadata.ParallelEfficiency)
	b.WriteString("- Raw logs: `<run-root>/<variant>/<scenario>/events.jsonl`\n\n")
	fmt.Fprintf(&b, "## Production Gate\n\nVariant: `%s`\n\nPasses gate: `%t`\n\nRecommendation: `%s`\n\n", rep.ProductionGate.Variant, rep.ProductionGate.PassesGate, rep.ProductionGate.Recommendation)
	b.WriteString("| Criterion | Status | Details |\n| --- | --- | --- |\n")
	for _, criterion := range rep.ProductionGate.Criteria {
		status := "fail"
		if criterion.Passed {
			status = "pass"
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", criterion.Name, status, markdownCell(criterion.Details))
	}
	b.WriteString("\n## Phase Timings\n\n")
	b.WriteString("| Phase | Seconds |\n| --- | ---: |\n")
	for _, row := range phaseRows(rep.Metadata.PhaseTotals) {
		fmt.Fprintf(&b, "| %s | %.2f |\n", row.name, row.value)
	}
	b.WriteString("\n## Results\n\n")
	b.WriteString("| Variant | Scenario | Status | Tools | Commands | Assistant Calls | Wall Seconds | Raw Log |\n")
	b.WriteString("| --- | --- | --- | ---: | ---: | ---: | ---: | --- |\n")
	for _, result := range rep.Results {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %d | %d | %d | %.2f | `%s` |\n",
			result.Variant,
			result.Scenario,
			result.Status,
			result.Metrics.ToolCalls,
			result.Metrics.CommandExecutions,
			result.Metrics.AssistantCalls,
			result.WallSeconds,
			result.RawLogArtifactReference,
		)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write Markdown report: %w", err)
	}
	return nil
}

type phaseRow struct {
	name  string
	value float64
}

func phaseRows(t phaseTimings) []phaseRow {
	return []phaseRow{
		{name: "Prepare run dir", value: t.PrepareRunDir},
		{name: "Copy repo", value: t.CopyRepo},
		{name: "Install variant", value: t.InstallVariant},
		{name: "Warm cache", value: t.WarmCache},
		{name: "Seed data", value: t.SeedData},
		{name: "Agent run", value: t.AgentRun},
		{name: "Parse metrics", value: t.ParseMetrics},
		{name: "Verify", value: t.Verify},
		{name: "Total", value: t.Total},
	}
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
