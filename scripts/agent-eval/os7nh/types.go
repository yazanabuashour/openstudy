package main

import (
	"context"
	"encoding/json"
	"time"
)

type runConfig struct {
	Parallel   int
	Variant    string
	Scenario   string
	RunRoot    string
	ReportDir  string
	ReportName string
	CodexBin   string
	RepoRoot   string
	CacheMode  string
}

type cacheConfig struct {
	Mode    string
	RunRoot string
}

type evalJob struct {
	Index    int
	Variant  string
	Scenario scenario
}

type scenario struct {
	ID     string
	Title  string
	Prompt string
}

type report struct {
	Metadata       reportMetadata        `json:"metadata"`
	Results        []jobResult           `json:"results"`
	ProductionGate productionGateSummary `json:"production_gate"`
}

type reportMetadata struct {
	GeneratedAt              time.Time    `json:"generated_at"`
	Model                    string       `json:"model"`
	ReasoningEffort          string       `json:"reasoning_effort"`
	Harness                  string       `json:"harness"`
	ConfiguredParallelism    int          `json:"configured_parallelism"`
	CacheMode                string       `json:"cache_mode"`
	HarnessElapsedSeconds    float64      `json:"harness_elapsed_seconds"`
	EffectiveParallelSpeedup float64      `json:"effective_parallel_speedup,omitempty"`
	ParallelEfficiency       float64      `json:"parallel_efficiency,omitempty"`
	PhaseTotals              phaseTimings `json:"phase_totals"`
	RunRootArtifactReference string       `json:"run_root_artifact_reference"`
	RawLogPlaceholder        string       `json:"raw_log_placeholder"`
	Variants                 []string     `json:"variants"`
	Scenarios                []string     `json:"scenarios"`
	ReleaseBlocking          bool         `json:"release_blocking"`
	RawLogsCommitted         bool         `json:"raw_logs_committed"`
	RawLogsNote              string       `json:"raw_logs_note"`
}

type phaseTimings struct {
	PrepareRunDir  float64 `json:"prepare_run_dir_seconds,omitempty"`
	CopyRepo       float64 `json:"copy_repo_seconds,omitempty"`
	InstallVariant float64 `json:"install_variant_seconds,omitempty"`
	WarmCache      float64 `json:"warm_cache_seconds,omitempty"`
	SeedData       float64 `json:"seed_data_seconds,omitempty"`
	AgentRun       float64 `json:"agent_run_seconds,omitempty"`
	ParseMetrics   float64 `json:"parse_metrics_seconds,omitempty"`
	Verify         float64 `json:"verify_seconds,omitempty"`
	Total          float64 `json:"total_seconds,omitempty"`
}

type jobResult struct {
	Variant                 string             `json:"variant"`
	Scenario                string             `json:"scenario"`
	ScenarioTitle           string             `json:"scenario_title"`
	Passed                  bool               `json:"passed"`
	Status                  string             `json:"status"`
	Error                   string             `json:"error,omitempty"`
	ExitCode                int                `json:"exit_code"`
	WallSeconds             float64            `json:"wall_seconds"`
	PhaseTimings            phaseTimings       `json:"phase_timings"`
	Metrics                 metrics            `json:"metrics"`
	Verification            verificationResult `json:"verification"`
	PromptSummary           string             `json:"prompt_summary"`
	RawLogArtifactReference string             `json:"raw_log_artifact_reference"`
	StartedAt               time.Time          `json:"started_at"`
	CompletedAt             *time.Time         `json:"completed_at,omitempty"`
}

type metrics struct {
	AssistantCalls           int            `json:"assistant_calls"`
	ToolCalls                int            `json:"tool_calls"`
	CommandExecutions        int            `json:"command_executions"`
	FileInspectionCommands   int            `json:"file_inspection_commands"`
	ModuleCacheInspection    bool           `json:"module_cache_inspection"`
	BroadRepoSearch          bool           `json:"broad_repo_search"`
	DirectSQLiteAccess       bool           `json:"direct_sqlite_access"`
	SourceBuiltRunnerUsage   bool           `json:"source_built_runner_usage"`
	AdHocScriptUsage         bool           `json:"ad_hoc_script_usage"`
	RunnerCardsUsed          bool           `json:"runner_cards_used"`
	RunnerReviewUsed         bool           `json:"runner_review_used"`
	RunnerSourcesUsed        bool           `json:"runner_sources_used"`
	RunnerWindowsUsed        bool           `json:"runner_windows_used"`
	ValidationRejected       bool           `json:"validation_rejected"`
	UsageExposed             bool           `json:"usage_exposed"`
	InputTokens              *int           `json:"input_tokens,omitempty"`
	CachedInputTokens        *int           `json:"cached_input_tokens,omitempty"`
	NonCachedInputTokens     *int           `json:"non_cached_input_tokens,omitempty"`
	OutputTokens             *int           `json:"output_tokens,omitempty"`
	EventTypeCounts          map[string]int `json:"event_type_counts"`
	BypassEvidence           []string       `json:"bypass_evidence,omitempty"`
	CommandMetricLimitations string         `json:"command_metric_limitations"`
}

type verificationResult struct {
	Passed        bool   `json:"passed"`
	DatabasePass  bool   `json:"database_pass"`
	AssistantPass bool   `json:"assistant_pass"`
	Details       string `json:"details"`
}

type productionGateSummary struct {
	Variant        string                    `json:"variant"`
	PassesGate     bool                      `json:"passes_gate"`
	Recommendation string                    `json:"recommendation"`
	Criteria       []productionGateCriterion `json:"criteria"`
}

type productionGateCriterion struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details"`
}

type evalPaths struct {
	DatabasePath string
	GoCache      string
	GoModCache   string
	CodexHome    string
	ZDotDir      string
	Temp         string
}

type codexEvent struct {
	Type      string          `json:"type"`
	ThreadID  string          `json:"thread_id"`
	Item      json.RawMessage `json:"item"`
	Action    json.RawMessage `json:"action"`
	Usage     *usage          `json:"usage"`
	Command   string          `json:"command"`
	Cmd       string          `json:"cmd"`
	ParsedCmd json.RawMessage `json:"parsed_cmd"`
}

type usage struct {
	InputTokens        int           `json:"input_tokens"`
	OutputTokens       int           `json:"output_tokens"`
	CachedInputTokens  int           `json:"cached_input_tokens"`
	InputTokensDetails *usageDetails `json:"input_tokens_details"`
	PromptTokens       int           `json:"prompt_tokens"`
	CompletionTokens   int           `json:"completion_tokens"`
	PromptDetails      *usageDetails `json:"prompt_tokens_details"`
}

type usageDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type parsedTurn struct {
	metrics      metrics
	finalMessage string
	parseError   error
	parseSeconds float64
}

type jobRunner func(context.Context, runConfig, evalJob, cacheConfig) jobResult
