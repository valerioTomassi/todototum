package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func ensureTemplateAndChdir(t *testing.T, dir string) {
	t.Helper()
	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	content := []byte("<html><body>Total: {{.Summary.Total}}</body></html>")
	if err := os.WriteFile(filepath.Join(tmplDir, "report.html"), content, 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	origWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	_ = os.Chdir(dir)
}

func writeGoWithTodo(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("package main\n// TODO: something\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	return p
}

func TestScan_Command_Serve_DefaultOut_UsesOutDir_AndOpens(t *testing.T) {
	tmp := t.TempDir()
	writeGoWithTodo(t, tmp, "main.go")
	ensureTemplateAndChdir(t, tmp)

	outDir := filepath.Join(tmp, "rep")
	called := 0
	var openedPath string
	orig := browserOpen
	t.Cleanup(func() { browserOpen = orig })
	browserOpen = func(p string) error {
		called++
		openedPath = p
		return nil
	}

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--serve", "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan --serve failed: %v", err)
	}
	// Should have written default report.html under outDir
	outPath := filepath.Join(outDir, "report.html")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected html at %s: %v", outPath, err)
	}
	if called != 1 {
		t.Fatalf("expected opener called once, got %d", called)
	}
	// openedPath should be absolute on most platforms due to abs normalization
	if openedPath == "" {
		t.Fatalf("expected opener to receive a path")
	}
}

func TestScan_Command_Serve_AbsoluteOut_IgnoresOutDir_AndOpens(t *testing.T) {
	// Windows absolute path requirements differ; skip if absolute join fails sanity
	tmp := t.TempDir()
	writeGoWithTodo(t, tmp, "main.go")
	ensureTemplateAndChdir(t, tmp)

	absOut := filepath.Join(tmp, "serve_abs.html")
	outDir := filepath.Join(tmp, "ignored")

	called := 0
	orig := browserOpen
	t.Cleanup(func() { browserOpen = orig })
	browserOpen = func(p string) error {
		called++
		// Ensure it tries to open the absolute file we requested
		if runtime.GOOS != "windows" { // windows normalization could differ
			if p != absOut {
				t.Fatalf("opener path mismatch: got %q want %q", p, absOut)
			}
		}
		return nil
	}

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--serve", "--out", absOut, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan --serve abs failed: %v", err)
	}
	if _, err := os.Stat(absOut); err != nil {
		t.Fatalf("expected html at absolute path: %v", err)
	}
	if _, err := os.Stat(outDir); err == nil {
		t.Fatalf("out-dir should not be created when absolute out is used")
	}
	if called != 1 {
		t.Fatalf("expected opener called once, got %d", called)
	}
}

func TestScan_Command_Serve_CoercesReportValue(t *testing.T) {
	tmp := t.TempDir()
	writeGoWithTodo(t, tmp, "main.go")
	ensureTemplateAndChdir(t, tmp)

	out := filepath.Join(tmp, "coerced.html")
	called := 0
	orig := browserOpen
	t.Cleanup(func() { browserOpen = orig })
	browserOpen = func(p string) error {
		called++
		return nil
	}

	// Even if report json is passed, --serve should force HTML generation and open it
	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--serve", "--out", out})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan --serve with report json failed: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected html report at %s: %v", out, err)
	}
	if called != 1 {
		t.Fatalf("expected opener called once, got %d", called)
	}
}
