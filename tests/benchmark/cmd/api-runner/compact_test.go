package main

import (
	"testing"
)

func TestCompactAnthropicNoOp(t *testing.T) {
	conv := make([]interface{}, 8)
	for i := range conv {
		conv[i] = map[string]interface{}{"role": "user", "content": "msg"}
	}
	result := CompactAnthropicConversation(conv, "summary")
	if len(result) != 8 {
		t.Errorf("got %d; want 8 (no compaction)", len(result))
	}
}

func TestCompactAnthropicTriggered(t *testing.T) {
	conv := make([]interface{}, 12)
	conv[0] = map[string]interface{}{"role": "user", "content": "initial"}
	for i := 1; i < 12; i++ {
		conv[i] = map[string]interface{}{"role": "assistant", "content": i}
	}

	result := CompactAnthropicConversation(conv, "summary text")

	if len(result) != 8 {
		t.Errorf("got %d; want 8 (head + summary + 6 recent)", len(result))
	}

	summary := result[1].(map[string]interface{})
	if summary["content"] != "summary text" {
		t.Errorf("summary content = %v", summary["content"])
	}
}

func TestCompactOpenAINoOp(t *testing.T) {
	conv := make([]interface{}, 14)
	for i := range conv {
		conv[i] = map[string]interface{}{"type": "message"}
	}
	result := CompactOpenAIConversation(conv, "summary")
	if len(result) != 14 {
		t.Errorf("got %d; want 14 (no compaction)", len(result))
	}
}

func TestCompactOpenAITriggered(t *testing.T) {
	conv := make([]interface{}, 20)
	conv[0] = map[string]interface{}{"role": "user", "content": "initial"}
	for i := 1; i < 20; i++ {
		conv[i] = map[string]interface{}{"type": "function_call", "id": i}
	}

	result := CompactOpenAIConversation(conv, "summary text")

	if len(result) != 12 {
		t.Errorf("got %d; want 12 (head + summary + 10 recent)", len(result))
	}

	summary := result[1].(map[string]interface{})
	content := summary["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("summary content length = %d; want 1", len(content))
	}
	part := content[0].(map[string]interface{})
	if part["text"] != "summary text" {
		t.Errorf("summary text = %v", part["text"])
	}
}

func TestCompactConversationRouting(t *testing.T) {
	conv := make([]interface{}, 20)
	conv[0] = map[string]interface{}{"role": "user", "content": "initial"}
	for i := 1; i < 20; i++ {
		conv[i] = map[string]interface{}{"role": "assistant"}
	}

	anthropic := CompactConversation("anthropic", conv, "summary")
	if len(anthropic) != 8 {
		t.Errorf("anthropic compaction = %d; want 8", len(anthropic))
	}

	openai := CompactConversation("openai", conv, "summary")
	if len(openai) != 12 {
		t.Errorf("openai compaction = %d; want 12", len(openai))
	}
}
