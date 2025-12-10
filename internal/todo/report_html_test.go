package todo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Test suite for HTML report generation consolidated here to reduce file sprawl
// and keep related scenarios in one place.

// --- test helpers ---

type nopWriteCloser struct{ io.Writer }

func (n nopWriteCloser) Close() error { return nil }

type mockFileWriter struct{ buf *bytes.Buffer }

func (m mockFileWriter) Create(_ string) (io.WriteCloser, error) { return nopWriteCloser{m.buf}, nil }

type badFileWriter struct{}

func (badFileWriter) Create(_ string) (io.WriteCloser, error) {
	return nil, fmt.Errorf("mock create failure")
}

type errWriteCloser struct{}

func (e errWriteCloser) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }
func (e errWriteCloser) Close() error                { return nil }

type errOnWriteFileWriter struct{}

func (errOnWriteFileWriter) Create(_ string) (io.WriteCloser, error) { return errWriteCloser{}, nil }

// --- tests ---

func TestReport_GenerateHTML(t *testing.T) {
	t.Run("success with writer buffer", func(t *testing.T) {
		tmp := t.TempDir()
		tmplDir := filepath.Join(tmp, "templates")
		if err := os.Mkdir(tmplDir, 0o755); err != nil {
			t.Fatal(err)
		}
		tmpl := `<html><body>Total: {{.Summary.Total}}; First: {{(index .Todos 0).Tag}}</body></html>`
		if err := os.WriteFile(filepath.Join(tmplDir, "report.html"), []byte(tmpl), 0o644); err != nil {
			t.Fatal(err)
		}
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(tmp)

		items := []Todo{{File: "a.go", Line: 1, Tag: "TODO", Text: "x"}, {File: "b.go", Line: 2, Tag: "FIXME", Text: "y"}}
		var buf bytes.Buffer
		writer := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", writer); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "Total: 2") || !strings.Contains(out, "First: TODO") {
			t.Errorf("unexpected output: %s", out)
		}
	})

	t.Run("missing template error (with writer)", func(t *testing.T) {
		tmp := t.TempDir()
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(tmp)
		var buf bytes.Buffer
		writer := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(nil, "ignored.html", writer); err == nil {
			t.Fatal("expected error for missing template")
		}
	})

	t.Run("create error is propagated", func(t *testing.T) {
		tmp := t.TempDir()
		tmplDir := filepath.Join(tmp, "templates")
		_ = os.Mkdir(tmplDir, 0o755)
		_ = os.WriteFile(filepath.Join(tmplDir, "report.html"), []byte(`{{.Summary.Total}}`), 0o644)
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(tmp)
		items := []Todo{{File: "x.go", Line: 1, Tag: "BUG", Text: "fail"}}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", badFileWriter{}); err == nil {
			t.Fatal("expected create error")
		}
	})

	t.Run("execute error from writer surfaces", func(t *testing.T) {
		// Ensure a template exists in first candidate path.
		tmp := t.TempDir()
		_ = os.Mkdir(filepath.Join(tmp, "templates"), 0o755)
		_ = os.WriteFile(filepath.Join(tmp, "templates", "report.html"), []byte(`{{.Summary.Total}}`), 0o644)
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(tmp)
		items := []Todo{{File: "a.go", Line: 1, Tag: "TODO", Text: "x"}}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", errOnWriteFileWriter{}); err == nil {
			t.Fatalf("expected error from writer during Execute, got nil")
		}
	})

	t.Run("falls back to template near executable", func(t *testing.T) {
		// Prepare a fake executable directory with a templates subfolder.
		exeDir := t.TempDir()
		if err := os.Mkdir(filepath.Join(exeDir, "templates"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		tmpl := []byte(`<html><body>Total: {{.Summary.Total}}</body></html>`)
		if err := os.WriteFile(filepath.Join(exeDir, "templates", "report.html"), tmpl, 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}
		// Place us in a different working directory so the first candidate is not found.
		cwd := t.TempDir()
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(cwd)
		// Trick the code into thinking the executable lives under exeDir.
		origArgs := os.Args
		t.Cleanup(func() { os.Args = origArgs })
		os.Args = []string{filepath.Join(exeDir, "todototum-test-bin")}
		var buf bytes.Buffer
		mw := mockFileWriter{buf: &buf}
		items := []Todo{{File: "x.go", Line: 3, Tag: "TODO", Text: "x"}}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", mw); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := buf.String(); got == "" {
			t.Fatalf("expected some rendered HTML, got empty output")
		}
	})

	t.Run("sorts by file then line", func(t *testing.T) {
		cwd := t.TempDir()
		if err := os.Mkdir(filepath.Join(cwd, "templates"), 0o755); err != nil {
			t.Fatal(err)
		}
		// Render file:line pairs to assert sorting explicitly.
		tmpl := "{{range .Todos}}{{.File}}:{{.Line}}\n{{end}}"
		if err := os.WriteFile(filepath.Join(cwd, "templates", "report.html"), []byte(tmpl), 0o644); err != nil {
			t.Fatal(err)
		}
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(cwd)
		items := []Todo{{File: "same.go", Line: 20, Tag: "TODO", Text: "later"}, {File: "a.go", Line: 5, Tag: "BUG", Text: "first by file"}, {File: "same.go", Line: 10, Tag: "FIXME", Text: "should come before line 20"}}
		var buf bytes.Buffer
		mw := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", mw); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		want := []string{"a.go:5", "same.go:10", "same.go:20"}
		if len(lines) != len(want) {
			t.Fatalf("unexpected number of lines: got %d, want %d: %v", len(lines), len(want), lines)
		}
		for i := range want {
			if lines[i] != want[i] {
				t.Fatalf("order mismatch at %d: got %q want %q; full: %v", i, lines[i], want[i], lines)
			}
		}
	})

	t.Run("Create wrapper writes file", func(t *testing.T) {
		tmp := t.TempDir()
		tmplDir := filepath.Join(tmp, "templates")
		_ = os.Mkdir(tmplDir, 0o755)
		_ = os.WriteFile(filepath.Join(tmplDir, "report.html"), []byte(`<html><body>{{range .Todos}}{{.Tag}} {{end}}</body></html>`), 0o644)
		origWD, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Chdir(tmp)
		items := []Todo{{File: "x.go", Line: 1, Tag: "NOTE", Text: "ok"}}
		out := filepath.Join(tmp, "report.html")
		if err := Create(items, out); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		data, _ := os.ReadFile(out)
		if !strings.Contains(string(data), "NOTE") {
			t.Errorf("expected NOTE in output, got: %s", string(data))
		}
	})
}

// merged from report_builddata_test.go
func TestBuildReportData_EnrichesTextAndComputesPercents(t *testing.T) {
	items := []Todo{
		{File: "b.go", Line: 10, Tag: "FIXME", Text: "fix it"},
		{File: "a.go", Line: 2, Tag: "TODO", Text: "do it"},
		{File: "a.go", Line: 1, Tag: "TODO", Text: ""}, // empty text should become just tag
	}
	data := buildReportData(items)

	// Todos must be sorted by file then line
	if len(data.Todos) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(data.Todos))
	}
	if data.Todos[0].File != "a.go" || data.Todos[0].Line != 1 {
		t.Fatalf("first todo should be a.go:1, got %s:%d", data.Todos[0].File, data.Todos[0].Line)
	}

	// Text enrichment should prefix with tag, and empty becomes just tag
	wantTexts := []string{"TODO", "TODO: do it", "FIXME: fix it"}
	gotTexts := []string{data.Todos[0].Text, data.Todos[1].Text, data.Todos[2].Text}
	if !reflect.DeepEqual(gotTexts, wantTexts) {
		t.Fatalf("unexpected texts: got %v want %v", gotTexts, wantTexts)
	}

	// Summary counts and TagStats with integer percentages
	if data.Summary.Total != 3 {
		t.Fatalf("total = %d, want 3", data.Summary.Total)
	}
	if data.Summary.ByTag["TODO"] != 2 || data.Summary.ByTag["FIXME"] != 1 {
		t.Fatalf("unexpected byTag: %#v", data.Summary.ByTag)
	}
	// TagStats sorted by tag name
	if len(data.TagStats) != 2 {
		t.Fatalf("expected 2 tag stats, got %d", len(data.TagStats))
	}
	if data.TagStats[0].Tag != "FIXME" || data.TagStats[0].Percent == 0 {
		t.Fatalf("first stat should be FIXME with percent > 0, got %#v", data.TagStats[0])
	}
	if data.TagStats[1].Tag != "TODO" {
		t.Fatalf("second stat should be TODO, got %#v", data.TagStats[1])
	}
	// Percentages should sum approx 100 after rounding
	sum := data.TagStats[0].Percent + data.TagStats[1].Percent
	if sum < 99 || sum > 101 {
		t.Fatalf("unexpected percent sum: %d", sum)
	}
}
