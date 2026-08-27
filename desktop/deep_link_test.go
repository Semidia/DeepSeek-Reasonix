package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDeepLinkProject(t *testing.T) {
	root := `C:\work\demo`
	raw := "reasonix://topic/" + url.PathEscape("topic-42") + "?scope=project&workspace=" + url.QueryEscape(root)
	got, err := parseDeepLink(raw)
	if err != nil {
		t.Fatalf("parseDeepLink(%q): %v", raw, err)
	}
	if got.TopicID != "topic-42" {
		t.Fatalf("TopicID = %q, want topic-42", got.TopicID)
	}
	if got.Scope != "project" {
		t.Fatalf("Scope = %q, want project", got.Scope)
	}
	if got.WorkspaceRoot != root {
		t.Fatalf("WorkspaceRoot = %q, want %q", got.WorkspaceRoot, root)
	}
}

func TestParseDeepLinkGlobal(t *testing.T) {
	got, err := parseDeepLink("reasonix://topic/global-topic?scope=global")
	if err != nil {
		t.Fatalf("parseDeepLink: %v", err)
	}
	if got.TopicID != "global-topic" || got.Scope != "global" || got.WorkspaceRoot != "" {
		t.Fatalf("unexpected parse: %+v", got)
	}
}

func TestParseDeepLinkDecodesTopicID(t *testing.T) {
	got, err := parseDeepLink("reasonix://topic/" + url.PathEscape("a b+c#d") + "?scope=global")
	if err != nil {
		t.Fatalf("parseDeepLink: %v", err)
	}
	if got.TopicID != "a b+c#d" {
		t.Fatalf("TopicID = %q, want decoded id", got.TopicID)
	}
}

func TestParseDeepLinkRejects(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{"empty", "", errDeepLinkBadScheme},
		{"non-url", "not a url with spaces", errDeepLinkBadScheme},
		{"wrong scheme", "https://topic/x?scope=global", errDeepLinkBadScheme},
		{"scheme case-insensitive ok", "REASONIX://topic/x?scope=global", nil},
		{"wrong host", "reasonix://nope/x?scope=global", errDeepLinkBadHost},
		{"missing id", "reasonix://topic/?scope=global", errDeepLinkBadID},
		{"no path", "reasonix://topic?scope=global", errDeepLinkBadID},
		{"encoded slash id", "reasonix://topic/a%2Fb?scope=global", errDeepLinkBadID},
		{"bad scope", "reasonix://topic/x?scope=banana", errDeepLinkBadScope},
		{"missing scope", "reasonix://topic/x", errDeepLinkBadScope},
		{"project no workspace", "reasonix://topic/x?scope=project", errDeepLinkNoWorkspace},
		{"project empty workspace", "reasonix://topic/x?scope=project&workspace=", errDeepLinkNoWorkspace},
		{"global with workspace", "reasonix://topic/x?scope=global&workspace=C%3A%5Cfoo", errDeepLinkBadParam},
		{"unknown param", "reasonix://topic/x?scope=global&token=secret", errDeepLinkBadParam},
		{"corrupt url", "reasonix://topic/%zz?scope=global", errDeepLinkBadScheme},
		{"null byte id", "reasonix://topic/x%00y?scope=global", errDeepLinkBadID},
		{"blank id", "reasonix://topic/%20?scope=global", errDeepLinkBadID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDeepLink(tc.raw)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("parseDeepLink(%q) = %v, want success", tc.raw, err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("parseDeepLink(%q) = %v, want %v", tc.raw, err, tc.want)
			}
		})
	}
}

func TestParseDeepLinkRejectsSecretParams(t *testing.T) {
	for _, key := range []string{"session", "sessionPath", "token", "key", "body", "cookie", "auth"} {
		raw := "reasonix://topic/x?scope=global&" + key + "=secret"
		if _, err := parseDeepLink(raw); !errors.Is(err, errDeepLinkBadParam) {
			t.Fatalf("param %q: got %v, want errDeepLinkBadParam", key, err)
		}
	}
}

func TestDeepLinkValidateTarget(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		d    deepLinkTopic
		want bool
	}{
		{"global ok", deepLinkTopic{Scope: "global"}, true},
		{"empty workspace", deepLinkTopic{Scope: "project", WorkspaceRoot: ""}, false},
		{"relative workspace", deepLinkTopic{Scope: "project", WorkspaceRoot: "relative/dir"}, false},
		{"missing dir", deepLinkTopic{Scope: "project", WorkspaceRoot: filepath.Join(dir, "nope")}, false},
		{"file not dir", deepLinkTopic{Scope: "project", WorkspaceRoot: file}, false},
		{"existing dir", deepLinkTopic{Scope: "project", WorkspaceRoot: dir}, true},
		{"null byte", deepLinkTopic{Scope: "project", WorkspaceRoot: dir + string(rune(0))}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.d.validateTarget()
			if (err == nil) != tc.want {
				t.Fatalf("validateTarget() = %v, want success=%v", err, tc.want)
			}
		})
	}
}

func TestIsDeepLinkArg(t *testing.T) {
	if !isDeepLinkArg("reasonix://topic/x?scope=global") {
		t.Fatal("reasonix:// arg not detected")
	}
	if isDeepLinkArg("https://example.com") {
		t.Fatal("https arg misdetected")
	}
	if isDeepLinkArg("--safe-mode") {
		t.Fatal("switch misdetected")
	}
}
