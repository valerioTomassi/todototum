package todo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

type jsonNopWriteCloser struct{ io.Writer }

func (n jsonNopWriteCloser) Close() error { return nil }

type jsonMockFileWriter struct{ buf *bytes.Buffer }

func (m jsonMockFileWriter) Create(_ string) (io.WriteCloser, error) {
	return jsonNopWriteCloser{m.buf}, nil
}

type jsonBadFileWriter struct{}

func (jsonBadFileWriter) Create(_ string) (io.WriteCloser, error) {
	return nil, errors.New("create failed")
}

type jsonErrWriteCloser struct{}

func (jsonErrWriteCloser) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }
func (jsonErrWriteCloser) Close() error                { return nil }

type jsonErrOnWriteFileWriter struct{}

func (jsonErrOnWriteFileWriter) Create(_ string) (io.WriteCloser, error) {
	return jsonErrWriteCloser{}, nil
}

func TestGenerateJSONReport_WithWriter_Success(t *testing.T) {
	items := []Todo{
		{File: "b.go", Line: 10, Tag: "FIXME", Text: "second"},
		{File: "a.go", Line: 2, Tag: "TODO", Text: "first"},
		{File: "a.go", Line: 20, Tag: "BUG", Text: "third"},
	}
	var buf bytes.Buffer
	mw := jsonMockFileWriter{buf: &buf}
	if err := GenerateJSONReportWithWriter(items, "ignored.json", mw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Todos   []Todo  `json:"todos"`
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v\ncontent: %s", err, buf.String())
	}
	if got.Summary.Total != 3 {
		t.Fatalf("summary total = %d, want 3", got.Summary.Total)
	}
	if got.Summary.ByTag["TODO"] != 1 || got.Summary.ByTag["FIXME"] != 1 || got.Summary.ByTag["BUG"] != 1 {
		t.Fatalf("unexpected tag counts: %#v", got.Summary.ByTag)
	}
	// Assert order by file then line
	if len(got.Todos) != 3 {
		t.Fatalf("todos len = %d, want 3", len(got.Todos))
	}
	if got.Todos[0].File != "a.go" || got.Todos[0].Line != 2 || got.Todos[0].Tag != "TODO" || got.Todos[0].Text != "TODO: first" {
		t.Fatalf("first todo unexpected: %#v", got.Todos[0])
	}
	if got.Todos[1].File != "a.go" || got.Todos[1].Line != 20 || got.Todos[1].Tag != "BUG" || got.Todos[1].Text != "BUG: third" {
		t.Fatalf("second todo unexpected: %#v", got.Todos[1])
	}
	if got.Todos[2].File != "b.go" || got.Todos[2].Line != 10 || got.Todos[2].Tag != "FIXME" || got.Todos[2].Text != "FIXME: second" {
		t.Fatalf("third todo unexpected: %#v", got.Todos[2])
	}
}

func TestGenerateJSONReport_WithWriter_CreateError(t *testing.T) {
	items := []Todo{{File: "x.go", Line: 1, Tag: "TODO", Text: "x"}}
	if err := GenerateJSONReportWithWriter(items, "ignored.json", jsonBadFileWriter{}); err == nil {
		t.Fatal("expected error from Create")
	}
}

func TestGenerateJSONReport_WithWriter_WriteError(t *testing.T) {
	items := []Todo{{File: "x.go", Line: 1, Tag: "TODO", Text: "x"}}
	if err := GenerateJSONReportWithWriter(items, "ignored.json", jsonErrOnWriteFileWriter{}); err == nil {
		t.Fatal("expected error from writer during json.Encode")
	}
}
