package regpatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const key = `System\ControlSet001\Services\winebus`

var (
	now  = time.Unix(1781508202, 0).UTC()
	vals = map[string]uint32{"DisableHidraw": 1}
)

const base = `WINE REGISTRY Version 2
;; All keys relative to REGISTRY\\Machine

#arch=win64

[Software\\Foo] 1781508202
#time=1dcfc97d8f077e2
"Quoted"="say \"hi\"\n"

[System\\ControlSet001\\Services\\winehid] 1781508202
#time=1dcfc97d901775e
"DisplayName"="Wine HID"
`

func TestPatchAddsMissingSection(t *testing.T) {
	out, changed := Patch([]byte(base), key, vals, now)
	if !changed {
		t.Fatal("expected a change")
	}
	if !strings.HasPrefix(string(out), base) {
		t.Fatal("existing content must be preserved byte for byte")
	}
	want := "\n[System\\\\ControlSet001\\\\Services\\\\winebus] 1781508202\n" +
		"#time=1dcfc97d8b0d100\n" +
		"\"DisableHidraw\"=dword:00000001\n"
	if got := string(out)[len(base):]; got != want {
		t.Fatalf("appended section mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestPatchIsIdempotent(t *testing.T) {
	once, _ := Patch([]byte(base), key, vals, now)
	twice, changed := Patch(once, key, vals, now.Add(time.Hour))
	if changed {
		t.Fatal("second patch must not report a change")
	}
	if string(once) != string(twice) {
		t.Fatal("second patch must not alter the content")
	}
}

func TestPatchAddsValueToExistingSection(t *testing.T) {
	in := base + "\n[System\\\\ControlSet001\\\\Services\\\\winebus] 1781508202\n" +
		"#time=1dcfc97d901775e\n" +
		"\"Other\"=dword:00000002\n\n" +
		"[System\\\\ControlSet001\\\\Services\\\\winemouse] 1781508202\n" +
		"\"Keep\"=dword:00000003\n"
	out, changed := Patch([]byte(in), key, vals, now)
	if !changed {
		t.Fatal("expected a change")
	}
	got := string(out)
	for _, want := range []string{
		"\"Other\"=dword:00000002\n\"DisableHidraw\"=dword:00000001\n\n[System\\\\ControlSet001\\\\Services\\\\winemouse]",
		"\"Keep\"=dword:00000003\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Count(got, "DisableHidraw") != 1 {
		t.Error("value must appear exactly once")
	}
}

func TestPatchFixesWrongValueCaseInsensitively(t *testing.T) {
	in := base + "\n[system\\\\controlset001\\\\SERVICES\\\\WineBus] 1781508202\n" +
		"\"disablehidraw\"=dword:00000000\n"
	out, changed := Patch([]byte(in), key, vals, now)
	if !changed {
		t.Fatal("expected a change")
	}
	got := string(out)
	if !strings.Contains(got, "\"DisableHidraw\"=dword:00000001\n") {
		t.Errorf("value not corrected:\n%s", got)
	}
	if strings.Contains(got, "dword:00000000") {
		t.Error("old value must be gone")
	}
	if strings.Count(got, "[") != strings.Count(in, "[") {
		t.Error("must not add a second section when one exists")
	}
}

func TestPatchPreservesCRLF(t *testing.T) {
	in := strings.ReplaceAll(base, "\n", "\r\n")
	out, _ := Patch([]byte(in), key, vals, now)
	if strings.Contains(strings.ReplaceAll(string(out), "\r\n", ""), "\n") {
		t.Fatal("bare LF introduced into a CRLF file")
	}
}

func TestPatchEmptyAndNoTrailingNewline(t *testing.T) {
	out, changed := Patch([]byte("WINE REGISTRY Version 2"), key, vals, now)
	if !changed || !strings.Contains(string(out), "\"DisableHidraw\"=dword:00000001") {
		t.Fatalf("bad output: %q", out)
	}
}

func TestPatchFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "system.reg")

	if changed, err := PatchFile(path, key, vals); changed || err != nil {
		t.Fatalf("missing file must be a no-op, got %v, %v", changed, err)
	}

	if err := os.WriteFile(path, []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := PatchFile(path, key, vals)
	if err != nil || !changed {
		t.Fatalf("expected patch, got %v, %v", changed, err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode not preserved: %v", fi.Mode())
	}
	if changed, err = PatchFile(path, key, vals); changed || err != nil {
		t.Fatalf("second run must be a no-op, got %v, %v", changed, err)
	}
	if left, _ := filepath.Glob(path + ".vinegar-*"); len(left) != 0 {
		t.Errorf("temporary files left behind: %v", left)
	}
}

func TestPatchFileRejectsOtherFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system.reg")
	const junk = "not a registry\n"
	if err := os.WriteFile(path, []byte(junk), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PatchFile(path, key, vals); err == nil {
		t.Fatal("expected an error")
	}
	if b, _ := os.ReadFile(path); string(b) != junk {
		t.Fatal("file must be untouched")
	}
}
