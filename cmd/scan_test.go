package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valerioTomassi/todototum/internal/todo"
)

func TestScanCommand_NoTODOs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"scan", "--path", "."})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan command failed: %v", err)
	}
	// not asserting output content, just execution path
}

// --- out-dir tests (merged from scan_outdir_test.go) ---

func writeSampleFile(t *testing.T, dir string) {
	t.Helper()
	content := []byte("package main\n// TODO: a\nfunc main(){}\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestScan_Command_JSON_WithOutDir_RelativeFilename(t *testing.T) {
	tmp := t.TempDir()
	writeSampleFile(t, tmp)

	outDir := filepath.Join(tmp, "reports")
	jsonName := "report.json"

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--out", jsonName, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan json with out-dir failed: %v", err)
	}

	target := filepath.Join(outDir, jsonName)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected json at %s: %v", target, err)
	}
}

func TestScan_Command_JSON_WithOutDir_AbsoluteFilename_IgnoresOutDir(t *testing.T) {
	tmp := t.TempDir()
	writeSampleFile(t, tmp)

	outDir := filepath.Join(tmp, "reports2")
	absJSON := filepath.Join(tmp, "abs.json")

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--out", absJSON, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan json abs path failed: %v", err)
	}

	if _, err := os.Stat(absJSON); err != nil {
		t.Fatalf("expected json at absolute path %s: %v", absJSON, err)
	}
	// Ensure outDir wasn't created implicitly
	if _, err := os.Stat(outDir); err == nil {
		t.Fatalf("out-dir %s should not be created when absolute json path is used", outDir)
	}
}

func TestScan_Command_JSON_OutDirCreatedIfMissing(t *testing.T) {
	tmp := t.TempDir()
	writeSampleFile(t, tmp)

	outDir := filepath.Join(tmp, "nested", "reports")
	jsonName := "r2.json"

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--out", jsonName, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan json create out-dir failed: %v", err)
	}

	if _, err := os.Stat(outDir); err != nil {
		t.Fatalf("expected out-dir created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, jsonName)); err != nil {
		t.Fatalf("expected report in created out-dir: %v", err)
	}
}

// --- helper/utility tests (merged from helpers_test.go) ---

func captureStdout(t *testing.T, fn func()) string {
	// uses io import
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestBuildIgnoreList_Variants(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{"vendor,.git,node_modules", []string{"vendor", ".git", "node_modules"}},
		{" vendor , , .idea ", []string{"vendor", ".idea"}},
	}
	for _, c := range cases {
		got := buildIgnoreList(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("len mismatch for %q: got %d want %d (%v)", c.in, len(got), len(c.want), got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("elem %d mismatch for %q: got %q want %q (full: %v)", i, c.in, got[i], c.want[i], got)
			}
		}
	}
}

func TestResolveOutputPath_Variants(t *testing.T) {
	// absolute should ignore outDir
	abs := filepath.Join(os.TempDir(), "x.html")
	if p := resolveOutputPath(abs, "reports"); p != abs {
		t.Fatalf("absolute path should be returned as-is: %q", p)
	}
	// relative + outDir
	rel := "report.json"
	od := filepath.Join(os.TempDir(), "odir")
	if p := resolveOutputPath(rel, od); p != filepath.Join(od, rel) {
		t.Fatalf("relative should join outDir: got %q", p)
	}
	// empty filename
	if p := resolveOutputPath("", od); p != "" {
		t.Fatalf("empty filename should return empty, got %q", p)
	}
}

func TestEnsureParentDir_Creates(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "nested", "deep", "file.out")
	if err := ensureParentDir(target); err != nil {
		t.Fatalf("ensureParentDir error: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Fatalf("parent dir not created: %v", err)
	}
}

func TestPrintSummary_OutputAndOrder(t *testing.T) {
	items := []todo.Todo{
		{File: "a.go", Line: 1, Tag: "FIXME", Text: "x"},
		{File: "b.go", Line: 2, Tag: "TODO", Text: "y"},
		{File: "c.go", Line: 3, Tag: "BUG", Text: "z"},
		{File: "d.go", Line: 4, Tag: "NOTE", Text: "n"},
	}
	out := captureStdout(t, func() { printSummary(items) })
	if !strings.Contains(out, "Total: 4") {
		t.Fatalf("missing total in summary: %s", out)
	}
	// Order should be alphabetical by tag
	bugIdx := strings.Index(out, "BUG:")
	fixIdx := strings.Index(out, "FIXME:")
	notIdx := strings.Index(out, "NOTE:")
	todoIdx := strings.Index(out, "TODO:")
	if bugIdx >= fixIdx || fixIdx >= notIdx || notIdx >= todoIdx {
		t.Fatalf("unexpected tag order in summary: %s", out)
	}
}

func TestRenderTable_Basic(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "table.txt")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = f.Close() }()

	items := []todo.Todo{{File: "x.go", Line: 42, Tag: "TODO", Text: "do it"}}
	renderTable(f, items)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(data)
	for _, must := range []string{"FILE", "LINE", "TAG", "TEXT", "x.go", "42", "TODO", "TODO: do it"} {
		if !strings.Contains(out, must) {
			t.Fatalf("table output missing %q in:\n%s", must, out)
		}
	}
}
