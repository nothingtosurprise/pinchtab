// Command api-runner is the Go port of
// tests/benchmark/scripts/run-api-benchmark.ts.
//
// Exit codes:
//
//	0  success (or --help)
//	1  argument/setup error
//	2  max turns reached
//	3  idle turn limit reached
//	4  budget exceeded
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	args, err := ParseArgs(argv)
	if err != nil {
		if errors.Is(err, errHelp) {
			WriteUsage(stdout)
			return 0
		}
		fmt.Fprintf(stderr, "api-runner: %v\n\n", err)
		WriteUsage(stderr)
		return 1
	}

	if args.DryRun {
		fmt.Fprint(stdout, formatPlan(args))
		return 0
	}

	benchDir := resolveBenchDir()
	provider := resolveProvider(args)
	model := resolveModel(provider, args.Model)
	groups := resolveGroupsFromArgs(args)

	reportFile := args.ReportFile
	if reportFile == "" {
		if args.SkipInit {
			reportFile = resolveReportPath(benchDir, args.Lane)
		} else {
			reportFile = initializeLane(benchDir, args, provider, model)
		}
	}

	fmt.Fprintf(stdout, "[benchmark-runner] provider=%s model=%s lane=%s groups=%s report=%s\n",
		provider, model, args.Lane, formatGroups(groups), reportFile)

	runner := createRunner(provider, model, args)
	shell, err := NewPersistentShell(benchDir)
	if err != nil {
		fmt.Fprintf(stderr, "api-runner: shell init failed: %v\n", err)
		return 1
	}
	defer shell.Close(false)

	cfg := LoopConfig{
		Lane:            args.Lane,
		Provider:        provider,
		Model:           model,
		Groups:          groups,
		ReportFile:      reportFile,
		MaxTurns:        args.MaxTurns,
		MaxIdleTurns:    args.MaxIdleTurns,
		TimeoutSeconds:  args.TimeoutSeconds,
		TurnDelayMs:     args.TurnDelayMs,
		Finalize:        args.Finalize,
		BenchDir:        benchDir,
		CommandLogFile:  filepath.Join(benchDir, "results", "agent_commands.ndjson"),
		MaxInputTokens:  args.MaxInputTokens,
		MaxOutputTokens: args.MaxOutputTokens,
		Stdout:          stdout,
		Stderr:          stderr,
	}

	result := RunLoop(cfg, runner, shell)
	return result.ExitCode
}

func resolveBenchDir() string {
	cwd, _ := os.Getwd()
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "tests", "benchmark")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(cwd, "tests", "benchmark")
		}
		dir = parent
	}
}

func resolveProvider(args Args) string {
	if args.Provider != ProviderUnset {
		return string(args.Provider)
	}
	hasOpenAI := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != ""
	hasAnthropic := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != ""
	if hasOpenAI && hasAnthropic {
		fmt.Fprintln(os.Stderr, "warning: multiple providers configured, defaulting to anthropic")
		return "anthropic"
	}
	if hasOpenAI {
		return "openai"
	}
	if hasAnthropic {
		return "anthropic"
	}
	return "anthropic"
}

func resolveModel(provider, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if provider == "openai" {
		if m := os.Getenv("OPENAI_MODEL"); m != "" {
			return m
		}
		return "gpt-5"
	}
	if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
		return m
	}
	return "claude-haiku-4-5-20251001"
}

func resolveGroupsFromArgs(args Args) []int {
	if len(args.Groups) > 0 {
		return args.Groups
	}
	switch args.Profile {
	case "common10":
		return []int{0, 1, 2, 3}
	default:
		return nil
	}
}

func resolveReportPath(benchDir string, lane Lane) string {
	ptrFile := filepath.Join(benchDir, "results", "current_agent_report.txt")
	if lane == LaneAgentBrowser {
		ptrFile = filepath.Join(benchDir, "results", "current_agent_browser_report.txt")
	}
	data, err := os.ReadFile(ptrFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func initializeLane(benchDir string, args Args, provider, model string) string {
	var script string
	if args.Lane == LanePinchtab {
		script = filepath.Join(benchDir, "scripts", "run-optimization.sh")
	} else {
		script = filepath.Join(benchDir, "scripts", "run-agent-browser-benchmark.sh")
	}
	cmd := exec.Command(script)
	cmd.Dir = benchDir
	cmd.Env = append(os.Environ(),
		"BENCHMARK_MODEL="+model,
		"BENCHMARK_RUNNER="+runnerSource(provider),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return resolveReportPath(benchDir, args.Lane)
}

func runnerSource(provider string) string {
	if provider == "openai" {
		return "openai-responses"
	}
	return "anthropic-messages"
}

func createRunner(provider, model string, args Args) Runner {
	promptCaching := !args.NoPromptCaching
	if provider == "openai" {
		apiKey := os.Getenv("OPENAI_API_KEY")
		return NewOpenAIRunner(apiKey, model, args.MaxTokens, args.Temperature, promptCaching)
	}
	if provider == "fake" {
		return NewFakeRunner(model, nil)
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	return NewAnthropicRunner(apiKey, model, args.MaxTokens, args.Temperature, promptCaching)
}

func formatPlan(a Args) string {
	var b strings.Builder
	b.WriteString("api-runner (Go) — resolved plan\n")
	fmt.Fprintf(&b, "  lane:              %s\n", a.Lane)
	fmt.Fprintf(&b, "  provider:          %s\n", stringOr(string(a.Provider), "(auto-detect from env)"))
	fmt.Fprintf(&b, "  model:             %s\n", stringOr(a.Model, "(provider default)"))
	fmt.Fprintf(&b, "  groups:            %s\n", formatGroups(a.Groups))
	fmt.Fprintf(&b, "  profile:           %s\n", stringOr(a.Profile, "(none)"))
	fmt.Fprintf(&b, "  max-tokens:        %d\n", a.MaxTokens)
	fmt.Fprintf(&b, "  temperature:       %g\n", a.Temperature)
	fmt.Fprintf(&b, "  max-turns:         %d\n", a.MaxTurns)
	fmt.Fprintf(&b, "  max-idle-turns:    %d\n", a.MaxIdleTurns)
	fmt.Fprintf(&b, "  timeout-seconds:   %d\n", a.TimeoutSeconds)
	fmt.Fprintf(&b, "  turn-delay-ms:     %d\n", a.TurnDelayMs)
	fmt.Fprintf(&b, "  report-file:       %s\n", stringOr(a.ReportFile, "(auto-generated)"))
	fmt.Fprintf(&b, "  skip-init:         %t\n", a.SkipInit)
	fmt.Fprintf(&b, "  no-prompt-caching: %t\n", a.NoPromptCaching)
	fmt.Fprintf(&b, "  finalize:          %t\n", a.Finalize)
	fmt.Fprintf(&b, "  dry-run:           %t\n", a.DryRun)
	fmt.Fprintf(&b, "  index-file:        %s\n", stringOr(a.IndexFile, "(default)"))
	return b.String()
}

func stringOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func formatGroups(groups []int) string {
	if len(groups) == 0 {
		return "all"
	}
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = fmt.Sprintf("%d", g)
	}
	return strings.Join(parts, ",")
}
