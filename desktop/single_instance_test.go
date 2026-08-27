package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestSingleInstanceLockRestoresExistingInstance(t *testing.T) {
	t.Setenv("REASONIX_HOME", t.TempDir())
	app := NewApp()
	lock := singleInstanceLock(app)

	if lock == nil {
		t.Fatal("singleInstanceLock returned nil")
	}
	id := singleInstanceID()
	if lock.UniqueId != id {
		t.Fatalf("UniqueId = %q, want %q", lock.UniqueId, id)
	}
	if !strings.HasPrefix(lock.UniqueId, singleInstanceIDPrefix+".") {
		t.Fatalf("UniqueId = %q, want prefix %s.", lock.UniqueId, singleInstanceIDPrefix)
	}
	if lock.OnSecondInstanceLaunch == nil {
		t.Fatal("OnSecondInstanceLaunch should restore the existing window")
	}

	lock.OnSecondInstanceLaunch(options.SecondInstanceData{})
}

// The lock now keys on the executable path so OS protocol invocations (which
// do not inherit REASONIX_HOME) still route back to the running instance.
func TestSingleInstanceIDScopesToExecutable(t *testing.T) {
	first := filepath.Join(t.TempDir(), "reasonix-desktop.exe")
	second := filepath.Join(t.TempDir(), "reasonix-desktop.exe")
	if firstID, secondID := singleInstanceIDForPath(first), singleInstanceIDForPath(second); firstID == secondID {
		t.Fatalf("different executables produced the same id %q", firstID)
	}
	// Same path reached through a redundant "." component canonicalizes equal.
	dotted := filepath.Join(first, "..", "reasonix-desktop.exe")
	if got := singleInstanceIDForPath(dotted); got != singleInstanceIDForPath(first) {
		t.Fatalf("canonically equal paths produced different ids: %q != %q", got, singleInstanceIDForPath(first))
	}
}

func TestSingleInstanceIDDoesNotSplitReleaseChannels(t *testing.T) {
	t.Setenv("REASONIX_HOME", t.TempDir())
	oldChannel := channel
	t.Cleanup(func() { channel = oldChannel })
	channel = "stable"
	stableID := singleInstanceID()
	channel = "canary"
	if got := singleInstanceID(); got != stableID {
		t.Fatalf("same executable split by channel: stable=%q canary=%q", stableID, got)
	}
}

// A symlink alias of the same executable canonicalizes to one lock key, so a
// launch through a junctioned install path still routes to the running binary.
func TestSingleInstanceIDResolvesAliasThroughSymlink(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "reasonix-desktop.exe")
	if err := os.WriteFile(real, []byte("placeholder"), 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.exe")
	if err := os.Symlink(real, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	realID := singleInstanceIDForPath(real)
	if got := singleInstanceIDForPath(alias); got != realID {
		t.Fatalf("aliased executable produced different ids: %q != %q", got, realID)
	}
}

func TestSingleInstanceLockSkipsInDevMode(t *testing.T) {
	t.Setenv("REASONIX_DEV", "1")
	if lock := singleInstanceLock(NewApp()); lock != nil {
		t.Fatalf("singleInstanceLock returned %#v, want nil in dev mode", lock)
	}
}
