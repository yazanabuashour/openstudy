package main

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

var (
	unixHomePathPattern    = regexp.MustCompile(`/(Users|home)/[^/\s"'\\)]+`)
	windowsHomePathPattern = regexp.MustCompile(`(?i)\b[A-Z]:\\Users\\[^\\\s"']+`)
)

func parseMetrics(eventsPath string) (parsedTurn, error) {
	file, err := os.Open(eventsPath)
	if err != nil {
		return parsedTurn{metrics: emptyMetrics()}, err
	}
	defer func() { _ = file.Close() }()
	out := parsedTurn{metrics: emptyMetrics()}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inputTotal := 0
	cachedTotal := 0
	outputTotal := 0
	usageExposed := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type != "" {
			out.metrics.EventTypeCounts[event.Type]++
		}
		if event.Usage != nil {
			usageExposed = true
			input, cached, output := usageNumbers(*event.Usage)
			inputTotal += input
			cachedTotal += cached
			outputTotal += output
		}
		itemText := string(event.Item)
		if event.Type == "message" || strings.Contains(itemText, `"type":"message"`) || strings.Contains(itemText, `"type":"agent_message"`) {
			if strings.Contains(itemText, `"role":"assistant"`) || strings.Contains(itemText, `"type":"message"`) || strings.Contains(itemText, `"type":"agent_message"`) {
				out.metrics.AssistantCalls++
				if msg := extractAssistantText(event.Item); msg != "" {
					out.finalMessage = msg
				}
			}
		}
		commands := eventCommandTexts(event)
		if len(commands) > 0 {
			out.metrics.ToolCalls += len(commands)
		} else if event.Type == "tool_call" || strings.Contains(itemText, `"type":"tool_call"`) || strings.Contains(itemText, `"call_id"`) {
			out.metrics.ToolCalls++
		}
		for _, command := range commands {
			out.metrics.CommandExecutions++
			classifyCommand(command, &out.metrics)
		}
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}
	if usageExposed {
		nonCached := inputTotal - cachedTotal
		if nonCached < 0 {
			nonCached = 0
		}
		out.metrics.UsageExposed = true
		out.metrics.InputTokens = &inputTotal
		out.metrics.CachedInputTokens = &cachedTotal
		out.metrics.NonCachedInputTokens = &nonCached
		out.metrics.OutputTokens = &outputTotal
	}
	return out, nil
}

func emptyMetrics() metrics {
	return metrics{
		EventTypeCounts:          map[string]int{},
		CommandMetricLimitations: "Command/file inspection metrics are inferred from codex exec JSON command events, not OS-level tracing.",
	}
}

func usageNumbers(value usage) (input int, cached int, output int) {
	input = value.InputTokens
	if input == 0 {
		input = value.PromptTokens
	}
	output = value.OutputTokens
	if output == 0 {
		output = value.CompletionTokens
	}
	cached = value.CachedInputTokens
	if value.InputTokensDetails != nil {
		cached += value.InputTokensDetails.CachedTokens
	}
	if value.PromptDetails != nil {
		cached += value.PromptDetails.CachedTokens
	}
	return input, cached, output
}

func eventCommandTexts(event codexEvent) []string {
	commands := commandTexts(event.Item)
	commands = append(commands, commandTexts(event.Action)...)
	for _, command := range []string{event.Command, event.Cmd} {
		if strings.TrimSpace(command) != "" {
			commands = append(commands, command)
		}
	}
	if parsed := parsedCommandText(event.ParsedCmd); parsed != "" {
		commands = append(commands, parsed)
	}
	return dedupeStrings(commands)
}

func parsedCommandText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := []string{}
		for _, part := range typed {
			if s, ok := part.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func extractAssistantText(raw json.RawMessage) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	texts := []string{}
	collectTextValues(value, &texts)
	if len(texts) == 0 {
		return ""
	}
	return strings.Join(texts, "\n")
}

func collectTextValues(value any, texts *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		if role, _ := typed["role"].(string); role == "assistant" {
			if content, ok := typed["content"].(string); ok && strings.TrimSpace(content) != "" {
				*texts = append(*texts, content)
			}
		}
		if typ, _ := typed["type"].(string); typ == "agent_message" {
			if text, ok := typed["text"].(string); ok && strings.TrimSpace(text) != "" {
				*texts = append(*texts, text)
			}
		}
		if typ, _ := typed["type"].(string); typ == "output_text" || typ == "text" {
			if text, ok := typed["text"].(string); ok && strings.TrimSpace(text) != "" {
				*texts = append(*texts, text)
			}
		}
		for _, nested := range typed {
			collectTextValues(nested, texts)
		}
	case []any:
		for _, nested := range typed {
			collectTextValues(nested, texts)
		}
	}
}

func commandTexts(raw json.RawMessage) []string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	out := []string{}
	collectCommandTexts(value, &out)
	return out
}

func collectCommandTexts(value any, out *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"cmd", "command"} {
			switch command := typed[key].(type) {
			case string:
				if command != "" {
					*out = append(*out, command)
				}
			case []any:
				parts := []string{}
				for _, part := range command {
					if s, ok := part.(string); ok {
						parts = append(parts, s)
					}
				}
				if len(parts) > 0 {
					*out = append(*out, strings.Join(parts, " "))
				}
			}
		}
		for _, nested := range typed {
			collectCommandTexts(nested, out)
		}
	case []any:
		for _, nested := range typed {
			collectCommandTexts(nested, out)
		}
	}
}

func classifyCommand(command string, m *metrics) {
	lower := strings.ToLower(command)
	actionText := strings.ReplaceAll(lower, `\"`, `"`)
	evidence := sanitizeMetricEvidence(command)
	addEvidence := func() {
		if len(m.BypassEvidence) < 8 {
			m.BypassEvidence = append(m.BypassEvidence, evidence)
		}
	}
	if strings.Contains(command, "GOMODCACHE") || strings.Contains(command, "/pkg/mod") || strings.Contains(command, "go env GOMODCACHE") {
		m.ModuleCacheInspection = true
		addEvidence()
	}
	if strings.Contains(command, "rg --files") || isBroadFindCommand(command) {
		m.BroadRepoSearch = true
		addEvidence()
	}
	if strings.Contains(lower, "sqlite3") || strings.Contains(lower, "select ") || strings.Contains(lower, "pragma ") || strings.Contains(lower, ".sqlite") {
		m.DirectSQLiteAccess = true
		addEvidence()
	}
	if strings.Contains(command, "go run ./cmd/openstudy") || strings.Contains(command, "./cmd/openstudy") {
		m.SourceBuiltRunnerUsage = true
		addEvidence()
	}
	if strings.Contains(lower, "python ") || strings.Contains(lower, "python3 ") || strings.Contains(lower, "node ") {
		m.AdHocScriptUsage = true
		addEvidence()
	}
	if isFileInspectionCommand(lower) {
		m.FileInspectionCommands++
	}
	if commandUsesRunnerDomain(lower, "cards") {
		m.RunnerCardsUsed = true
	}
	if commandUsesRunnerDomain(lower, "review") {
		m.RunnerReviewUsed = true
	}
	if commandUsesRunnerDomain(lower, "sources") {
		m.RunnerSourcesUsed = true
	}
	if commandUsesRunnerDomain(lower, "windows") {
		m.RunnerWindowsUsed = true
	}
	if strings.Contains(actionText, `"rejected":true`) || strings.Contains(actionText, `"rejected": true`) {
		m.ValidationRejected = true
	}
}

func commandUsesRunnerDomain(command string, domain string) bool {
	return strings.Contains(command, "openstudy "+domain)
}

func isFileInspectionCommand(command string) bool {
	for _, prefix := range []string{"cat ", "sed ", "nl ", "head ", "tail ", "less ", "grep ", "rg "} {
		if strings.HasPrefix(strings.TrimSpace(command), prefix) {
			return true
		}
	}
	return false
}

func isBroadFindCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	if !strings.Contains(trimmed, "find .") && !strings.Contains(trimmed, "find ..") {
		return false
	}
	if strings.Contains(trimmed, "-type d") && !strings.Contains(trimmed, "-type f") {
		return false
	}
	return true
}

func sanitizeMetricEvidence(value string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		value = strings.ReplaceAll(value, home, "<home>")
	}
	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		value = strings.ReplaceAll(value, tmp, "<tmp>")
	}
	value = unixHomePathPattern.ReplaceAllString(value, "<home>")
	return windowsHomePathPattern.ReplaceAllString(value, "<home>")
}
