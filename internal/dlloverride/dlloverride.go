// Package dlloverride interprets Wine's WINEDLLOVERRIDES variable.
package dlloverride

import (
	"path/filepath"
	"strings"
)

// Disabled returns the names of the DLLs that spec (the value of
// WINEDLLOVERRIDES) tells Wine not to load at all, lowercased and
// without a ".dll" extension.
//
// The syntax is a semicolon separated list of "dll[,dll...]=mode",
// where an empty mode disables the listed DLLs. Entries that merely
// select between native and builtin versions are ignored.
func Disabled(spec string) (names []string) {
	for entry := range strings.SplitSeq(spec, ";") {
		dlls, mode, ok := strings.Cut(entry, "=")
		if !ok || strings.TrimSpace(mode) != "" {
			continue
		}
		for dll := range strings.SplitSeq(dlls, ",") {
			dll = strings.ToLower(strings.TrimSpace(dll))
			dll = strings.TrimSuffix(dll, ".dll")
			if dll != "" {
				names = append(names, dll)
			}
		}
	}
	return names
}

// Shadowing returns the names of the DLLs disabled by spec that exist as
// "<name>.dll" in dir. Wine reports a disabled DLL as "not found" to any
// program that imports it, which makes such a program fail to start.
func Shadowing(spec, dir string) (found []string) {
	for _, name := range Disabled(spec) {
		if exists(dir, name+".dll") {
			found = append(found, name)
		}
	}
	return found
}

// exists reports whether dir contains the named file, ignoring case
// the same way Wine does when it looks for DLLs on a case sensitive
// filesystem.
func exists(dir, name string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "*"))
	for _, m := range matches {
		if strings.EqualFold(filepath.Base(m), name) {
			return true
		}
	}
	return false
}
