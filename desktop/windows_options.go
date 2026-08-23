package main

import "github.com/wailsapp/wails/v2/pkg/options/windows"

func desktopWindowsOptions(zoomFactor float64) *windows.Options {
	return &windows.Options{
		Theme:                windows.SystemDefault,
		ZoomFactor:           zoomFactor,
		WebviewGpuIsDisabled: windowsWebview2GPUDisabled(),
		WebviewUserDataPath:  webview2UserDataPath(),
	}
}
