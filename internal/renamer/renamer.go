// Package renamer walks a directory tree and rewrites file contents in place.
// It is the engine behind the rename CLI: a literal search/replace mode and a
// typo-fixing mode share the same walker; only the Replacer differs.
package renamer

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Replacer transforms file contents, returning the new contents and the
// number of replacements made.
type Replacer interface {
	Replace(content []byte) ([]byte, int)
}

// Options control a Run.
type Options struct {
	DryRun  bool
	Verbose bool
	Out     io.Writer // progress output; defaults to io.Discard
}

// Result summarises a Run.
type Result struct {
	FilesScanned int
	FilesChanged int
	Replacements int
}

// skipDirs are directory names never descended into.
var skipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
}

// Run applies r to every text file under each path (files may be given
// directly), rewriting changed files in place unless opts.DryRun is set.
func Run(paths []string, r Replacer, opts Options) (Result, error) {
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	var res Result
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return res, err
		}
		if !info.IsDir() {
			if err := processFile(root, r, opts, &res); err != nil {
				return res, err
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			return processFile(path, r, opts, &res)
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func processFile(path string, r Replacer, opts Options, res *Result) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if isBinary(content) {
		return nil
	}
	res.FilesScanned++
	updated, n := r.Replace(content)
	if n == 0 {
		return nil
	}
	res.FilesChanged++
	res.Replacements += n
	if opts.Verbose || opts.DryRun {
		fmt.Fprintf(opts.Out, "%s: %d replacement(s)\n", path, n)
	}
	if opts.DryRun {
		return nil
	}
	return writeInPlace(path, updated)
}

// writeInPlace writes via a temp file in the same directory and renames it
// over the original (the same new-file-then-mv dance as the shell script),
// preserving the original file mode.
func writeInPlace(path string, content []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".rename-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// isBinary reports whether content looks like a binary file (NUL byte in the
// first 8KB), matching ack's notion of files worth searching.
func isBinary(content []byte) bool {
	probe := content
	if len(probe) > 8192 {
		probe = probe[:8192]
	}
	return bytes.IndexByte(probe, 0) >= 0
}

// Literal is a Replacer that swaps every occurrence of Search for With.
type Literal struct {
	Search string
	With   string
}

func (l Literal) Replace(content []byte) ([]byte, int) {
	n := bytes.Count(content, []byte(l.Search))
	if n == 0 {
		return content, 0
	}
	return bytes.ReplaceAll(content, []byte(l.Search), []byte(l.With)), n
}
