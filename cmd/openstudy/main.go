package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/yazanabuashour/openstudy/internal/runner"
)

var version string

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		writeUsage(stdout)
		return 0
	case "version", "--version":
		writeVersion(stdout)
		return 0
	case "cards":
		return runCards(args[1:], stdin, stdout, stderr)
	case "review":
		return runReview(args[1:], stdin, stdout, stderr)
	case "sources":
		return runSources(args[1:], stdin, stdout, stderr)
	case "windows":
		return runWindows(args[1:], stdin, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown openstudy command %q\n", args[0])
		writeUsage(stderr)
		return 2
	}
}

func runCards(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	config, ok := parseConfig("cards", args, stderr)
	if !ok {
		return 2
	}
	var request runner.CardsTaskRequest
	if rejection, err := decodeRequest(stdin, &request); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode cards request: %v\n", err)
		return 1
	} else if rejection != "" {
		return encodeResult(stdout, stderr, runner.CardsTaskResult{BaseResult: rejectedBase(rejection)})
	}
	result, err := runner.RunCardsTask(context.Background(), config, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run cards task: %v\n", err)
		return 1
	}
	return encodeResult(stdout, stderr, result)
}

func runReview(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	config, ok := parseConfig("review", args, stderr)
	if !ok {
		return 2
	}
	var request runner.ReviewTaskRequest
	if rejection, err := decodeRequest(stdin, &request); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode review request: %v\n", err)
		return 1
	} else if rejection != "" {
		return encodeResult(stdout, stderr, runner.ReviewTaskResult{BaseResult: rejectedBase(rejection)})
	}
	result, err := runner.RunReviewTask(context.Background(), config, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run review task: %v\n", err)
		return 1
	}
	return encodeResult(stdout, stderr, result)
}

func runSources(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	config, ok := parseConfig("sources", args, stderr)
	if !ok {
		return 2
	}
	var request runner.SourcesTaskRequest
	if rejection, err := decodeRequest(stdin, &request); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode sources request: %v\n", err)
		return 1
	} else if rejection != "" {
		return encodeResult(stdout, stderr, runner.SourcesTaskResult{BaseResult: rejectedBase(rejection)})
	}
	result, err := runner.RunSourcesTask(context.Background(), config, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run sources task: %v\n", err)
		return 1
	}
	return encodeResult(stdout, stderr, result)
}

func runWindows(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	config, ok := parseConfig("windows", args, stderr)
	if !ok {
		return 2
	}
	var request runner.WindowsTaskRequest
	if rejection, err := decodeRequest(stdin, &request); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode windows request: %v\n", err)
		return 1
	} else if rejection != "" {
		return encodeResult(stdout, stderr, runner.WindowsTaskResult{BaseResult: rejectedBase(rejection)})
	}
	result, err := runner.RunWindowsTask(context.Background(), config, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run windows task: %v\n", err)
		return 1
	}
	return encodeResult(stdout, stderr, result)
}

func parseConfig(name string, args []string, stderr io.Writer) (runner.Config, bool) {
	fs := flag.NewFlagSet("openstudy "+name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	databasePath := fs.String("db", "", "OpenStudy SQLite database path")
	if err := fs.Parse(args); err != nil {
		return runner.Config{}, false
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected positional arguments: %v\n", fs.Args())
		return runner.Config{}, false
	}
	return runner.Config{DatabasePath: *databasePath}, true
}

func decodeRequest[T any](stdin io.Reader, request *T) (string, error) {
	decoder := json.NewDecoder(stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		if schemaDecodeError(err) {
			return err.Error(), nil
		}
		return "", err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return "", errors.New("multiple JSON values are not supported")
		}
		return "", err
	}
	return "", nil
}

func schemaDecodeError(err error) bool {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return true
	}
	return strings.HasPrefix(err.Error(), "json: unknown field ")
}

func rejectedBase(reason string) runner.BaseResult {
	return runner.BaseResult{
		Rejected:        true,
		RejectionReason: reason,
		Summary:         reason,
	}
}

func encodeResult[T any](stdout io.Writer, stderr io.Writer, result T) int {
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		_, _ = fmt.Fprintf(stderr, "encode result: %v\n", err)
		return 1
	}
	return 0
}

func writeVersion(w io.Writer) {
	info, ok := readBuildInfo()
	_, _ = fmt.Fprintf(w, "openstudy %s\n", resolvedVersion(version, info, ok))
}

func readBuildInfo() (*debug.BuildInfo, bool) {
	return debug.ReadBuildInfo()
}

func resolvedVersion(linkerVersion string, info *debug.BuildInfo, ok bool) string {
	if linkerVersion != "" {
		return linkerVersion
	}
	if ok && info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func writeUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: openstudy <version|cards|review|sources|windows> [--db path]
       openstudy cards [--db path] < request.json
       openstudy review [--db path] < request.json
       openstudy sources [--db path] < request.json
       openstudy windows [--db path] < request.json
`)
}
