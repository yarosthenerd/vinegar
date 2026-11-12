//go:build linux

package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAMDGPUVRAM(t *testing.T) {
	dir := t.TempDir()
	write := func(name, v string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(v), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mem_info_vram_total", "8573157376\n")
	write("mem_info_vram_used", "1073741824\n")

	m, ok := Card{Driver: "amdgpu", Device: dir}.VRAM()
	if !ok || m.Total != 8573157376 || m.Free != 8573157376-1073741824 {
		t.Fatalf("got %+v, %v", m, ok)
	}

	// Used above total must not underflow.
	write("mem_info_vram_used", "9999999999999\n")
	if m, ok := (Card{Driver: "amdgpu", Device: dir}).VRAM(); !ok || m.Free != 0 {
		t.Fatalf("got %+v, %v", m, ok)
	}
}

func TestVRAMUnknown(t *testing.T) {
	for _, c := range []Card{
		{Driver: "i915", Device: t.TempDir()},
		{Driver: "amdgpu", Device: t.TempDir()}, // no sysfs files
		{},
	} {
		if m, ok := c.VRAM(); ok {
			t.Errorf("%s: unexpected %+v", c.Driver, m)
		}
	}
}
