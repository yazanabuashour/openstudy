package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/yazanabuashour/openstudy/internal/runner"
)

func TestRunnerVersionAndUsage(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"--version"}, {"version"}} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := run(args, strings.NewReader(""), &stdout, &stderr)
		if code != 0 {
			t.Fatalf("run %v exit = %d stderr=%s", args, code, stderr.String())
		}
		if got := strings.TrimSpace(stdout.String()); !strings.HasPrefix(got, "openstudy ") {
			t.Fatalf("version output = %q, want openstudy prefix", got)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "openstudy cards") {
		t.Fatalf("usage = %q", stdout.String())
	}
}

func TestResolvedVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		linkerVersion string
		info          *debug.BuildInfo
		ok            bool
		want          string
	}{
		{
			name:          "linker version wins",
			linkerVersion: "v0.1.0",
			info:          &debug.BuildInfo{Main: debug.Module{Version: "v0.0.9"}},
			ok:            true,
			want:          "v0.1.0",
		},
		{
			name: "module version",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}},
			ok:   true,
			want: "v0.1.0",
		},
		{
			name: "development fallback",
			info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			ok:   true,
			want: "dev",
		},
		{
			name: "missing build info fallback",
			want: "dev",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolvedVersion(tt.linkerVersion, tt.info, tt.ok); got != tt.want {
				t.Fatalf("resolvedVersion = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunnerCardsSourcesWindowsAndReviewRoundTrip(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "data", "openstudy.sqlite")
	createRequest := `{"action":"create_card","card":{"front":"What command lists ready work?","back":"bd ready"}}`
	var createResult runner.CardsTaskResult
	code, stderr := runJSON(t, []string{"cards", "--db", dbPath}, createRequest, &createResult)
	if code != 0 {
		t.Fatalf("create exit = %d stderr=%s", code, stderr)
	}
	if createResult.Card == nil || createResult.Card.ID == 0 || createResult.Card.Schedule == nil {
		t.Fatalf("create result = %+v", createResult)
	}
	cardID := createResult.Card.ID

	listRequest := `{"action":"list_cards","status":"active","limit":50}`
	var listResult runner.CardsTaskResult
	code, stderr = runJSON(t, []string{"cards", "--db", dbPath}, listRequest, &listResult)
	if code != 0 {
		t.Fatalf("list exit = %d stderr=%s", code, stderr)
	}
	if len(listResult.Cards) != 1 || listResult.Cards[0].ID != cardID || listResult.Cards[0].Schedule == nil {
		t.Fatalf("list result = %+v", listResult)
	}

	sourceRequest := `{"action":"attach_source","card_id":%d,"source":{"source_system":"external-notes","source_key":"note-123","label":"planning note"}}`
	var sourceResult runner.SourcesTaskResult
	code, stderr = runJSON(t, []string{"sources", "--db", dbPath}, sprintf(sourceRequest, cardID), &sourceResult)
	if code != 0 {
		t.Fatalf("source exit = %d stderr=%s", code, stderr)
	}
	if sourceResult.Source == nil || sourceResult.Source.SourceKey != "note-123" {
		t.Fatalf("source result = %+v", sourceResult)
	}

	var sourcesResult runner.SourcesTaskResult
	code, stderr = runJSON(t, []string{"sources", "--db", dbPath}, sprintf(`{"action":"list_sources","card_id":%d}`, cardID), &sourcesResult)
	if code != 0 {
		t.Fatalf("list sources exit = %d stderr=%s", code, stderr)
	}
	if len(sourcesResult.Sources) != 1 || sourcesResult.Sources[0].SourceSystem != "external-notes" {
		t.Fatalf("sources result = %+v", sourcesResult)
	}

	asOf := "2099-01-01T00:00:00Z"
	var windowResult runner.WindowsTaskResult
	code, stderr = runJSON(t, []string{"windows", "--db", dbPath}, `{"action":"due_cards","limit":10,"now":"`+asOf+`"}`, &windowResult)
	if code != 0 {
		t.Fatalf("window exit = %d stderr=%s", code, stderr)
	}
	if windowResult.Now != "2099-01-01T00:00:00Z" || len(windowResult.Cards) != 1 {
		t.Fatalf("window result = %+v", windowResult)
	}

	var sessionResult runner.ReviewTaskResult
	code, stderr = runJSON(t, []string{"review", "--db", dbPath}, `{"action":"start_session","session":{"card_limit":10,"time_limit_seconds":600},"now":"`+asOf+`"}`, &sessionResult)
	if code != 0 {
		t.Fatalf("start session exit = %d stderr=%s", code, stderr)
	}
	if sessionResult.Session == nil || sessionResult.Session.ID == 0 || len(sessionResult.Cards) != 1 {
		t.Fatalf("session result = %+v", sessionResult)
	}
	sessionID := sessionResult.Session.ID

	recordRequest := sprintf(`{"action":"record_answer","session_id":%d,"card_id":%d,"answer_text":"bd ready lists ready work","rating":"good","grader":"self","answered_at":"2099-01-01T00:05:00Z"}`, sessionID, cardID)
	var recordResult runner.ReviewTaskResult
	code, stderr = runJSON(t, []string{"review", "--db", dbPath}, recordRequest, &recordResult)
	if code != 0 {
		t.Fatalf("record answer exit = %d stderr=%s", code, stderr)
	}
	if recordResult.Attempt == nil || recordResult.Transition == nil || recordResult.Transition.After.Reps != 1 {
		t.Fatalf("record result = %+v", recordResult)
	}

	var summaryResult runner.ReviewTaskResult
	code, stderr = runJSON(t, []string{"review", "--db", dbPath}, sprintf(`{"action":"summary","session_id":%d}`, sessionID), &summaryResult)
	if code != 0 {
		t.Fatalf("summary exit = %d stderr=%s", code, stderr)
	}
	if summaryResult.SummaryDTO == nil || summaryResult.SummaryDTO.AttemptCount != 1 || summaryResult.SummaryDTO.RatingCounts["good"] != 1 {
		t.Fatalf("summary result = %+v", summaryResult)
	}

	var finishResult runner.ReviewTaskResult
	code, stderr = runJSON(t, []string{"review", "--db", dbPath}, sprintf(`{"action":"finish_session","session_id":%d}`, sessionID), &finishResult)
	if code != 0 {
		t.Fatalf("finish exit = %d stderr=%s", code, stderr)
	}
	if finishResult.Session == nil || finishResult.Session.Status != "completed" {
		t.Fatalf("finish result = %+v", finishResult)
	}

	var rejected runner.ReviewTaskResult
	code, stderr = runJSON(t, []string{"review", "--db", dbPath}, recordRequest, &rejected)
	if code != 0 {
		t.Fatalf("record after finish exit = %d stderr=%s", code, stderr)
	}
	if !rejected.Rejected || !strings.Contains(rejected.RejectionReason, "not active") {
		t.Fatalf("record after finish result = %+v", rejected)
	}

	var archiveResult runner.CardsTaskResult
	code, stderr = runJSON(t, []string{"cards", "--db", dbPath}, sprintf(`{"action":"archive_card","card_id":%d}`, cardID), &archiveResult)
	if code != 0 {
		t.Fatalf("archive exit = %d stderr=%s", code, stderr)
	}
	if archiveResult.Card == nil || archiveResult.Card.Status != "archived" {
		t.Fatalf("archive result = %+v", archiveResult)
	}
}

func TestRunnerValidationRejectionDoesNotCreateDatabase(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "data", "openstudy.sqlite")
	request := `{"action":"create_card","card":{"front":"Missing back"}}`
	var result runner.CardsTaskResult
	code, stderr := runJSON(t, []string{"cards", "--db", dbPath}, request, &result)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	if !result.Rejected || result.RejectionReason != "card.back is required" {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Dir(dbPath)); !os.IsNotExist(err) {
		t.Fatalf("data dir exists after rejected request: %v", err)
	}
}

func TestRunnerValidationRejectionsAcrossDomains(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "data", "openstudy.sqlite")
	tests := []struct {
		name   string
		args   []string
		input  string
		reason string
	}{
		{name: "cards negative limit", args: []string{"cards", "--db", dbPath}, input: `{"action":"list_cards","limit":-1}`, reason: "limit must be greater than or equal to 0"},
		{name: "sources missing key", args: []string{"sources", "--db", dbPath}, input: `{"action":"attach_source","card_id":1,"source":{"source_system":"external-notes"}}`, reason: "source.source_key is required"},
		{name: "windows malformed now", args: []string{"windows", "--db", dbPath}, input: `{"action":"due_cards","now":"tomorrow"}`, reason: "now must be an RFC3339 timestamp"},
		{name: "review negative card limit", args: []string{"review", "--db", dbPath}, input: `{"action":"start_session","session":{"card_limit":-1}}`, reason: "session.card_limit must be greater than or equal to 0"},
		{name: "review missing answer", args: []string{"review", "--db", dbPath}, input: `{"action":"record_answer","session_id":1,"card_id":1,"rating":"good","grader":"self"}`, reason: "answer_text is required"},
		{name: "cards unknown field", args: []string{"cards", "--db", dbPath}, input: `{"action":"validate","extra":true}`, reason: `json: unknown field "extra"`},
		{name: "windows type mismatch", args: []string{"windows", "--db", dbPath}, input: `{"action":"due_cards","limit":"many"}`, reason: `json: cannot unmarshal string into Go struct field WindowsTaskRequest.limit of type int`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var result struct {
				Rejected        bool   `json:"rejected"`
				RejectionReason string `json:"rejection_reason"`
			}
			code, stderr := runJSON(t, tt.args, tt.input, &result)
			if code != 0 {
				t.Fatalf("exit = %d stderr=%s", code, stderr)
			}
			if !result.Rejected || result.RejectionReason != tt.reason {
				t.Fatalf("result = %+v, want %q", result, tt.reason)
			}
		})
	}
}

func TestRunnerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		input  string
		want   int
		stderr string
	}{
		{name: "unknown command", args: []string{"unknown"}, input: `{}`, want: 2, stderr: "unknown openstudy command"},
		{name: "bad json", args: []string{"cards"}, input: `{`, want: 1, stderr: "decode cards request"},
		{name: "multiple json", args: []string{"cards"}, input: `{} {}`, want: 1, stderr: "multiple JSON values"},
		{name: "unexpected arg", args: []string{"windows", "extra"}, input: `{}`, want: 2, stderr: "unexpected positional arguments"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.input), &stdout, &stderr)
			if code != tt.want {
				t.Fatalf("exit = %d, want %d; stderr=%s", code, tt.want, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.stderr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.stderr)
			}
		})
	}
}

func runJSON(t *testing.T, args []string, input string, output any) (int, string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(args, strings.NewReader(input), &stdout, &stderr)
	if output != nil && stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), output); err != nil {
			t.Fatalf("decode stdout %q: %v", stdout.String(), err)
		}
	}
	return code, stderr.String()
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
