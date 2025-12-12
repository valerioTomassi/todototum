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
		items := []Todo{{File: "a.go", Line: 1, Tag: "TODO", Text: "x"}, {File: "b.go", Line: 2, Tag: "FIXME", Text: "y"}}
		var buf bytes.Buffer
		writer := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", writer); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, ">Total<") && !strings.Contains(out, "Total") {
			t.Errorf("expected the word 'Total' in embedded template output, got: %s", out)
		}
		if !strings.Contains(out, ">2<") && !strings.Contains(out, "2") {
			t.Errorf("expected count 2 in output, got: %s", out)
		}
		if !strings.Contains(out, "TODO") || !strings.Contains(out, "FIXME") {
			t.Errorf("unexpected output: %s", out)
		}
	})

	t.Run("embedded template always available (no missing template error)", func(t *testing.T) {
		var buf bytes.Buffer
		writer := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(nil, "ignored.html", writer); err != nil {
			t.Fatalf("did not expect error with embedded template, got: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("expected some rendered HTML, got empty output")
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
		items := []Todo{{File: "a.go", Line: 1, Tag: "TODO", Text: "x"}}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", errOnWriteFileWriter{}); err == nil {
			t.Fatalf("expected error from writer during Execute, got nil")
		}
	})

	t.Run("sorts by file then line", func(t *testing.T) {
		items := []Todo{{File: "same.go", Line: 20, Tag: "TODO", Text: "later"}, {File: "a.go", Line: 5, Tag: "BUG", Text: "first by file"}, {File: "same.go", Line: 10, Tag: "FIXME", Text: "should come before line 20"}}
		var buf bytes.Buffer
		mw := mockFileWriter{buf: &buf}
		if err := GenerateHTMLReportWithWriter(items, "ignored.html", mw); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		// Check that rows appear in the correct order within the HTML table.
		i1 := strings.Index(out, "a.go")
		i2 := strings.Index(out, "same.go</td>\n                    <td>10")
		if i2 == -1 { // fallback: just search for first same.go occurrence
			i2 = strings.Index(out, "same.go")
		}
		i3 := strings.LastIndex(out, "same.go")
		if i1 == -1 || i2 == -1 || i3 == -1 {
			t.Fatalf("expected file names in output, got: %s", out)
		}
		if i1 >= i2 || i2 >= i3 {
			t.Fatalf("expected order a.go -> same.go(10) -> same.go(20); positions: %d, %d, %d", i1, i2, i3)
		}
	})

	t.Run("Create wrapper writes file", func(t *testing.T) {
		tmp := t.TempDir()
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

	// Summary counts and TagStats with fractional percentages
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
	if sum < 99.0 || sum > 101.0 {
		t.Fatalf("unexpected percent sum: %v", sum)
	}
}
