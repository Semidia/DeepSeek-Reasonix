//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"

	"reasonix/internal/installlayout"
)

const deepLinkRegistryBase = `Software\Classes\reasonix`

// registerDeepLinkProtocol wires the running executable as the handler for
// reasonix:// deep links at the current-user level. It never touches HKLM, so a
// non-elevated install can self-register without admin rights. The command line
// is `"<executable>" "%1"`: Wails' single-instance lock then forwards the URL
// to the primary process via OnSecondInstanceLaunch.
func registerDeepLinkProtocol() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	// Point the association at the stable launcher when a versioned layout is
	// active, mirroring icon/shortcut repair: the versioned binary is deleted on
	// update, which would silently break the protocol.
	target := executable
	if root, err := installlayout.ResolveInstallRoot(executable); err == nil && root != "" {
		launcher := filepath.Join(root, installlayout.LauncherBinaryName())
		if info, err := os.Lstat(launcher); err == nil && info.Mode().IsRegular() {
			target = launcher
		}
	}
	command := fmt.Sprintf(`"%s" "%%1"`, target)
	key, _, err := registry.CreateKey(registry.CURRENT_USER, deepLinkRegistryBase, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("", "URL:Reasonix Topic Protocol"); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	openKey, _, err := registry.CreateKey(registry.CURRENT_USER, deepLinkRegistryBase+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer openKey.Close()
	return openKey.SetStringValue("", command)
}
