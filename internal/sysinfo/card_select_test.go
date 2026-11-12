package sysinfo

import "testing"

func TestPreferredDiscrete(t *testing.T) {
	const igpuAddr, dgpuAddr = "0000:00:02.0", "0000:02:00.0"

	// Mirrors a real hybrid laptop where the NVIDIA driver also exposes
	// an "eDP" connector, so both cards look embedded.
	igpu := Card{Device: "/sys/devices/pci0000:00/" + igpuAddr, Driver: "i915", Embedded: true, BootVGA: true}
	dgpu := Card{Device: "/sys/devices/pci0000:00/" + dgpuAddr, Driver: "nvidia", Embedded: true}
	dgpuClean := Card{Device: dgpu.Device, Driver: "nvidia"}

	for _, tc := range []struct {
		name  string
		cards []Card
		want  string // PCI address, empty for nil
	}{
		{"none", nil, ""},
		{"single integrated", []Card{igpu}, ""},
		{"single discrete", []Card{dgpuClean}, ""},
		{"hybrid, integrated first", []Card{igpu, dgpu}, dgpuAddr},
		{"hybrid, discrete first", []Card{dgpu, igpu}, dgpuAddr}, // order must not matter
		{"hybrid, discrete not marked embedded", []Card{igpu, dgpuClean}, dgpuAddr},
		{"desktop, display on the integrated card", []Card{
			{Device: "/a/" + igpuAddr, BootVGA: true}, dgpuClean,
		}, ""},
		{"no boot information, integrated first", []Card{
			{Device: igpu.Device, Embedded: true}, {Device: dgpuClean.Device},
		}, dgpuAddr},
		{"no boot information, discrete first", []Card{
			{Device: dgpuClean.Device}, {Device: igpu.Device, Embedded: true},
		}, ""},
	} {
		got := PreferredDiscrete(tc.cards)
		switch {
		case tc.want == "" && got != nil:
			t.Errorf("%s: got %s, want nil", tc.name, got.Addr())
		case tc.want != "" && (got == nil || got.Addr() != tc.want):
			t.Errorf("%s: got %v, want %s", tc.name, got, tc.want)
		}
	}
}
