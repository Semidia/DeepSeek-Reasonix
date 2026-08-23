package main

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/billing"
	"reasonix/internal/config"
)

func setDesktopHomeEnv(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("REASONIX_HOME", home)
}

func TestIsolateDesktopUserDirsOverridesInheritedReasonixHome(t *testing.T) {
	inherited := t.TempDir()
	t.Setenv("REASONIX_HOME", inherited)

	isolationRoot := isolateDesktopUserDirs(t)
	got := config.ReasonixHomeDir()
	if filepath.Clean(got) != filepath.Clean(isolationRoot) {
		t.Fatalf("Reasonix home = %q, want isolated test root %q (inherited %q must not escape)", got, isolationRoot, inherited)
	}
}

func TestConnectKeyUsesConfiguredNetworkClient(t *testing.T) {
	t.Setenv("REASONIX_HOME", t.TempDir())
	isolateDesktopUserDirs(t)
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")

	cfg := config.Default()
	cfg.Network.ProxyMode = "off"
	if err := cfg.SaveTo(config.UserConfigPath()); err != nil {
		t.Fatalf("save direct-network config: %v", err)
	}

	oldFetch := connectKeyBalanceFetch
	var gotClient *http.Client
	connectKeyBalanceFetch = func(_ context.Context, client *http.Client, _, _ string) (*billing.Balance, error) {
		gotClient = client
		return nil, errors.New("stop after network client capture")
	}
	t.Cleanup(func() { connectKeyBalanceFetch = oldFetch })

	app := NewApp()
	app.ctx = context.Background()
	if _, err := app.ConnectKey("sk-test"); err == nil || !strings.Contains(err.Error(), "network client capture") {
		t.Fatalf("ConnectKey error = %v, want capture sentinel", err)
	}
	if gotClient == nil {
		t.Fatal("ConnectKey passed a nil HTTP client and ignored Reasonix network settings")
	}
	transport, ok := gotClient.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatalf("ConnectKey transport = %T proxy=%v, want direct transport for proxy_mode=off", gotClient.Transport, transportProxy(transport))
	}
}

func transportProxy(transport *http.Transport) any {
	if transport == nil {
		return nil
	}
	return transport.Proxy
}
