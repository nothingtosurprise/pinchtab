package actions

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("tab", "", "")
	cmd.Flags().String("threshold", "", "")
	cmd.Flags().Bool("explain", false, "")
	cmd.Flags().Bool("ref-only", false, "")
	cmd.Flags().Bool("json", false, "")
	return cmd
}

func TestFind_NegativeQueryForwarded(t *testing.T) {
	m := newMockServer()
	m.response = `{"best_ref":"e1","matches":[{"ref":"e1","role":"button","name":"Cancel"}]}`
	defer m.close()

	cmd := newFindCmd()
	_ = cmd.Flags().Set("ref-only", "true")
	Find(m.server.Client(), m.base(), "", "button not submit", cmd)

	if m.lastPath != "/find" {
		t.Errorf("expected /find, got %s", m.lastPath)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(m.lastBody), &body)
	if body["query"] != "button not submit" {
		t.Errorf("expected query passthrough, got %v", body["query"])
	}
}

func TestFind_VisualQueryWithTabRoute(t *testing.T) {
	m := newMockServer()
	m.response = `{"best_ref":"e2","matches":[{"ref":"e2","role":"button","name":"Action"}]}`
	defer m.close()

	cmd := newFindCmd()
	_ = cmd.Flags().Set("tab", "tab1")
	_ = cmd.Flags().Set("ref-only", "true")
	Find(m.server.Client(), m.base(), "", "bottom button", cmd)

	if m.lastPath != "/tabs/tab1/find" {
		t.Errorf("expected /tabs/tab1/find, got %s", m.lastPath)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(m.lastBody), &body)
	if body["query"] != "bottom button" {
		t.Errorf("expected visual query passthrough, got %v", body["query"])
	}
}
