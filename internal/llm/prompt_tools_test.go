package llm

import "testing"

func TestParseToolCallsAcceptsArgumentsAlias(t *testing.T) {
	text := "```tool\n{\"name\":\"run_command\",\"arguments\":{\"command\":\"echo ok\"}}\n```"
	_, calls := ParseToolCallsFromText(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Function.Name != "run_command" {
		t.Fatalf("tool name = %q", calls[0].Function.Name)
	}
	if calls[0].Function.Arguments != `{"command":"echo ok"}` {
		t.Fatalf("arguments = %q", calls[0].Function.Arguments)
	}
}

func TestParseToolCallsAcceptsMislabeledFence(t *testing.T) {
	text := "```python\n{\"name\":\"run_command\",\"args\":{\"command\":\"echo ok\"}}\n```"
	_, calls := ParseToolCallsFromText(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Function.Name != "run_command" {
		t.Fatalf("tool name = %q", calls[0].Function.Name)
	}
}
