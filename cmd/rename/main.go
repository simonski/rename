// rename is a Go reimplementation of rename.sh: recursive in-place
// search/replace over the files in a directory tree, plus a -typo mode that
// fixes common misspellings (run `rename -typos` for the list).
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/simonski/rename/internal/renamer"
)

//go:embed VERSION
var embeddedVersion string

func version() string {
	return strings.TrimSpace(embeddedVersion)
}

func usage(out *os.File) {
	fmt.Fprintf(out, `rename %s — recursive in-place search and replace

Usage:
  rename [flags] <search> <replace> [path ...]   replace literal text in files
  rename -typo [flags] [path ...]                fix common typos (see -typos for the list)
  rename -typos                                  list the known typo corrections
  rename version                                 print the version

Paths default to the current directory. Directories are walked recursively;
binary files and %s are skipped.

Flags:
  -typo      typo-fixing mode
  -typos     print the typo table and exit
  -n         dry run: report what would change without writing
  -v         verbose: report each changed file
`, version(), ".git/.hg/.svn/node_modules")
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { usage(os.Stderr) }
	typoMode := fs.Bool("typo", false, "fix common typos instead of search/replace")
	listTypos := fs.Bool("typos", false, "list the known typo corrections and exit")
	dryRun := fs.Bool("n", false, "dry run: report changes without writing")
	verbose := fs.Bool("v", false, "verbose: report each changed file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()

	if *listTypos {
		for _, pair := range renamer.TypoList() {
			fmt.Printf("%-12s -> %s\n", pair[0], pair[1])
		}
		return nil
	}
	if len(rest) == 1 && rest[0] == "version" {
		fmt.Println(version())
		return nil
	}

	opts := renamer.Options{DryRun: *dryRun, Verbose: *verbose, Out: os.Stdout}

	var r renamer.Replacer
	var paths []string
	if *typoMode {
		r = renamer.Typos()
		paths = rest
	} else {
		if len(rest) < 2 {
			usage(os.Stderr)
			return fmt.Errorf("expected <search> and <replace> arguments")
		}
		if rest[0] == "" {
			return fmt.Errorf("search term must not be empty")
		}
		r = renamer.Literal{Search: rest[0], With: rest[1]}
		paths = rest[2:]
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	res, err := renamer.Run(paths, r, opts)
	if err != nil {
		return err
	}
	verb := "made"
	if *dryRun {
		verb = "would make"
	}
	fmt.Printf("%d file(s) scanned, %s %d replacement(s) in %d file(s)\n",
		res.FilesScanned, verb, res.Replacements, res.FilesChanged)
	return nil
}
