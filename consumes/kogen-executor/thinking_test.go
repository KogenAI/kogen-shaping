package consumes

import (
	"reflect"
	"testing"

	"github.com/KogenAI/kogen-executor/thinking"
)

var _ func([]any) []any = thinking.Strip

// TestThinkingStrip pins thinking.Strip as shape/window.go uses it on the kept items of a
// reset conversation: thinking and redacted_thinking blocks leave provider-owned messages,
// everything else stays and the input is not mutated.
func TestThinkingStrip(t *testing.T) {
	text := map[string]any{"type": "text", "text": "answer"}
	toolUse := map[string]any{"type": "tool_use", "id": "t1", "name": "read_file"}
	mixed := map[string]any{
		"role": "assistant", "content": "plain mixed",
		"anthropic_message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hm", "signature": "sig"},
			text,
			map[string]any{"type": "redacted_thinking", "data": "xx"},
			toolUse,
		}},
	}
	onlyThinking := map[string]any{
		"role": "assistant", "content": "plain only",
		"anthropic_message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hm", "signature": "sig"},
		}},
	}
	user := map[string]any{"role": "user", "content": "hello"}
	call := map[string]any{"type": "function_call", "name": "shell"}
	input := []any{mixed, onlyThinking, user, call, "not a map", nil}

	got := thinking.Strip(input)
	if len(got) != len(input) {
		t.Fatalf("thinking.Strip: %d items out of %d in", len(got), len(input))
	}

	wantMixed := map[string]any{
		"role": "assistant", "content": "plain mixed",
		"anthropic_message": map[string]any{"role": "assistant", "content": []any{text, toolUse}},
	}
	if !reflect.DeepEqual(got[0], wantMixed) {
		t.Errorf("thinking.Strip: a message with thinking, redacted_thinking, text and tool_use blocks = %v, want only the text and tool_use blocks kept: %v", got[0], wantMixed)
	}
	wantOnly := map[string]any{"role": "assistant", "content": "plain only"}
	if !reflect.DeepEqual(got[1], wantOnly) {
		t.Errorf("thinking.Strip: a message left without blocks = %v, want it to lose anthropic_message and render from its plain content: %v", got[1], wantOnly)
	}
	for i, want := range []any{user, call, "not a map", nil} {
		if !reflect.DeepEqual(got[i+2], want) {
			t.Errorf("thinking.Strip: item %d = %v, want it as it was: %v", i+2, got[i+2], want)
		}
	}

	blocks := mixed["anthropic_message"].(map[string]any)["content"].([]any)
	if len(blocks) != 4 || blocks[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("thinking.Strip: the input message was mutated: %v", blocks)
	}
	if _, ok := onlyThinking["anthropic_message"]; !ok {
		t.Errorf("thinking.Strip: the input message lost anthropic_message: %v", onlyThinking)
	}
	if empty := thinking.Strip(nil); len(empty) != 0 {
		t.Errorf("thinking.Strip: nil items gave %v, want none", empty)
	}
}
