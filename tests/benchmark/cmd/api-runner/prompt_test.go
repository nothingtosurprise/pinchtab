package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemPromptGolden(t *testing.T) {
	golden, err := os.ReadFile("testdata/golden/system-prompt.txt")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got := SystemPrompt()
	want := strings.TrimSpace(string(golden))
	if got != want {
		t.Errorf("SystemPrompt mismatch:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestLaneSubsetInstructionsFull(t *testing.T) {
	got := LaneSubsetInstructions(nil)
	want := "Execute the full benchmark task set."
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestLaneSubsetInstructionsSubset(t *testing.T) {
	got := LaneSubsetInstructions([]int{0, 1, 2})
	if !strings.Contains(got, "Execute only these benchmark groups: 0, 1, 2.") {
		t.Errorf("missing group list in %q", got)
	}
	if !strings.Contains(got, "Do not attempt groups outside this subset.") {
		t.Errorf("missing subset constraint in %q", got)
	}
}

func TestLanePromptConfigAgent(t *testing.T) {
	cfg := PromptConfig{
		RepoRoot:    "/repo",
		BenchDir:    "/repo/tests/benchmark",
		BenchRunDir: "/repo/tests/benchmark/benchmark-run",
		SetupPTFile: "/repo/tests/benchmark/setup-pinchtab.md",
		SetupABFile: "/repo/tests/benchmark/setup-agent-browser.md",
	}
	lc := LanePromptConfig(LanePinchtab, cfg)
	if lc.Label != "PinchTab agent" {
		t.Errorf("Label = %q; want 'PinchTab agent'", lc.Label)
	}
	if lc.Wrapper != "./scripts/pt" {
		t.Errorf("Wrapper = %q; want './scripts/pt'", lc.Wrapper)
	}
}

func TestLanePromptConfigAgentBrowser(t *testing.T) {
	cfg := PromptConfig{
		RepoRoot:    "/repo",
		BenchDir:    "/repo/tests/benchmark",
		BenchRunDir: "/repo/tests/benchmark/benchmark-run",
		SetupPTFile: "/repo/tests/benchmark/setup-pinchtab.md",
		SetupABFile: "/repo/tests/benchmark/setup-agent-browser.md",
	}
	lc := LanePromptConfig(LaneAgentBrowser, cfg)
	if lc.Label != "agent-browser" {
		t.Errorf("Label = %q; want 'agent-browser'", lc.Label)
	}
	if lc.Wrapper != "./scripts/ab" {
		t.Errorf("Wrapper = %q; want './scripts/ab'", lc.Wrapper)
	}
}

func TestLaneUserPromptStructure(t *testing.T) {
	cfg := DefaultPromptConfig()
	reportFile := cfg.BenchDir + "/results/test_report.json"

	prompt := LaneUserPrompt(LanePinchtab, cfg, reportFile, nil)

	checks := []string{
		"Work in this repo:",
		"Benchmark lane: PinchTab agent execution.",
		"./scripts/pt",
		"record-step.sh",
		"verify-step.sh",
		"bootstrap command sequence",
	}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("prompt missing %q", check)
		}
	}
}

func TestBenchmarkRunGroupFile(t *testing.T) {
	got := BenchmarkRunGroupFile("/dir", 5)
	want := "/dir/group-05.md"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestBenchmarkRunAllGroups(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"group-00.md", "group-01.md", "group-10.md", "index.md"} {
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644)
	}
	groups := BenchmarkRunAllGroups(dir)
	if len(groups) != 3 {
		t.Errorf("got %d groups; want 3", len(groups))
	}
}

func TestAgentBrowserGroup0Content(t *testing.T) {
	content := agentBrowserGroup0()
	if !strings.Contains(content, "Group 0: Setup & Diagnosis (agent-browser lane)") {
		t.Error("missing group 0 header")
	}
	if !strings.Contains(content, "./scripts/ab open http://fixtures/") {
		t.Error("missing open command")
	}
}
