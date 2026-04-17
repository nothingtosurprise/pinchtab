package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PromptConfig struct {
	RepoRoot     string
	BenchDir     string
	BenchRunDir  string
	SetupPTFile  string
	SetupABFile  string
}

func DefaultPromptConfig() PromptConfig {
	cwd, _ := os.Getwd()
	repoRoot := findRepoRoot(cwd)
	benchDir := filepath.Join(repoRoot, "tests", "benchmark")
	return PromptConfig{
		RepoRoot:    repoRoot,
		BenchDir:    benchDir,
		BenchRunDir: filepath.Join(benchDir, "benchmark-run"),
		SetupPTFile: filepath.Join(benchDir, "setup-pinchtab.md"),
		SetupABFile: filepath.Join(benchDir, "setup-agent-browser.md"),
	}
}

func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}

type LaneConfig struct {
	Label             string
	SkillInstruction  string
	Wrapper           string
	RecordType        string
	WorkflowSummary   []string
	AdapterNotes      []string
	BootstrapCommands []string
}

func LanePromptConfig(lane Lane, cfg PromptConfig) LaneConfig {
	if lane == LanePinchtab {
		return LaneConfig{
			Label:            "PinchTab agent",
			SkillInstruction: fmt.Sprintf("Read %s exactly once before acting.", cfg.SetupPTFile),
			Wrapper:          "./scripts/pt",
			RecordType:       "pinchtab",
			WorkflowSummary: []string{
				"use only ./scripts/pt for browser actions",
				"keep one shared tab via PINCHTAB_TAB",
				"use snap -i -c for actionable refs",
				"use text or text --full for reading content",
				"refresh refs after any navigation or DOM change",
			},
			AdapterNotes: []string{
				`a navigation-expected click can return "Error 409: unexpected page navigation ..."; treat that as likely success and verify with a fresh snapshot/text read`,
				"do not assume every command returns JSON; only parse JSON when the command actually returned JSON",
				`the download endpoint returns JSON with base64 content in "data", not a local file path`,
				`this environment uses BSD/macOS userland tools; avoid GNU-only flags such as "head -n -1"`,
			},
			BootstrapCommands: []string{
				"./scripts/pt health",
				"./scripts/pt tab",
				"export PINCHTAB_TAB=$(./scripts/pt nav http://fixtures/)",
				"./scripts/pt snap -i -c",
			},
		}
	}
	return LaneConfig{
		Label:            "agent-browser",
		SkillInstruction: fmt.Sprintf("Read %s exactly once before acting, then load the official agent-browser skill exactly once with `./scripts/ab skills get agent-browser --full`.", cfg.SetupABFile),
		Wrapper:          "./scripts/ab",
		RecordType:       "agent-browser",
		WorkflowSummary: []string{
			"use only ./scripts/ab for browser actions",
			"keep the shared benchmark session across commands",
			"use snapshot -i -c for actionable refs",
			"use fresh refs after any navigation or DOM change",
			"reuse the current browser state instead of reopening pages unless needed",
		},
		AdapterNotes: []string{
			"after navigation-triggering clicks, verify the resulting page with a fresh snapshot/text read instead of trusting the click response alone",
			"do not assume every command returns JSON; parse only when the command actually returned JSON",
			"for downloads, inspect returned content instead of assuming a file already exists locally",
			`this environment uses BSD/macOS userland tools; avoid GNU-only flags such as "head -n -1"`,
		},
		BootstrapCommands: []string{
			"./scripts/ab skills get agent-browser --full",
			"./scripts/ab open http://fixtures/",
			"./scripts/ab snapshot -i -c",
		},
	}
}

func SystemPrompt() string {
	return `You are a precise benchmark execution agent. Use tools to inspect the repo and run the benchmark lane exactly as instructed.

Rules:
- Never fabricate command output or task results.
- Use the shell tool for all file reads and command execution.
- Do not use destructive commands such as rm -rf, git reset, or checkout.
- After recording an answer, verify it immediately against the task oracle.
- Prefer factual command output over long reasoning.`
}

func LaneSubsetInstructions(groups []int) string {
	if len(groups) == 0 {
		return "Execute the full benchmark task set."
	}
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = fmt.Sprintf("%d", g)
	}
	return fmt.Sprintf(`Execute only these benchmark groups: %s.
Do not attempt groups outside this subset.
Treat all other groups as out of scope for this run rather than as failures.
For the selected groups, execute every step in the group unless blocked.`, strings.Join(parts, ", "))
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func BenchmarkRunGroupFile(dir string, group int) string {
	return filepath.Join(dir, fmt.Sprintf("group-%02d.md", group))
}

func BenchmarkRunAllGroups(dir string) []int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var groups []int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if m := groupFileRegex.FindStringSubmatch(e.Name()); m != nil {
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			groups = append(groups, n)
		}
	}
	return groups
}

func BenchmarkRunSelectedText(cfg PromptConfig, groups []int) string {
	indexPath := filepath.Join(cfg.BenchRunDir, "index.md")
	selected := groups
	if len(selected) == 0 {
		selected = BenchmarkRunAllGroups(cfg.BenchRunDir)
	}
	chunks := []string{readText(indexPath)}
	for _, g := range selected {
		path := BenchmarkRunGroupFile(cfg.BenchRunDir, g)
		if content := readText(path); content != "" {
			chunks = append(chunks, content)
		}
	}
	return strings.Join(chunks, "\n\n")
}

func LaneTaskSourceInstructions(lane Lane, cfg PromptConfig, groups []int) string {
	indexPath := filepath.Join(cfg.BenchRunDir, "index.md")

	if len(groups) == 0 {
		return fmt.Sprintf("Use the benchmark run index at %s plus all group files in %s.\n\n%s",
			indexPath, cfg.BenchRunDir, BenchmarkRunSelectedText(cfg, nil))
	}

	if lane == LanePinchtab {
		subset := BenchmarkRunSelectedText(cfg, groups)
		return fmt.Sprintf("Use only this selected task subset from %s:\n\n%s", cfg.BenchRunDir, subset)
	}

	var sections []string
	for _, g := range groups {
		if g == 0 {
			sections = append(sections, agentBrowserGroup0())
			continue
		}
		path := BenchmarkRunGroupFile(cfg.BenchRunDir, g)
		if content := readText(path); content != "" {
			sections = append(sections, content)
		}
	}

	return fmt.Sprintf("Use only this selected task subset for agent-browser: Group 0 from %s, Groups 1+ from %s.\n\n%s\n\n%s",
		cfg.SetupABFile, cfg.BenchRunDir, readText(indexPath), strings.Join(sections, "\n\n"))
}

func agentBrowserGroup0() string {
	return `## Group 0: Setup & Diagnosis (agent-browser lane)

### 0.1 Open fixtures home
Run ` + "`./scripts/ab open http://fixtures/`" + `.

**Verify**: Open succeeds and the home page loads.

### 0.2 Snapshot interactive refs
Run ` + "`./scripts/ab snapshot -i -c`" + ` on the home page.

**Verify**: Interactive refs are returned without error.

### 0.3 Session persists across commands
Use the same wrapper session across multiple commands and confirm browser state persists.

**Verify**: A follow-up action can use the existing page/session state.`
}

func LaneUserPrompt(lane Lane, cfg PromptConfig, reportFile string, groups []int) string {
	lc := LanePromptConfig(lane, cfg)
	firstGroup := 0
	if len(groups) > 0 {
		firstGroup = groups[0]
	}

	var bootstrapLines []string
	for i, cmd := range lc.BootstrapCommands {
		bootstrapLines = append(bootstrapLines, fmt.Sprintf("%d. %s", i+1, cmd))
	}
	bootstrap := strings.Join(bootstrapLines, "\n")

	taskInstr := LaneTaskSourceInstructions(lane, cfg, groups)
	taskInstr = strings.ReplaceAll(taskInstr, "\n", "\n  ")

	subsetInstr := LaneSubsetInstructions(groups)
	subsetInstr = strings.ReplaceAll(subsetInstr, "\n", "\n  - ")

	return fmt.Sprintf(`Work in this repo: %s

Benchmark lane: %s execution.

Requirements:
- Follow a linear execution flow: skill once, selected groups, execute, record, verify, continue.
- Do not read README, browse directories, or inspect unrelated files unless a command path is missing.
- %s
- Tool wrapper:
  - %s
- Workflow summary:
  - %s
- Adapter notes:
  - %s
- Task scope:
  %s
- For each completed step:
  1. record the observed answer with:
  - ./scripts/record-step.sh --report-file %s --type %s <group> <step> answer "<what you saw>" "notes"
  2. immediately verify it with:
  - ./scripts/verify-step.sh --report-file %s --type %s <group> <step> <pass|fail|skip> "verification notes"
- If a step cannot be completed, record fail or skip in the same report.
- Do not leave answered steps pending verification.
- Keep commands concise. Prefer rg/sed/cat only when you must inspect a specific file.
- After the skill step, begin actual benchmark execution immediately.
- Start with this bootstrap command sequence before attempting the selected steps:
%s
- After the bootstrap, immediately execute Group %d step 1.
- Subset selection:
  - %s
- Finish when all selected steps are executed or when you are blocked.

Your final answer should briefly summarize completion status and the main blockers, if any.`,
		cfg.RepoRoot,
		lc.Label,
		lc.SkillInstruction,
		lc.Wrapper,
		strings.Join(lc.WorkflowSummary, "\n  - "),
		strings.Join(lc.AdapterNotes, "\n  - "),
		taskInstr,
		reportFile, lc.RecordType,
		reportFile, lc.RecordType,
		bootstrap,
		firstGroup,
		subsetInstr)
}
