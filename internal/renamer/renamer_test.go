package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestLiteralReplace(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "sub", "b.txt")
	writeFile(t, a, "hello foo, foo again")
	writeFile(t, b, "no match here")

	res, err := Run([]string{dir}, Literal{Search: "foo", With: "bar"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChanged != 1 || res.Replacements != 2 {
		t.Fatalf("got %+v, want 1 file changed, 2 replacements", res)
	}
	if got := readFile(t, a); got != "hello bar, bar again" {
		t.Fatalf("a.txt = %q", got)
	}
	if got := readFile(t, b); got != "no match here" {
		t.Fatalf("b.txt modified: %q", got)
	}
}

func TestSkipsGitDirAndBinary(t *testing.T) {
	dir := t.TempDir()
	inGit := filepath.Join(dir, ".git", "config")
	bin := filepath.Join(dir, "blob.bin")
	writeFile(t, inGit, "foo")
	writeFile(t, bin, "foo\x00foo")

	res, err := Run([]string{dir}, Literal{Search: "foo", With: "bar"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChanged != 0 {
		t.Fatalf("expected no changes, got %+v", res)
	}
	if got := readFile(t, inGit); got != "foo" {
		t.Fatalf(".git file modified: %q", got)
	}
	if got := readFile(t, bin); got != "foo\x00foo" {
		t.Fatalf("binary file modified: %q", got)
	}
}

func TestDryRun(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	writeFile(t, a, "foo")

	res, err := Run([]string{dir}, Literal{Search: "foo", With: "bar"}, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChanged != 1 || res.Replacements != 1 {
		t.Fatalf("got %+v", res)
	}
	if got := readFile(t, a); got != "foo" {
		t.Fatalf("dry run wrote to file: %q", got)
	}
}

func TestSingleFilePath(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	writeFile(t, a, "foo")

	res, err := Run([]string{a}, Literal{Search: "foo", With: "bar"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChanged != 1 {
		t.Fatalf("got %+v", res)
	}
	if got := readFile(t, a); got != "bar" {
		t.Fatalf("a.txt = %q", got)
	}
}

func TestPreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "run.sh")
	writeFile(t, a, "foo")
	if err := os.Chmod(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Run([]string{dir}, Literal{Search: "foo", With: "bar"}, Options{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestTypoReplacer(t *testing.T) {
	r := Typos()
	tests := []struct {
		in, want string
		count    int
	}{
		{"the cat", "the cat", 1},
		{"The cat", "The cat", 1},
		{"THE CAT", "THE CAT", 1},
		{"thee cat", "the cat", 1},
		{"you should fix this", "you should fix this", 1},
		{"i received the mail", "i received the mail", 2},
		{"tether", "tether", 0},          // no mid-word matches
		{"path/to/the/file", "path/to/the/file", 1},
		{"nothing wrong", "nothing wrong", 0},
	}
	for _, tt := range tests {
		got, n := r.Replace([]byte(tt.in))
		if string(got) != tt.want || n != tt.count {
			t.Errorf("Replace(%q) = %q, %d; want %q, %d", tt.in, got, n, tt.want, tt.count)
		}
	}
}

func TestTypoRunOnTree(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "doc.md")
	writeFile(t, a, "The plan: we should definitely ship tomorrow.")

	res, err := Run([]string{dir}, Typos(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Replacements != 4 {
		t.Fatalf("got %+v, want 4 replacements", res)
	}
	want := "The plan: we should definitely ship tomorrow."
	if got := readFile(t, a); got != want {
		t.Fatalf("doc.md = %q, want %q", got, want)
	}
}
