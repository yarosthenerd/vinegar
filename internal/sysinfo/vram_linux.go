//go:build linux

package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"codeberg.org/puregotk/purego"
)

// VRAM reports the dedicated video memory of the card, if it can be
// determined. This is only known for AMD cards (through sysfs) and NVIDIA
// cards (through the driver's NVML library, which is loaded on demand).
// Integrated cards use system memory and report nothing.
func (c Card) VRAM() (VRAM, bool) {
	switch {
	case c.Driver == "amdgpu":
		return amdgpuVRAM(c.Device)
	case strings.HasPrefix(c.Driver, "nvidia"):
		return nvmlVRAM(c.Addr())
	}
	return VRAM{}, false
}

func amdgpuVRAM(device string) (VRAM, bool) {
	read := func(name string) (uint64, bool) {
		b, err := os.ReadFile(filepath.Join(device, name))
		if err != nil {
			return 0, false
		}
		v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
		return v, err == nil
	}

	total, ok1 := read("mem_info_vram_total")
	used, ok2 := read("mem_info_vram_used")
	if !ok1 || !ok2 || total == 0 {
		return VRAM{}, false
	}
	return VRAM{Free: total - min(used, total), Total: total}, true
}

// Mirrors nvmlMemory_t.
type nvmlMemoryInfo struct {
	Total, Free, Used uint64
}

func nvmlVRAM(busID string) (m VRAM, ok bool) {
	// Registering a function that does not exist panics.
	defer func() {
		if recover() != nil {
			m, ok = VRAM{}, false
		}
	}()

	lib, err := purego.Dlopen("libnvidia-ml.so.1", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return VRAM{}, false
	}
	defer purego.Dlclose(lib)

	var (
		init     func() int32
		shutdown func() int32
		byPci    func(busID string, device *uintptr) int32
		memInfo  func(device uintptr, mem unsafe.Pointer) int32
	)
	purego.RegisterLibFunc(&init, lib, "nvmlInit_v2")
	purego.RegisterLibFunc(&shutdown, lib, "nvmlShutdown")
	purego.RegisterLibFunc(&byPci, lib, "nvmlDeviceGetHandleByPciBusId_v2")
	purego.RegisterLibFunc(&memInfo, lib, "nvmlDeviceGetMemoryInfo")

	if init() != 0 { // NVML_SUCCESS
		return VRAM{}, false
	}
	defer shutdown()

	var dev uintptr
	if byPci(busID, &dev) != 0 {
		return VRAM{}, false
	}

	var mem nvmlMemoryInfo
	if memInfo(dev, unsafe.Pointer(&mem)) != 0 || mem.Total == 0 {
		return VRAM{}, false
	}
	return VRAM{Free: mem.Free, Total: mem.Total}, true
}
