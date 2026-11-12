//go:build !linux

package sysinfo

// VRAM is not available on this platform.
func (c Card) VRAM() (VRAM, bool) { return VRAM{}, false }
