package todo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- helper mocks ---

type mockFileReader struct {
	files map[string]string
}

func (m mockFileReader) Open(name string) (io.ReadCloser, error) {
	if content, ok := m.files[name]; ok {
		return io.NopCloser(strings.NewReader(content)), nil
	}
	if content, ok := m.files[filepath.Base(name)]; ok {
		return io.NopCloser(strings.NewReader(content)), nil
	}
	return nil, os.ErrNotExist
}

// --- tests ---

func TestScanFileWithReader_OpenError_OSReader(t *testing.T) {
	if _, err := scanFileWithReader("/definitely/not/here.go", OSFileReader{}); err == nil {
		t.Fatal("expected error opening missing file")
	}
}

func TestScanDir_WalkDirError(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Chmod(tmp, 0000); err != nil {
		t.Skip("chmod unsupported")
	}
	defer func() {
		if err := os.Chmod(tmp, 0755); err != nil {
			t.Logf("cleanup chmod failed: %v", err)
		}
	}()
	if _, err := ScanDir(tmp, nil); err == nil {
		t.Log("expected permission error handled gracefully")
	}
}

func TestScanDirWithReader_FindsAndIgnores(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "vendor"), 0755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "vendor/ignore.go"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("write vendor/ignore.go: %v", err)
	}

	mock := mockFileReader{
		files: map[string]string{
			"main.go":   "// TODO: refactor\n// NOTE: perf",
			"ignore.go": "// FIXME: skip me",
		},
	}

	todos, err := ScanDirWithReader(tmp, []string{"vendor"}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(todos); got != 2 {
		t.Errorf("expected 2 todos, got %d", got)
	}

	tags := []string{"TODO", "NOTE"}
	for _, tag := range tags {
		found := false
		for _, td := range todos {
			if td.Tag == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %s", tag)
		}
	}
}

func TestScanFileWithReader_OpenError(t *testing.T) {
	mock := mockFileReader{files: map[string]string{}}
	if _, err := scanFileWithReader("nope.go", mock); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPattern_MatchesTagsAndText(t *testing.T) {
	cases := []struct {
		line string
		tag  string
		text string
	}{
		{"// TODO: implement", "TODO", "implement"},
		{"//FIXME: refactor", "FIXME", "refactor"},
		{"# Bug: missing check", "BUG", "missing check"},
		{"-- note: clarify", "NOTE", "clarify"},
		{"//todo no colon", "TODO", "no colon"},
		{"//random", "", ""},
	}

	for _, c := range cases {
		m := pattern.FindStringSubmatch(c.line)
		if c.tag == "" {
			if m != nil {
				t.Errorf("expected no match for %q", c.line)
			}
			continue
		}
		if m == nil {
			t.Errorf("no match for %q", c.line)
			continue
		}
		if gotTag := strings.ToUpper(m[1]); gotTag != c.tag {
			t.Errorf("got tag %q, want %q", gotTag, c.tag)
		}
		if gotText := strings.TrimSpace(m[2]); gotText != c.text {
			t.Errorf("got text %q, want %q", gotText, c.text)
		}
	}
}

func TestPattern_IsCaseInsensitive(t *testing.T) {
	lines := []string{
		"// todo: one",
		"// FixMe: two",
		"# bug: three",
		"-- note: four",
	}
	for _, l := range lines {
		if !pattern.MatchString(l) {
			t.Errorf("pattern should match case-insensitive line: %q", l)
		}
	}
}

// --- gitignore support tests (merged from scan_gitignore_test.go) ---

// helper to write file with dirs created
func mustWriteFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdirs for %s: %v", p, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func makeGitRepo(t *testing.T, root string, gitignore string) {
	t.Helper()
	// create a .git directory to mark repo root
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if gitignore != "" {
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			t.Fatalf("write .gitignore: %v", err)
		}
	}
}

func TestScanDir_RespectsGitIgnore_DirectoryRule(t *testing.T) {
	root := t.TempDir()
	makeGitRepo(t, root, "node_modules/\n")
	// ignored
	mustWriteFile(t, root, "node_modules/lib/a.go", "package x\n// TODO: ignored\n")
	// included
	mustWriteFile(t, root, "src/b.go", "package x\n// TODO: included\n")

	items, err := ScanDir(root, nil)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (dir ignored), got %d: %#v", len(items), items)
	}
	if items[0].File != filepath.Join("src", "b.go") {
		t.Fatalf("unexpected file: %s", items[0].File)
	}
}

func TestScanDir_RespectsGitIgnore_FilePattern(t *testing.T) {
	root := t.TempDir()
	makeGitRepo(t, root, "*.gen.go\n")
	mustWriteFile(t, root, "x.gen.go", "package main\n// FIXME: should be ignored\n")
	mustWriteFile(t, root, "x.go", "package main\n// FIXME: should be kept\n")

	items, err := ScanDir(root, nil)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (file pattern ignored), got %d: %#v", len(items), items)
	}
	if items[0].File != "x.go" {
		t.Fatalf("unexpected file: %s", items[0].File)
	}
}

func TestScanDir_RespectsGitIgnore_Negation(t *testing.T) {
	root := t.TempDir()
	makeGitRepo(t, root, "*.tmp\n!keep.tmp\n")
	mustWriteFile(t, root, "keep.tmp", "hello\n# TODO: keep this\n")
	mustWriteFile(t, root, "drop.tmp", "hello\n# TODO: drop this\n")

	items, err := ScanDir(root, nil)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected only negated file to be included, got %d: %#v", len(items), items)
	}
	if items[0].File != "keep.tmp" {
		t.Fatalf("unexpected file: %s", items[0].File)
	}
}

func TestScanDir_SkipsDotGitDirRegardlessOfGitignore(t *testing.T) {
	root := t.TempDir()
	// do NOT write a .gitignore; only create .git with a file that would match
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	mustWriteFile(t, root, ".git/config", "# TODO: should not be scanned\n")
	mustWriteFile(t, root, "src/main.go", "package main\n// TODO: should be scanned\n")

	items, err := ScanDir(root, nil)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	// Expect only the src/main.go TODO, not the one inside .git
	if len(items) != 1 {
		t.Fatalf("expected 1 item (skip .git), got %d: %#v", len(items), items)
	}
	if items[0].File != filepath.Join("src", "main.go") {
		t.Fatalf("unexpected file: %s", items[0].File)
	}
}

// --- extra gitignore utility tests (merged from gitignore_extra_test.go) ---

func TestMatchPattern_InvalidFallsBackToEquality(t *testing.T) {
	if !matchPattern("[", "[") { // invalid glob should degrade to equality
		t.Fatalf("expected invalid pattern to match identical name")
	}
	if matchPattern("[", "x") {
		t.Fatalf("invalid pattern should not match different name")
	}
}

func TestNormalizePath_OSSeparatorToSlashes(t *testing.T) {
	in := "dir" + string(os.PathSeparator) + "sub" + string(os.PathSeparator) + "file.txt"
	got := normalizePath(in)
	if got != "dir/sub/file.txt" {
		t.Fatalf("normalizePath: got %q want %q", got, "dir/sub/file.txt")
	}
}

func TestFindRepoRoot_ClimbsToGitDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdirs: %v", err)
	}
	// create .git at root/a
	gitAt := filepath.Join(root, "a", ".git")
	if err := os.MkdirAll(gitAt, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	// starting in a/b should return a
	got := findRepoRoot(sub)
	want := filepath.Join(root, "a")
	if got != want {
		t.Fatalf("findRepoRoot: got %q want %q", got, want)
	}
}
