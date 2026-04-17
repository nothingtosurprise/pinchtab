package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnthropicToolDefinitions(t *testing.T) {
	r := NewAnthropicRunner("key", "claude-haiku-4-5-20251001", 4096, 0, true)
	tools := r.ToolDefinitions().([]map[string]interface{})
	if len(tools) != 1 {
		t.Fatalf("got %d tools; want 1", len(tools))
	}
	if tools[0]["name"] != "run_command" {
		t.Errorf("tool name = %v; want 'run_command'", tools[0]["name"])
	}
}

func TestAnthropicInitialConversation(t *testing.T) {
	r := NewAnthropicRunner("key", "claude-haiku-4-5-20251001", 4096, 0, true)
	conv := r.InitialConversation("test prompt")
	if len(conv) != 1 {
		t.Fatalf("got %d messages; want 1", len(conv))
	}
	msg := conv[0].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("role = %v; want 'user'", msg["role"])
	}
	if msg["content"] != "test prompt" {
		t.Errorf("content = %v; want 'test prompt'", msg["content"])
	}
}

func TestAnthropicSend(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s; want POST", r.Method)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("X-Api-Key = %s; want test-key", r.Header.Get("X-Api-Key"))
		}
		if r.Header.Get("Anthropic-Version") != "2023-06-01" {
			t.Errorf("Anthropic-Version = %s; want 2023-06-01", r.Header.Get("Anthropic-Version"))
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": "Hello"},
			},
			"usage": map[string]interface{}{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		})
	}))
	defer server.Close()

	r := NewAnthropicRunner("test-key", "claude-haiku-4-5-20251001", 4096, 0.5, true)
	r.retryCfg.Sleep = func(d time.Duration) {}

	originalURL := AnthropicAPIURL
	defer func() { _ = originalURL }()

	conv := r.InitialConversation("hello")
	r.client = &http.Client{Transport: &testTransport{server.URL}}

	_, err := r.Send("system", conv)
	if err != nil {
		t.Fatal(err)
	}

	if receivedBody["model"] != "claude-haiku-4-5-20251001" {
		t.Errorf("model = %v; want claude-haiku-4-5-20251001", receivedBody["model"])
	}
	if receivedBody["max_tokens"] != float64(4096) {
		t.Errorf("max_tokens = %v; want 4096", receivedBody["max_tokens"])
	}
	if _, ok := receivedBody["cache_control"]; !ok {
		t.Error("missing cache_control")
	}

	usage := r.Usage()
	if usage.RequestCount != 1 {
		t.Errorf("RequestCount = %d; want 1", usage.RequestCount)
	}
	if usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d; want 100", usage.InputTokens)
	}
}

type testTransport struct {
	targetURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, _ := http.NewRequest(req.Method, t.targetURL, req.Body)
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return http.DefaultTransport.RoundTrip(newReq)
}

func TestAnthropicExtractToolCalls(t *testing.T) {
	r := NewAnthropicRunner("key", "model", 4096, 0, true)
	response := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type":  "tool_use",
				"id":    "toolu_123",
				"input": map[string]interface{}{"command": "ls -la", "timeout_seconds": 30.0},
			},
			map[string]interface{}{
				"type": "text",
				"text": "Let me list files",
			},
		},
	}

	calls := r.ExtractToolCalls(response, 120*time.Second)
	if len(calls) != 1 {
		t.Fatalf("got %d calls; want 1", len(calls))
	}
	if calls[0].ID != "toolu_123" {
		t.Errorf("ID = %s; want toolu_123", calls[0].ID)
	}
	if calls[0].Command != "ls -la" {
		t.Errorf("Command = %s; want 'ls -la'", calls[0].Command)
	}
	if calls[0].TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d; want 30", calls[0].TimeoutSeconds)
	}
}

func TestAnthropicExtractFinalText(t *testing.T) {
	r := NewAnthropicRunner("key", "model", 4096, 0, true)
	response := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "Hello"},
			map[string]interface{}{"type": "tool_use", "id": "123"},
			map[string]interface{}{"type": "text", "text": "World"},
		},
	}

	text := r.ExtractFinalText(response)
	if text != "Hello\nWorld" {
		t.Errorf("got %q; want 'Hello\\nWorld'", text)
	}
}

func TestAnthropicAppendToolResults(t *testing.T) {
	r := NewAnthropicRunner("key", "model", 4096, 0, true)
	conv := []interface{}{}
	response := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "tool_use", "id": "123"},
		},
	}
	results := []ToolExecutionResult{
		{ID: "123", IsError: false, Content: "output"},
	}

	conv = r.AppendToolResults(conv, response, results)
	if len(conv) != 2 {
		t.Fatalf("got %d messages; want 2", len(conv))
	}

	assistant := conv[0].(map[string]interface{})
	if assistant["role"] != "assistant" {
		t.Errorf("first message role = %v; want 'assistant'", assistant["role"])
	}

	user := conv[1].(map[string]interface{})
	if user["role"] != "user" {
		t.Errorf("second message role = %v; want 'user'", user["role"])
	}
}
