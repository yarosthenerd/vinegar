package main

import (
	"log/slog"
	"path/filepath"
	"time"

	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/vinegarhq/vinegar/internal/dirs"
	"github.com/vinegarhq/vinegar/internal/prune"
)

// pruneStudioLogs removes old Roblox Studio log files. Studio never
// cleans these up and a single session can write hundreds of megabytes.
//
// It must only be called while no Studio is running, as is the case
// during setup.
func (b *bootstrapper) pruneStudioLogs() {
	dir := filepath.Join(dirs.AppDataPath, "Roblox", "logs")

	res, err := prune.Logs(dir, prune.DefaultLogs, time.Now())
	if err != nil {
		slog.Warn("Failed to prune Studio logs", "err", err)
	}
	if res.Files > 0 {
		slog.Info("Pruned old Studio logs",
			"files", res.Files, "freed", glib.FormatSizeForDisplay(res.Bytes))
	}
}

// pruneWebViewInstallers removes the downloaded WebView installers, as
// they are only needed until WebView is installed in the Wineprefix. They
// are fetched again if it ever has to be reinstalled.
func (b *bootstrapper) pruneWebViewInstallers() {
	res, err := prune.Glob(dirs.Cache, "webview-*.exe")
	if err != nil {
		slog.Warn("Failed to prune WebView installers", "err", err)
	}
	if res.Files > 0 {
		slog.Info("Removed WebView installers",
			"files", res.Files, "freed", glib.FormatSizeForDisplay(res.Bytes))
	}
}

// pruneLocalDXVK removes DXVK from the Studio installation, for when
// another renderer is selected.
func (b *bootstrapper) pruneLocalDXVK() {
	res, err := prune.LocalDXVK(b.dir)
	if err != nil {
		slog.Warn("Failed to remove DXVK", "err", err)
	}
	if res.Files > 0 {
		slog.Info("Removed DXVK, as it is not the selected renderer",
			"files", res.Files)
	}
}
