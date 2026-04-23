package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pinchtab/pinchtab/internal/cli"
	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/pinchtab/pinchtab/internal/cli/output"
	"github.com/spf13/cobra"
)

// TabList lists all open tabs.
func TabList(client *http.Client, base, token string, cmd *cobra.Command) {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoGet(client, base, token, "/tabs", nil)
		return
	}

	// Terse: one line per tab: [*]<id>\t<url>\t<title>
	body := apiclient.DoGetRaw(client, base, token, "/tabs", nil)
	var tabs []map[string]any
	if err := json.Unmarshal(body, &tabs); err != nil {
		fmt.Println(string(body))
		return
	}
	for _, tab := range tabs {
		id, _ := tab["id"].(string)
		url, _ := tab["url"].(string)
		title, _ := tab["title"].(string)
		active, _ := tab["active"].(bool)
		prefix := ""
		if active {
			prefix = "*"
		}
		fmt.Printf("%s%s\t%s\t%s\n", prefix, id, url, title)
	}
}

// TabNew opens a new tab (exported for cobra subcommand).
func TabNew(client *http.Client, base, token string, body map[string]any, cmd *cobra.Command) {
	// Check if any instances are running
	instances := getInstances(client, base, token)
	if len(instances) == 0 {
		fmt.Fprintln(os.Stderr, cli.StyleStderr(cli.WarningStyle, "No instances running, launching default..."))
		launchInstance(client, base, token, "default")
		fmt.Fprintln(os.Stderr, cli.StyleStderr(cli.SuccessStyle, "Instance launched"))
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, "/tab", body)
		return
	}

	result := apiclient.DoPostQuiet(client, base, token, "/tab", body)
	if tabID, ok := result["tabId"].(string); ok {
		output.Value(tabID)
	} else {
		output.Success()
	}
}

// TabClose closes a tab by ID.
func TabClose(client *http.Client, base, token string, tabID string, cmd *cobra.Command) {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, "/tab", map[string]any{
			"action": "close",
			"tabId":  tabID,
		})
		return
	}

	apiclient.DoPostQuiet(client, base, token, "/tab", map[string]any{
		"action": "close",
		"tabId":  tabID,
	})
	output.Success()
}

// TabFocus switches to a tab by ID, making it the active tab
// for subsequent commands.
func TabFocus(client *http.Client, base, token string, tabID string, cmd *cobra.Command) {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, "/tab", map[string]any{
			"action": "focus",
			"tabId":  tabID,
		})
		return
	}

	apiclient.DoPostQuiet(client, base, token, "/tab", map[string]any{
		"action": "focus",
		"tabId":  tabID,
	})
	output.Value(tabID)
}

// TabHandoff pauses automation on a tab for manual operator intervention.
func TabHandoff(client *http.Client, base, token, tabID, reason string, timeoutMS int) {
	body := map[string]any{}
	if reason != "" {
		body["reason"] = reason
	}
	if timeoutMS > 0 {
		body["timeoutMs"] = timeoutMS
	}
	apiclient.DoPost(client, base, token, fmt.Sprintf("/tabs/%s/handoff", tabID), body)
}

// TabResume resumes automation on a tab after manual intervention.
func TabResume(client *http.Client, base, token, tabID, status string) {
	body := map[string]any{}
	if status != "" {
		body["status"] = status
	}
	apiclient.DoPost(client, base, token, fmt.Sprintf("/tabs/%s/resume", tabID), body)
}

// TabHandoffStatus shows whether a tab is in paused_handoff or active state.
func TabHandoffStatus(client *http.Client, base, token, tabID string) {
	apiclient.DoGet(client, base, token, fmt.Sprintf("/tabs/%s/handoff", tabID), nil)
}
