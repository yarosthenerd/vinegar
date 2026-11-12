package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vinegarhq/vinegar/internal/sysinfo"
)

func TestDLLOverrides(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"mimalloc.dll", "mimalloc-redirect.dll"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The exact value that broke Studio on a real machine.
	p := DLLOverrides("mimalloc-redirect=", dir)
	if p == nil {
		t.Fatal("expected a problem")
	}
	if !strings.Contains(p.Body, "mimalloc-redirect") || p.ID == "" || p.Title == "" {
		t.Errorf("unhelpful problem: %+v", p)
	}

	for _, spec := range []string{"", "mimalloc-redirect=n,b", "dxdiagn,mscoree=", "unrelated="} {
		if p := DLLOverrides(spec, dir); p != nil {
			t.Errorf("%q: unexpected problem %+v", spec, p)
		}
	}
}

func TestGPUMemory(t *testing.T) {
	const gib = 1 << 30
	total := uint64(12 * gib)

	// The situation that crashed Studio: a local LLM held 10.9 of 12 GB.
	p := GPUMemory("NVIDIA GeForce RTX 5070 Ti (0000:02:00.0)",
		sysinfo.VRAM{Free: 1 * gib, Total: total}, true)
	if p == nil {
		t.Fatal("expected a problem")
	}
	for _, want := range []string{"1.0 GiB", "12.0 GiB", "RTX 5070 Ti"} {
		if !strings.Contains(p.Body, want) {
			t.Errorf("body missing %q: %s", want, p.Body)
		}
	}

	if p := GPUMemory("x", sysinfo.VRAM{Free: MinFreeVRAM, Total: total}, true); p != nil {
		t.Error("exactly the minimum must be fine")
	}
	if p := GPUMemory("x", sysinfo.VRAM{}, false); p != nil {
		t.Error("unknown memory must never be a problem")
	}
}

func TestSize(t *testing.T) {
	for n, want := range map[uint64]string{
		0: "0 MiB", 512 << 20: "512 MiB", 1 << 30: "1.0 GiB", 3<<30 + 512<<20: "3.5 GiB",
	} {
		if got := size(n); got != want {
			t.Errorf("size(%d) = %q, want %q", n, got, want)
		}
	}
}
