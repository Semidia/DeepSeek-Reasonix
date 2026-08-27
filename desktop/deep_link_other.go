//go:build !windows

package main

// registerDeepLinkProtocol is a no-op outside Windows: macOS uses CFBundleURLTypes
// in Info.plist and Linux uses .desktop MimeType=x-scheme-handler/reasonix, both
// declared at packaging time rather than self-registered at runtime.
func registerDeepLinkProtocol() error {
	return nil
}
