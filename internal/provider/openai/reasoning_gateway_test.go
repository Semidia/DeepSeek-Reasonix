package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reasonix/internal/provider"
)

// Adapted from #7763: a generic gateway explicitly configured with
// thinking=enabled inherits DeepSeek's empty-key replay fallback.
func TestBuildRequestThinkingEnabledGatewayRoundTripsToolCallReasoning(t *testing.T) {
	p, err := New(provider.Config{
		Name: "custom", BaseURL: "https://gateway.example/v1", Model: "ds4-flash", APIKey: "k",
		Extra: map[string]any{"thinking": "enabled"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !provider.RequiresToolCallReasoning(p) || !provider.AllowsEmptyReasoningFallback(p) {
		t.Fatal("thinking-enabled gateway must use DeepSeek tool reasoning replay")
	}
	req := p.(*client).buildRequest(provider.Request{Messages: []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: `{"path":"main.go"}`}}},
		{Role: provider.RoleTool, ToolCallID: "c1", Name: "read_file", Content: "package main"},
		{Role: provider.RoleAssistant, ReasoningContent: "read first", ToolCalls: []provider.ToolCall{{ID: "c2", Name: "read_file", Arguments: `{"path":"go.mod"}`}}},
		{Role: provider.RoleTool, ToolCallID: "c2", Name: "read_file", Content: "module demo"},
		{Role: provider.RoleAssistant, Content: "done", ReasoningContent: "do not replay"},
	}})
	if got := req.Messages[0].ReasoningContent; got == nil || *got != "" {
		t.Fatalf("empty fallback = %v", got)
	}
	if got := req.Messages[2].ReasoningContent; got == nil || *got != "read first" {
		t.Fatalf("captured replay = %v", got)
	}
	body, err := json.Marshal(req.Messages)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "do not replay") {
		t.Fatalf("plain assistant reasoning leaked: %s", body)
	}
}

// Some OpenAI-compatible relays expose a DeepSeek thinking model without
// declaring `thinking` or `reasoning_protocol`. Once the relay has returned
// reasoning_content on a tool-call turn, the model contract still requires the
// exact field on later requests. This is the persisted-session shape used by
// third-party DeepSeek V4 gateways such as the desktop's custom providers.
func TestBuildRequestDeepSeekModelGatewayReplaysToolCallReasoningByDefault(t *testing.T) {
	p, err := New(provider.Config{
		Name: "deepseek-relay", BaseURL: "https://gateway.example/v1", Model: "deepseek-v4-flash", APIKey: "k",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !provider.RequiresToolCallReasoning(p) || !provider.AllowsEmptyReasoningFallback(p) {
		t.Fatal("an implicit DeepSeek V4 relay must preserve tool-call reasoning history")
	}
	req := p.(*client).buildRequest(provider.Request{Messages: []provider.Message{
		{Role: provider.RoleUser, Content: "inspect"},
		{Role: provider.RoleAssistant, ReasoningContent: "read first", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "c1", Name: "read_file", Content: "result"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "c2", Name: "read_file", Arguments: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "c2", Name: "read_file", Content: "result"},
	}})
	if got := req.Messages[1].ReasoningContent; got == nil || *got != "read first" {
		t.Fatalf("provider-issued reasoning_content = %v, want exact replay", got)
	}
	if got := req.Messages[3].ReasoningContent; got == nil || *got != "" {
		t.Fatalf("missing reasoning_content fallback = %v, want explicit empty string", got)
	}
	if req.Thinking != nil || req.ReasoningEffort != "" {
		t.Fatalf("relay inference must not invent official DeepSeek controls: thinking=%+v effort=%q", req.Thinking, req.ReasoningEffort)
	}
}

func TestBuildRequestExplicitOpenAIProtocolOptsOutOfDeepSeekModelReplay(t *testing.T) {
	p, err := New(provider.Config{
		Name: "openai-relay", BaseURL: "https://gateway.example/v1", Model: "deepseek-v4-flash", APIKey: "k",
		Extra: map[string]any{"reasoning_protocol": "openai"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.RequiresToolCallReasoning(p) || provider.AllowsEmptyReasoningFallback(p) {
		t.Fatal("explicit OpenAI protocol must opt out of DeepSeek replay inference")
	}
	req := p.(*client).buildRequest(provider.Request{Messages: []provider.Message{{
		Role: provider.RoleAssistant, ReasoningContent: "do not replay", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: `{}`}},
	}}})
	if got := req.Messages[0].ReasoningContent; got != nil {
		t.Fatalf("explicit OpenAI protocol replayed reasoning_content: %q", *got)
	}
}

func TestStreamDeepSeekModelGatewayAvoidsMissingReasoning400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			http.Error(w, "read request", http.StatusBadRequest)
			return
		}
		var request struct {
			Messages []struct {
				Role             string            `json:"role"`
				ReasoningContent *string           `json:"reasoning_content"`
				ToolCalls        []json.RawMessage `json:"tool_calls"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "decode request", http.StatusBadRequest)
			return
		}
		for _, message := range request.Messages {
			if message.Role == "assistant" && len(message.ToolCalls) > 0 && message.ReasoningContent == nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"message":"The reasoning_content in the thinking mode must be passed back to the API."}}`)
				return
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, err := New(provider.Config{Name: "deepseek-relay", BaseURL: srv.URL, Model: "deepseek-v4-flash", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := p.Stream(context.Background(), provider.Request{Messages: []provider.Message{
		{Role: provider.RoleUser, Content: "inspect"},
		{Role: provider.RoleAssistant, ReasoningContent: "read first", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "c1", Name: "read_file", Content: "result"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "c2", Name: "read_file", Arguments: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "c2", Name: "read_file", Content: "result"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var textOut string
	for chunk := range stream {
		if chunk.Type == provider.ChunkError {
			t.Fatalf("strict gateway rejected replay request: %v", chunk.Err)
		}
		if chunk.Type == provider.ChunkText {
			textOut += chunk.Text
		}
	}
	if textOut != "ok" {
		t.Fatalf("stream text = %q, want ok", textOut)
	}
}
