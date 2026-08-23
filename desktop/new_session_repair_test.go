package main

import (
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/config"
)

func TestNewSessionKeepsFreshRuntimeWhenTopicRepairFails(t *testing.T) {
	home := isolateDesktopUserDirs(t)
	// The scattered test harness sets REASONIX_HOME to its existing temporary
	// home. Use a fresh child here so the test can replace the config directory
	// with a file without weakening the process-wide isolation guarantee.
	t.Setenv("REASONIX_HOME", filepath.Join(home, "blocked-desktop-config"))

	dir := config.SessionDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	path := agent.NewSessionPath(dir, "model-a")
	ctrl := controllerWithContent(t, path)
	app := NewApp()
	app.projectTreeChangedHook = func() {}
	app.setTestCtrl(ctrl, "model-a")
	tab := app.tabs["test"]
	tab.TopicID = "topic_old"
	tab.TopicTitle = "Old topic"

	// Block desktopConfigDir-backed topic-index writes without affecting the
	// session directory, which exercises the post-NewSession repair failure path.
	if err := os.MkdirAll(filepath.Dir(desktopConfigDir()), 0o755); err != nil {
		t.Fatalf("mkdir desktop config parent: %v", err)
	}
	if err := os.WriteFile(desktopConfigDir(), []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("block desktop config dir: %v", err)
	}

	if err := app.NewSession(); err != nil {
		t.Fatalf("NewSession should keep the fresh runtime even when topic repair fails: %v", err)
	}
	if got := tab.TopicID; got == "" || got == "topic_old" {
		t.Fatalf("new session topic ID = %q, want fresh ID distinct from the old topic", got)
	}
	if got := tab.TopicTitle; got != defaultTopicTitle {
		t.Fatalf("new session topic title = %q, want %q", got, defaultTopicTitle)
	}
	if got := ctrl.SessionPath(); got == "" || filepath.Clean(got) == filepath.Clean(path) {
		t.Fatalf("new session path = %q, want a fresh path distinct from %q", got, path)
	}
}
