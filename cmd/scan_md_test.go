package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_Command_MDOutput(t *testing.T) {
	tmp := t.TempDir()
	content := []byte("package main\n// TODO: implement feature\nfunc main(){}\n")
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out := filepath.Join(tmp, "report.md")

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "md", "--out", out})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan md failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading md: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty markdown output")
	}
}

func TestScan_Command_ReportMD_DefaultOutUsesOutDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("// TODO: x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	outDir := filepath.Join(tmp, "out")
	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "md", "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected success with default md out when --out omitted: %v", err)
	}
	// Should create report.md under outDir
	if _, err := os.Stat(filepath.Join(outDir, "report.md")); err != nil {
		t.Fatalf("expected default report.md under out-dir: %v", err)
	}
}

func TestScan_Command_MD_WithOutDir_RelativeFilename(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("// TODO: y"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	outDir := filepath.Join(tmp, "reports")
	mdName := "report.md"

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "md", "--out", mdName, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan md with out-dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, mdName)); err != nil {
		t.Fatalf("expected md at %s: %v", filepath.Join(outDir, mdName), err)
	}
}

func TestScan_Command_MD_WithOutDir_AbsoluteFilename_IgnoresOutDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("// TODO: z"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	outDir := filepath.Join(tmp, "reports2")
	absMD := filepath.Join(tmp, "abs.md")

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "md", "--out", absMD, "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan md abs path failed: %v", err)
	}
	if _, err := os.Stat(absMD); err != nil {
		t.Fatalf("expected md at absolute path %s: %v", absMD, err)
	}
	if _, err := os.Stat(outDir); err == nil {
		t.Fatalf("out-dir %s should not be created when absolute md path is used", outDir)
	}
}
