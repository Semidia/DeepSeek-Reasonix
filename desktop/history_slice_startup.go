package main

import (
	"log/slog"
	"os"
	"strings"
)

func (a *App) historySliceBeforeController(tab *WorkspaceTab, sessionDir, sessionPath string, req HistorySliceRequest) HistorySlice {
	if strings.TrimSpace(sessionPath) == "" {
		if a.historySliceTabIsStartingBlank(tab) {
			return emptyHistorySlice()
		}
		return failedHistorySlice("session path unavailable before controller ready")
	}
	slice, err := a.coldHistorySlice(sessionDir, sessionPath, req)
	if err == nil {
		return slice
	}
	if a.historySliceTabIsStartingBlank(tab) && os.IsNotExist(err) {
		return emptyHistorySlice()
	}
	slog.Debug("desktop: cold history slice failed", "path", sessionPath, "err", err)
	return failedHistorySlice(err.Error())
}

func (a *App) historySliceTabIsStartingBlank(tab *WorkspaceTab) bool {
	if tab == nil {
		return false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tabs[tab.ID] == tab && !tab.removed &&
		tab.buildGeneration > 0 && !tab.Ready && strings.TrimSpace(tab.StartupErr) == ""
}
