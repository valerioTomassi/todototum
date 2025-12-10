package todo

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// helpers specific to markdown tests

type mdNopWriteCloser struct{ io.Writer }

func (n mdNopWriteCloser) Close() error { return nil }

type mdMockFileWriter struct{ buf *bytes.Buffer }

func (m mdMockFileWriter) Create(_ string) (io.WriteCloser, error) {
	return mdNopWriteCloser{m.buf}, nil
}

type mdBadFileWriter struct{}

func (mdBadFileWriter) Create(_ string) (io.WriteCloser, error) {
	return nil, errors.New("create failed")
}

type mdErrWriteCloser struct{}

func (mdErrWriteCloser) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }
func (mdErrWriteCloser) Close() error                { return nil }

type mdErrOnWriteFileWriter struct{}

func (mdErrOnWriteFileWriter) Create(_ string) (io.WriteCloser, error) {
	return mdErrWriteCloser{}, nil
}

func TestGenerateMarkdownReport_WithWriter_Success(t *testing.T) {
	items := []Todo{
		{File: "b.go", Line: 10, Tag: "FIXME", Text: "second"},
		{File: "a.go", Line: 2, Tag: "TODO", Text: "first"},
		{File: "a.go", Line: 20, Tag: "BUG", Text: "third"},
	}
	var buf bytes.Buffer
	mw := mdMockFileWriter{buf: &buf}
	if err := GenerateMarkdownReportWithWriter(items, "ignored.md", mw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# todototum report") {
		t.Fatalf("missing title in markdown: %s", out)
	}
	if !strings.Contains(out, "Total: 3") {
		t.Fatalf("missing summary total in markdown: %s", out)
	}
	// first todo after sort should be a.go:2 with TODO: first
	if !strings.Contains(out, "| a.go | 2 | TODO | TODO: first |") {
		t.Fatalf("unexpected todos table content: %s", out)
	}
}

func TestGenerateMarkdownReport_WithWriter_CreateError(t *testing.T) {
	items := []Todo{{File: "x.go", Line: 1, Tag: "TODO", Text: "x"}}
	if err := GenerateMarkdownReportWithWriter(items, "ignored.md", mdBadFileWriter{}); err == nil {
		t.Fatal("expected error from Create")
	}
}

func TestGenerateMarkdownReport_WithWriter_WriteError(t *testing.T) {
	items := []Todo{{File: "x.go", Line: 1, Tag: "TODO", Text: "x"}}
	if err := GenerateMarkdownReportWithWriter(items, "ignored.md", mdErrOnWriteFileWriter{}); err == nil {
		t.Fatal("expected error from writer during markdown write")
	}
}
