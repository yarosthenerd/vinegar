// Package preflight detects problems that make Roblox Studio crash or fail
// to start without any explanation, so the user can be told beforehand.
package preflight

import (
	"fmt"
	"strings"

	"github.com/vinegarhq/vinegar/internal/dlloverride"
	"github.com/vinegarhq/vinegar/internal/sysinfo"
)

// MinFreeVRAM is the amount of free video memory below which Studio is
// likely to run out while opening a place, in bytes.
const MinFreeVRAM = 2 << 30

// Problem is something the user should be told about before Studio starts.
type Problem struct {
	ID    string // stable identifier, used to replace earlier notices
	Title string
	Body  string
}

// DLLOverrides reports whether the WINEDLLOVERRIDES value in spec disables
// a DLL that ships with the Studio installation in studioDir. Wine then
// reports the DLL as missing to anything importing it, which makes Studio
// exit right after being started.
func DLLOverrides(spec, studioDir string) *Problem {
	names := dlloverride.Shadowing(spec, studioDir)
	if len(names) == 0 {
		return nil
	}

	return &Problem{
		ID:    "dll-override",
		Title: "Studio may fail to start",
		Body: fmt.Sprintf("The WINEDLLOVERRIDES setting disables %s, which is part of Studio. "+
			"If Studio closes right after starting, remove it from the environment "+
			"variables in Vinegar's settings.", strings.Join(names, ", ")),
	}
}

// GPUMemory reports whether the video memory in vram, of the GPU named
// name, is too low for Studio. ok is false when the amount is unknown,
// which is never a problem.
func GPUMemory(name string, vram sysinfo.VRAM, ok bool) *Problem {
	if !ok || vram.Free >= MinFreeVRAM {
		return nil
	}

	return &Problem{
		ID:    "gpu-memory",
		Title: "Low GPU memory",
		Body: fmt.Sprintf("Only %s of %s video memory is free on %s, so Studio may crash "+
			"when opening a place. Close other programs that use the GPU, such as local AI "+
			"models, or choose another GPU in Vinegar's settings.",
			size(vram.Free), size(vram.Total), name),
	}
}

// size formats n bytes as MiB or GiB.
func size(n uint64) string {
	const mib = 1 << 20
	if n >= 1<<30 {
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	}
	return fmt.Sprintf("%d MiB", n/mib)
}
