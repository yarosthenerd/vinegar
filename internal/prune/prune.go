// Package prune removes old data that Vinegar, Wine and Roblox Studio
// accumulate on disk without ever cleaning up after themselves.
package prune

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogOptions controls how Roblox Studio's log files are pruned.
type LogOptions struct {
	// Newest files that are never removed, so that the logs of the most
	// recent sessions are always available for bug reports, regardless
	// of how large they are.
	Protect int
	// Maximum amount of log files to keep, including the protected ones.
	MaxFiles int
	// Maximum total size of the kept log files, in bytes. Protected
	// files count towards it, but are never removed because of it.
	MaxBytes int64
	// Files modified more recently than this are assumed to be in use
	// by a running Studio and are never removed.
	MinAge time.Duration
}

// DefaultLogs is the default policy for Roblox Studio logs. A single
// session has been seen producing a log of nearly 900MB.
var DefaultLogs = LogOptions{
	Protect:  3,
	MaxFiles: 20,
	MaxBytes: 256 << 20,
	MinAge:   10 * time.Minute,
}

// Result describes what a pruning operation removed.
type Result struct {
	Files int
	Bytes int64
}

type file struct {
	name string
	size int64
	mod  time.Time
}

// Logs removes the oldest *.log files in dir, which is not searched
// recursively, until the amount and size limits in opt are met.
// A missing directory is not an error.
func Logs(dir string, opt LogOptions, now time.Time) (Result, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Result{}, nil
	}
	if err != nil {
		return Result{}, err
	}

	var files []file
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // vanished in between
		}
		files = append(files, file{e.Name(), info.Size(), info.ModTime()})
	}

	var res Result
	var errs []error
	for _, f := range plan(files, opt, now) {
		if err := os.Remove(filepath.Join(dir, f.name)); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		res.Files++
		res.Bytes += f.size
	}

	return res, errors.Join(errs...)
}

// plan selects the files to remove, oldest first.
func plan(files []file, opt LogOptions, now time.Time) (remove []file) {
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod) // newest first
	})

	var total int64
	for i, f := range files {
		total += f.size

		protected := i < opt.Protect || now.Sub(f.mod) < opt.MinAge
		if protected {
			continue
		}
		if i >= opt.MaxFiles || total > opt.MaxBytes {
			// Does not count towards the total, as it is going away.
			total -= f.size
			remove = append(remove, f)
		}
	}
	return remove
}

// Glob removes every regular file in dir whose name matches pattern
// (see [filepath.Match]), except for the names in keep.
// A missing directory is not an error.
func Glob(dir, pattern string, keep ...string) (Result, error) {
	if _, err := filepath.Match(pattern, ""); err != nil {
		return Result{}, err
	}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Result{}, nil
	}
	if err != nil {
		return Result{}, err
	}

	var res Result
	var errs []error
entries:
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		if ok, _ := filepath.Match(pattern, e.Name()); !ok {
			continue
		}
		for _, k := range keep {
			if k == e.Name() {
				continue entries
			}
		}

		var size int64
		if info, err := e.Info(); err == nil {
			size = info.Size()
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		res.Files++
		res.Bytes += size
	}

	return res, errors.Join(errs...)
}
