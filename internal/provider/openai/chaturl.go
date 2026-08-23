package openai

import (
	"net/url"
	"strings"
)

// canonicalKnownVendorChatURL rewrites official-vendor bases whose documented
// form differs from the OpenAI-compatible shape (Token Rhythm, StepFun step_plan).
func canonicalKnownVendorChatURL(raw string) (string, bool) {
	if canonical, ok := canonicalTokenRhythmChatURL(raw); ok {
		return canonical, true
	}
	return canonicalStepFunPlanChatURL(raw)
}

func resolveOpenAIChatURL(baseURL string, extra map[string]any) string {
	requestURL, _ := extra["request_url"].(string)
	requestURL = strings.TrimSpace(requestURL)
	if canonical, ok := canonicalKnownVendorChatURL(requestURL); ok {
		return canonical
	}
	if requestURL != "" {
		return normalizeOpenAIRequestURL(requestURL)
	}
	legacyChatURL, _ := extra["chat_url"].(string)
	return normalizeChatURL(baseURL, legacyChatURL)
}

func normalizeChatURL(baseURL, chatURL string) string {
	if legacy := trimURLPathTrailingSlashes(chatURL); legacy != "" {
		if canonical, ok := canonicalKnownVendorChatURL(legacy); ok {
			return canonical
		}
		return normalizeOpenAIRequestURL(legacy)
	}
	if canonical, ok := canonicalKnownVendorChatURL(baseURL); ok {
		return canonical
	}
	return appendOpenAIChatCompletions(baseURL)
}

func normalizeOpenAIRequestURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	path := strings.TrimRight(parsed.Path, "/")
	switch path {
	case "":
		parsed.Path = "/chat/completions"
	case "/v1":
		parsed.Path = "/v1/chat/completions"
	default:
		return raw
	}
	return parsed.String()
}

func trimURLPathTrailingSlashes(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func appendOpenAIChatCompletions(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(raw, "/") + "/chat/completions"
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	parsed.Path = basePath + "/chat/completions"
	return parsed.String()
}
