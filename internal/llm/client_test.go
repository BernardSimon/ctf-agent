package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ollama root", in: "http://localhost:11434", want: "http://localhost:11434/v1"},
		{name: "already v1", in: "http://127.0.0.1:8000/v1", want: "http://127.0.0.1:8000/v1"},
		{name: "trailing slash", in: "http://127.0.0.1:8000/", want: "http://127.0.0.1:8000/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeBaseURL(tt.in); got != tt.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestChatStreamHandlesLargeSSELine(t *testing.T) {
	want := strings.Repeat("x", 70*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", want)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "test-model", false)
	resp, err := client.ChatStream(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, nil)
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}
	if resp.Content != want {
		t.Fatalf("content length = %d, want %d", len(resp.Content), len(want))
	}
}

func TestChatStreamReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "test-model", false)
	_, err := client.ChatStream(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "LLM API error 502") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}
