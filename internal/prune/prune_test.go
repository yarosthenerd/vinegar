package prune

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)

// mk creates a file of the given size, last modified age ago.
func mk(t *testing.T, dir, name string, size int, age time.Duration) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := now.Add(-age)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	slices.Sort(out)
	return out
}

func TestLogsCountLimit(t *testing.T) {
	dir := t.TempDir()
	for i := range 6 {
		mk(t, dir, string(rune('a'+i))+".log", 10, time.Duration(i+1)*time.Hour)
	}
	// a is newest ... f is oldest
	res, err := Logs(dir, LogOptions{Protect: 1, MaxFiles: 3, MaxBytes: 1 << 30, MinAge: time.Minute}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(t, dir), []string{"a.log", "b.log", "c.log"}; !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
	if res.Files != 3 || res.Bytes != 30 {
		t.Fatalf("result %+v", res)
	}
}

func TestLogsSizeLimitKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	mk(t, dir, "new.log", 40, 1*time.Hour)
	mk(t, dir, "mid.log", 40, 2*time.Hour)
	mk(t, dir, "old.log", 40, 3*time.Hour)

	if _, err := Logs(dir, LogOptions{Protect: 1, MaxFiles: 100, MaxBytes: 90, MinAge: time.Minute}, now); err != nil {
		t.Fatal(err)
	}
	// 40+40 fits, adding the third (120) does not.
	if got, want := names(t, dir), []string{"mid.log", "new.log"}; !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
}

func TestLogsProtectedSurviveSizeLimit(t *testing.T) {
	dir := t.TempDir()
	mk(t, dir, "huge.log", 1000, 1*time.Hour) // far above the limit
	mk(t, dir, "old.log", 5, 2*time.Hour)

	if _, err := Logs(dir, LogOptions{Protect: 1, MaxFiles: 100, MaxBytes: 100, MinAge: time.Minute}, now); err != nil {
		t.Fatal(err)
	}
	// The huge newest log is protected and is never removed, but the
	// older file no longer fits next to it.
	if got, want := names(t, dir), []string{"huge.log"}; !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
}

func TestLogsRecentFilesAreInUse(t *testing.T) {
	dir := t.TempDir()
	mk(t, dir, "running.log", 1000, 30*time.Second)
	mk(t, dir, "a.log", 10, 1*time.Hour)
	mk(t, dir, "b.log", 10, 2*time.Hour)

	if _, err := Logs(dir, LogOptions{Protect: 0, MaxFiles: 0, MaxBytes: 0, MinAge: 10 * time.Minute}, now); err != nil {
		t.Fatal(err)
	}
	if got, want := names(t, dir), []string{"running.log"}; !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
}

func TestLogsIgnoresOtherEntries(t *testing.T) {
	dir := t.TempDir()
	mk(t, dir, "keep.txt", 10, 100*time.Hour)
	mk(t, dir, "crash.dmp", 10, 100*time.Hour)
	if err := os.Mkdir(filepath.Join(dir, "crashes.log"), 0o755); err != nil { // a directory
		t.Fatal(err)
	}
	mk(t, dir, "x.log", 10, 100*time.Hour)

	if _, err := Logs(dir, LogOptions{MaxFiles: 0, MaxBytes: 0, MinAge: time.Minute}, now); err != nil {
		t.Fatal(err)
	}
	if got, want := names(t, dir), []string{"crash.dmp", "crashes.log", "keep.txt"}; !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
}

func TestLogsMissingDir(t *testing.T) {
	res, err := Logs(filepath.Join(t.TempDir(), "nope"), DefaultLogs, now)
	if err != nil || res != (Result{}) {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func TestLogsDefaultsKeepEverythingWhenSmall(t *testing.T) {
	dir := t.TempDir()
	for i := range 5 {
		mk(t, dir, string(rune('a'+i))+".log", 1000, time.Duration(i+1)*24*time.Hour)
	}
	res, err := Logs(dir, DefaultLogs, now)
	if err != nil || res.Files != 0 {
		t.Fatalf("a few small logs must be kept, got %+v, %v", res, err)
	}
}

func TestGlob(t *testing.T) {
	dir := t.TempDir()
	mk(t, dir, "webview-143.exe", 10, 0)
	mk(t, dir, "webview-144.exe", 20, 0)
	mk(t, dir, "webview-145.exe", 30, 0)
	mk(t, dir, "other.exe", 40, 0)
	if err := os.Mkdir(filepath.Join(dir, "webview-dir.exe"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Glob(dir, "webview-*.exe", "webview-144.exe")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"other.exe", "webview-144.exe", "webview-dir.exe"}
	if got := names(t, dir); !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
	if res.Files != 2 || res.Bytes != 40 {
		t.Fatalf("result %+v", res)
	}
}

func TestGlobBadPatternAndMissingDir(t *testing.T) {
	if _, err := Glob(t.TempDir(), "[", "x"); err == nil {
		t.Fatal("expected an error for a bad pattern")
	}
	if res, err := Glob(filepath.Join(t.TempDir(), "nope"), "*"); err != nil || res != (Result{}) {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func TestLocalDXVK(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// What a DXVK release looks like.
	write("d3d11.dll", "MZ....Game: \x00DXVK\x00v2.7\x00....")
	write("dxgi.dll", "MZ....dxvk-native....")
	write("d3d9.dll", "MZ....DXVK....")
	// The signature only exists as UTF-16 in some real DXVK DLLs.
	write("d3d8.dll", "MZ....D\x00X\x00V\x00K\x00....")
	// Named like DXVK, but is something else: must survive.
	write("d3d10core.dll", "MZ....a different library....")
	// Contains the signature, but is not one of DXVK's names: must survive.
	write("d3dcompiler_47.dll", "MZ....DXVK mentioned in passing....")
	write("RobloxStudioBeta.exe", "MZ....")

	res, err := LocalDXVK(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"RobloxStudioBeta.exe", "d3d10core.dll", "d3dcompiler_47.dll"}
	if got := names(t, dir); !slices.Equal(got, want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
	if res.Files != 4 {
		t.Fatalf("result %+v", res)
	}

	// Nothing left to remove, and a missing directory is fine.
	if res, err := LocalDXVK(dir); err != nil || res.Files != 0 {
		t.Fatalf("second run: %+v, %v", res, err)
	}
	if res, err := LocalDXVK(filepath.Join(dir, "missing")); err != nil || res.Files != 0 {
		t.Fatalf("missing dir: %+v, %v", res, err)
	}
}
