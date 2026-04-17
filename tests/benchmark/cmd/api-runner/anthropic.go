package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const AnthropicAPIURL = "https://api.anthropic.com/v1/messages"

type AnthropicRunner struct {
	apiKey        string
	model         string
	maxTokens     int
	temperature   float64
	promptCaching bool
	usage         UsageCounters
	client        *http.Client
	retryCfg      RetryConfig
}

func NewAnthropicRunner(apiKey, model string, maxTokens int, temperature float64, promptCaching bool) *AnthropicRunner {
	return &AnthropicRunner{
		apiKey:        apiKey,
		model:         model,
		maxTokens:     maxTokens,
		temperature:   temperature,
		promptCaching: promptCaching,
		client:        &http.Client{Timeout: 5 * time.Minute},
		retryCfg:      DefaultRetryConfig(),
	}
}

func (r *AnthropicRunner) Provider() string { return "anthropic" }
func (r *AnthropicRunner) Source() string   { return "anthropic-messages" }
func (r *AnthropicRunner) Model() string    { return r.model }
func (r *AnthropicRunner) Usage() UsageCounters { return r.usage }

func (r *AnthropicRunner) ToolDefinitions() interface{} {
	return []map[string]interface{}{
		{
			"name":        "run_command",
			"description": "Run a shell command in a persistent bash session rooted at tests/benchmark. Environment changes such as cd/export persist across calls.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command":         map[string]interface{}{"type": "string"},
					"timeout_seconds": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 600},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (r *AnthropicRunner) InitialConversation(userPrompt string) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": userPrompt,
		},
	}
}

func (r *AnthropicRunner) Send(systemPrompt string, conversation []interface{}) (interface{}, error) {
	body := map[string]interface{}{
		"model":       r.model,
		"max_tokens":  r.maxTokens,
		"temperature": r.temperature,
		"system":      systemPrompt,
		"tools":       r.ToolDefinitions(),
		"messages":    conversation,
	}
	if r.promptCaching {
		body["cache_control"] = map[string]string{"type": "ephemeral"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var result map[string]interface{}
	resp, err := DoWithRetry(context.Background(), r.retryCfg, func() (*http.Response, error) {
		req, err := http.NewRequest("POST", AnthropicAPIURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", r.apiKey)
		req.Header.Set("Anthropic-Version", "2023-06-01")
		return r.client.Do(req)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	r.updateUsage(result)
	return result, nil
}

func (r *AnthropicRunner) updateUsage(result map[string]interface{}) {
	usage, _ := result["usage"].(map[string]interface{})
	if usage == nil {
		return
	}
	r.usage.RequestCount++
	r.usage.InputTokens += toInt(usage["input_tokens"])
	r.usage.OutputTokens += toInt(usage["output_tokens"])
	r.usage.CacheCreationInputTokens += toInt(usage["cache_creation_input_tokens"])
	r.usage.CacheReadInputTokens += toInt(usage["cache_read_input_tokens"])
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func (r *AnthropicRunner) ExtractToolCalls(response interface{}, defaultTimeout time.Duration) []ToolCall {
	resp, _ := response.(map[string]interface{})
	content, _ := resp["content"].([]interface{})
	var calls []ToolCall
	for _, item := range content {
		m, _ := item.(map[string]interface{})
		if m["type"] != "tool_use" {
			continue
		}
		input, _ := m["input"].(map[string]interface{})
		cmd, _ := input["command"].(string)
		if cmd == "" {
			continue
		}
		timeout := int(defaultTimeout.Seconds())
		if t := toInt(input["timeout_seconds"]); t > 0 {
			timeout = t
		}
		calls = append(calls, ToolCall{
			ID:             fmt.Sprintf("%v", m["id"]),
			Command:        cmd,
			TimeoutSeconds: timeout,
		})
	}
	return calls
}

func (r *AnthropicRunner) AppendToolResults(conversation []interface{}, response interface{}, results []ToolExecutionResult) []interface{} {
	resp, _ := response.(map[string]interface{})
	conversation = append(conversation, map[string]interface{}{
		"role":    "assistant",
		"content": resp["content"],
	})
	if len(results) > 0 {
		toolResults := make([]interface{}, len(results))
		for i, res := range results {
			toolResults[i] = map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": res.ID,
				"is_error":    res.IsError,
				"content":     res.Content,
			}
		}
		conversation = append(conversation, map[string]interface{}{
			"role":    "user",
			"content": toolResults,
		})
	}
	return conversation
}

func (r *AnthropicRunner) ExtractFinalText(response interface{}) string {
	resp, _ := response.(map[string]interface{})
	content, _ := resp["content"].([]interface{})
	var texts []string
	for _, item := range content {
		m, _ := item.(map[string]interface{})
		if m["type"] != "text" {
			continue
		}
		if text, ok := m["text"].(string); ok && text != "" {
			texts = append(texts, text)
		}
	}
	return joinStrings(texts, "\n")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
