package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type LoopConfig struct {
	Lane            Lane
	Provider        string
	Model           string
	Groups          []int
	ReportFile      string
	MaxTurns        int
	MaxIdleTurns    int
	TimeoutSeconds  int
	TurnDelayMs     int
	Finalize        bool
	SkipInit        bool
	BenchDir        string
	CommandLogFile  string
	MaxInputTokens  int
	MaxOutputTokens int
	Stdout          io.Writer
	Stderr          io.Writer
}

type LoopResult struct {
	ExitCode  int
	FinalText string
	Usage     UsageCounters
}

func RunLoop(cfg LoopConfig, runner Runner, shell *PersistentShell) LoopResult {
	promptCfg := PromptConfig{
		RepoRoot:    filepath.Dir(filepath.Dir(cfg.BenchDir)),
		BenchDir:    cfg.BenchDir,
		BenchRunDir: filepath.Join(cfg.BenchDir, "benchmark-run"),
		SetupPTFile: filepath.Join(cfg.BenchDir, "setup-pinchtab.md"),
		SetupABFile: filepath.Join(cfg.BenchDir, "setup-agent-browser.md"),
	}

	userPrompt := LaneUserPrompt(cfg.Lane, promptCfg, cfg.ReportFile, cfg.Groups)
	conversation := runner.InitialConversation(userPrompt)
	systemPromptText := SystemPrompt()

	var finalText string
	exitCode := 0
	idleTurns := 0
	lastAnswered := ReadProgress(cfg.ReportFile).Answered

	clearCommandLog(cfg.CommandLogFile)

	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		if turn > 1 && cfg.TurnDelayMs > 0 {
			time.Sleep(time.Duration(cfg.TurnDelayMs) * time.Millisecond)
		}

		usage := runner.Usage()
		if cfg.MaxInputTokens > 0 && usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens >= cfg.MaxInputTokens {
			finalText = fmt.Sprintf("Budget exceeded: input tokens (%d) >= limit (%d)", usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens, cfg.MaxInputTokens)
			exitCode = 4
			break
		}
		if cfg.MaxOutputTokens > 0 && usage.OutputTokens >= cfg.MaxOutputTokens {
			finalText = fmt.Sprintf("Budget exceeded: output tokens (%d) >= limit (%d)", usage.OutputTokens, cfg.MaxOutputTokens)
			exitCode = 4
			break
		}

		response, err := runner.Send(systemPromptText, conversation)
		if err != nil {
			finalText = fmt.Sprintf("API error: %v", err)
			exitCode = 1
			break
		}

		toolCalls := runner.ExtractToolCalls(response, time.Duration(cfg.TimeoutSeconds)*time.Second)
		if len(toolCalls) > 0 {
			results := executeToolCalls(shell, toolCalls, cfg.CommandLogFile)
			conversation = runner.AppendToolResults(conversation, response, results)

			summary := BuildProgressSummary(cfg.ReportFile)
			conversation = CompactConversation(cfg.Provider, conversation, summary)

			progress := ReadProgress(cfg.ReportFile)
			if progress.Answered > lastAnswered {
				idleTurns = 0
				lastAnswered = progress.Answered
			} else {
				idleTurns++
			}

			if idleTurns >= cfg.MaxIdleTurns {
				finalText = fmt.Sprintf("Stopped after %d consecutive turns without recording a benchmark step. Check %s for the command trace.",
					idleTurns, cfg.CommandLogFile)
				exitCode = 3
				break
			}
			continue
		}

		finalText = runner.ExtractFinalText(response)
		break
	}

	if finalText == "" {
		finalText = fmt.Sprintf("Stopped after reaching max turns (%d).", cfg.MaxTurns)
		exitCode = 2
	}

	if _, err := os.Stat(cfg.ReportFile); err == nil {
		recordUsage(cfg.BenchDir, cfg.ReportFile, runner)
		if cfg.Finalize {
			finalizeReport(cfg.BenchDir, cfg.ReportFile)
		}
	}

	if finalText != "" {
		fmt.Fprintln(cfg.Stdout, finalText)
	}

	usage := runner.Usage()
	fmt.Fprintf(cfg.Stdout,
		"\n[run-usage] provider=%s requests=%d input=%d cache_create=%d cache_read=%d output=%d total=%d\n",
		runner.Provider(),
		usage.RequestCount,
		usage.InputTokens,
		usage.CacheCreationInputTokens,
		usage.CacheReadInputTokens,
		usage.OutputTokens,
		usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens+usage.OutputTokens,
	)

	return LoopResult{
		ExitCode:  exitCode,
		FinalText: finalText,
		Usage:     usage,
	}
}

func clearCommandLog(path string) {
	if path != "" {
		os.WriteFile(path, []byte{}, 0o644)
	}
}

func executeToolCalls(shell *PersistentShell, calls []ToolCall, logFile string) []ToolExecutionResult {
	var results []ToolExecutionResult
	for _, call := range calls {
		output, exitCode, err := shell.Run(call.Command, time.Duration(call.TimeoutSeconds)*time.Second)
		var result ToolExecutionResult
		if err != nil {
			result = ToolExecutionResult{
				ID:      call.ID,
				IsError: true,
				Content: fmt.Sprintf("$ %s\n[runner_error]\n%s", call.Command, err.Error()),
			}
			appendCommandLog(logFile, call.Command, -1, err.Error())
		} else {
			trimmed := TrimToolOutput(output)
			result = ToolExecutionResult{
				ID:      call.ID,
				IsError: exitCode != 0,
				Content: FormatToolResult(call.Command, exitCode, output),
			}
			appendCommandLog(logFile, call.Command, exitCode, trimmed)
		}
		results = append(results, result)
	}
	return results
}

func appendCommandLog(path, command string, exitCode int, output string) {
	if path == "" {
		return
	}
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"command":   command,
		"exit_code": exitCode,
		"output":    output,
	}
	data, _ := json.Marshal(entry)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

func recordUsage(benchDir, reportFile string, runner Runner) {
	usage := runner.Usage()
	script := filepath.Join(benchDir, "scripts", "record-run-usage.sh")
	cmd := exec.Command(script,
		"--report-file", reportFile,
		"--provider", runner.Provider(),
		"--source", runner.Source(),
		"--request-count", fmt.Sprintf("%d", usage.RequestCount),
		"--input-tokens", fmt.Sprintf("%d", usage.InputTokens),
		"--output-tokens", fmt.Sprintf("%d", usage.OutputTokens),
		"--cache-creation-input-tokens", fmt.Sprintf("%d", usage.CacheCreationInputTokens),
		"--cache-read-input-tokens", fmt.Sprintf("%d", usage.CacheReadInputTokens),
	)
	cmd.Dir = benchDir
	cmd.Run()
}

func finalizeReport(benchDir, reportFile string) {
	script := filepath.Join(benchDir, "scripts", "finalize-report.sh")
	cmd := exec.Command(script, reportFile)
	cmd.Dir = benchDir
	cmd.Run()
}
