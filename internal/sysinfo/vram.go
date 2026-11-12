package sysinfo

// VRAM is the amount of dedicated video memory of a [Card], in bytes.
type VRAM struct {
	Free  uint64
	Total uint64
}
