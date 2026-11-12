// Package regpatch edits individual DWORD values in the on-disk registry
// files of a Wineprefix (system.reg) without parsing and re-exporting
// the whole file.
//
// A full parse and export is not an option: the exporter shipped with
// the wine package does not preserve escape sequences in string values,
// which would silently corrupt the registry. This package instead only
// touches the lines of the one section it was asked to change.
//
// The Wineserver must not be running for the target Wineprefix, as it
// holds the registry in memory and overwrites the files when it exits.
package regpatch

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Offset between the Windows (1601) and Unix (1970) epochs, in seconds.
const epochDiff = 11644473600

// Patch ensures that the section for key (an absolute path inside the
// file, such as `System\ControlSet001\Services\winebus`, with single
// backslashes) contains each of the given DWORD values, and reports
// whether content had to be changed. Section and value names are matched
// case-insensitively, as the registry does. Everything else is preserved
// byte for byte.
//
// The values are written in sorted order so that the output is stable.
func Patch(content []byte, key string, values map[string]uint32, now time.Time) ([]byte, bool) {
	eol := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		eol = "\r\n"
	}

	header := "[" + strings.ReplaceAll(key, `\`, `\\`) + "]"
	lines := strings.Split(string(content), "\n")
	// A trailing newline produces an empty final element, which is not a line.
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	trim := func(s string) string { return strings.TrimSuffix(s, "\r") }

	// Locate the section and the range of lines that belong to it.
	start, end := -1, len(lines)
	for i, l := range lines {
		l = trim(l)
		if !strings.HasPrefix(l, "[") {
			continue
		}
		if start >= 0 {
			end = i
			break
		}
		name := l
		if j := strings.LastIndex(l, "]"); j >= 0 {
			name = l[:j+1]
		}
		if strings.EqualFold(name, header) {
			start = i
		}
	}

	names := slices.Sorted(maps.Keys(values))

	if start < 0 {
		out := lines
		if len(out) > 0 && trim(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		unix := now.Unix()
		out = append(out,
			fmt.Sprintf("%s %d", header, unix),
			fmt.Sprintf("#time=%x", uint64(unix+epochDiff)*10_000_000))
		for _, n := range names {
			out = append(out, valueLine(n, values[n]))
		}
		out = append(out, "")
		return []byte(strings.Join(out, eol)), true
	}

	// Trailing blank lines separate sections, new values go before them.
	insertAt := end
	for insertAt > start+1 && trim(lines[insertAt-1]) == "" {
		insertAt--
	}

	changed := false
	var added []string
	for _, n := range names {
		want := valueLine(n, values[n])
		prefix := strings.ToLower(`"` + n + `"=`)

		found := false
		for i := start + 1; i < end; i++ {
			if !strings.HasPrefix(strings.ToLower(trim(lines[i])), prefix) {
				continue
			}
			found = true
			if trim(lines[i]) != want {
				lines[i] = want
				changed = true
			}
			break
		}
		if !found {
			added = append(added, want)
			changed = true
		}
	}

	if !changed {
		return content, false
	}

	out := make([]string, 0, len(lines)+len(added))
	out = append(out, lines[:insertAt]...)
	out = append(out, added...)
	out = append(out, lines[insertAt:]...)
	return []byte(strings.Join(out, eol) + eol), true
}

// PatchFile applies [Patch] to the registry file at path, replacing it
// atomically. It reports whether the file was changed. A missing file is
// not an error and is left alone, as there is nothing to patch yet.
func PatchFile(path, key string, values map[string]uint32) (bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Refuse to touch anything that does not look like a Wine registry file.
	if !bytes.HasPrefix(content, []byte("WINE REGISTRY Version 2")) {
		return false, fmt.Errorf("%s: not a Wine registry file", path)
	}

	out, changed := Patch(content, key, values, time.Now())
	if !changed {
		return false, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".vinegar-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}

	return true, os.Rename(tmp.Name(), path)
}

func valueLine(name string, v uint32) string {
	return fmt.Sprintf(`"%s"=dword:%08x`, name, v)
}
