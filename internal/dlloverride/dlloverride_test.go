package dlloverride

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDisabled(t *testing.T) {
	for _, tc := range []struct {
		spec string
		want []string
	}{
		{"", nil},
		{"mimalloc-redirect=", []string{"mimalloc-redirect"}},
		{"mimalloc-redirect=;dxdiagn,winemenubuilder.exe,mscoree,mshtml=",
			[]string{"mimalloc-redirect", "dxdiagn", "winemenubuilder.exe", "mscoree", "mshtml"}},
		{"Foo.DLL = ; bar=n,b; baz=d", []string{"foo"}},
		{"native=n;noequals;=", nil},
		{"a,,b=", []string{"a", "b"}},
	} {
		if got := Disabled(tc.spec); !slices.Equal(got, tc.want) {
			t.Errorf("Disabled(%q) = %q, want %q", tc.spec, got, tc.want)
		}
	}
}

func TestShadowing(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"mimalloc.dll", "MiMalloc-Redirect.DLL", "other.dll"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := Shadowing("mimalloc-redirect=;dxdiagn,mscoree=;other=n", dir)
	if want := []string{"mimalloc-redirect"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := Shadowing("mimalloc-redirect=", filepath.Join(dir, "missing")); got != nil {
		t.Fatalf("missing directory must find nothing, got %q", got)
	}
}
