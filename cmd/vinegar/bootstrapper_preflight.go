package main

import (
	"fmt"
	"log/slog"
	"strings"

	"codeberg.org/puregotk/puregotk/v4/gio"
	"github.com/vinegarhq/vinegar/internal/gutil"
	"github.com/vinegarhq/vinegar/internal/preflight"
	"github.com/vinegarhq/vinegar/internal/sysinfo"
)

// preflight tells the user about known problems that would make Studio
// crash or exit right away, which would otherwise happen without any
// explanation. It never prevents Studio from being launched.
func (b *bootstrapper) preflight() {
	var problems []*preflight.Problem

	problems = append(problems, preflight.DLLOverrides(
		b.cfg.Studio.Env["WINEDLLOVERRIDES"], b.dir))

	if c := b.selectedCard(); c != nil {
		vram, ok := c.VRAM()
		name := fmt.Sprintf("%s (%s)",
			strings.TrimSpace(c.String()), c.Addr())
		slog.Debug("Checking GPU memory", "gpu", name, "known", ok,
			"free_mib", vram.Free>>20, "total_mib", vram.Total>>20)
		problems = append(problems, preflight.GPUMemory(name, vram, ok))
	}

	for _, p := range problems {
		if p == nil {
			continue
		}
		slog.Warn(p.Title, "detail", p.Body)

		// The window is hidden while Studio runs, so a log message
		// alone would never be seen.
		p := p
		gutil.IdleAdd(func() {
			n := gio.NewNotification(p.Title)
			n.SetBody(p.Body)
			b.app.SendNotification(p.ID, n)
		})
	}
}

// selectedCard returns the GPU that Studio is going to run on, or nil
// if that is up to the system.
func (b *bootstrapper) selectedCard() *sysinfo.Card {
	if addr := b.cfg.Studio.ForcedGpu; addr != "" {
		for i := range sysinfo.Cards {
			if sysinfo.Cards[i].Addr() == addr {
				return &sysinfo.Cards[i]
			}
		}
		return nil
	}

	if len(sysinfo.Cards) == 1 {
		return &sysinfo.Cards[0]
	}
	return nil
}
