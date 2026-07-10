# rename

Recursive in-place search and replace across a directory tree — a Go
reimplementation of `rename.sh` — with an extra mode that fixes common typos.

## Install

```bash
brew install simonski/tap/rename
```

Or from source:

```bash
make install
```

## Usage

```bash
# Replace literal text in every file under the current directory
rename oldname newname

# Restrict to specific paths (files or directories)
rename oldname newname src/ docs/README.md

# Fix common misspellings of the, should, definitely, receive, ... (see: rename -typos)
rename -typo
rename -typo docs/

# See what would change without writing anything
rename -n oldname newname
rename -typo -n

# List the built-in typo corrections
rename -typos
```

Directories are walked recursively. Binary files and `.git`, `.hg`, `.svn`
and `node_modules` directories are skipped. Typo matching is on word
boundaries and case-insensitive, and corrections preserve the case shape of
the match: lowercase stays lowercase, a leading capital is kept, and
all-caps stays all-caps.

## Development

```bash
make build     # build ./bin/rename
make test      # run tests
make lint      # golangci-lint + gosec
make release   # bump version, publish tarballs + formula to simonski/homebrew-tap
```
