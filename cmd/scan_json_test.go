package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScan_Command_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	// Create a file with a couple of TODO-like comments
	content := []byte("package main\n// TODO: implement feature\n// NOTE: soon\nfunc main(){}\n")
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out := filepath.Join(tmp, "report.json")

	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--out", out})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scan json failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading json: %v", err)
	}
	var parsed struct {
		Summary struct {
			Total int `json:"total"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v\ncontent: %s", err, string(data))
	}
	if parsed.Summary.Total == 0 {
		t.Fatalf("expected non-zero total in json summary")
	}
}

func TestScan_Command_InvalidReportValue(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("// TODO: x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "csv"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error on invalid --report value")
	}
}

func TestScan_Command_JSON_DefaultOutUsesOutDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("// TODO: x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	outDir := filepath.Join(tmp, "out")
	rootCmd.SetArgs([]string{"scan", "--path", tmp, "--report", "json", "--out-dir", outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected success with default json out when --out omitted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "report.json")); err != nil {
		t.Fatalf("expected default report.json under out-dir: %v", err)
	}
}
