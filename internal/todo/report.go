package todo

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"sort"
	"strings"
)

// Summary holds aggregate statistics.
type Summary struct {
	Total int            `json:"total"`
	ByTag map[string]int `json:"byTag"`
}

// TagStat provides a stable, presentation-friendly view of per-tag counts.
type TagStat struct {
	Tag     string  `json:"tag"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// ReportData feeds data into the HTML and JSON report templates.
type ReportData struct {
	Todos    []Todo    `json:"todos"`
	Summary  Summary   `json:"summary"`
	TagStats []TagStat `json:"tagStats"`
}

// FileWriter allows injecting file writers for testing or alternate outputs.
type FileWriter interface {
	Create(name string) (io.WriteCloser, error)
}

// OSFileWriter implements FileWriter using the real filesystem.
type OSFileWriter struct{}

// Create opens a file for writing on the local filesystem.
func (OSFileWriter) Create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

// GenerateHTMLReport writes an HTML report to the given output path using the
// default OS-backed writer. This is the production entry point.
func GenerateHTMLReport(items []Todo, output string) error {
	return GenerateHTMLReportWithWriter(items, output, OSFileWriter{})
}

// GenerateJSONReport writes a JSON report to the given output path using the
// default OS-backed writer. Suitable for CI consumption.
func GenerateJSONReport(items []Todo, output string) error {
	return GenerateJSONReportWithWriter(items, output, OSFileWriter{})
}

// Create is a top-level convenience wrapper for HTML report generation.
// It uses the real OS writer and defaults for production usage.
func Create(items []Todo, output string) error {
	return GenerateHTMLReport(items, output)
}

// buildReportData constructs Summary and returns a sorted copy of items.
func buildReportData(items []Todo) ReportData {
	counts := make(map[string]int)
	cp := make([]Todo, len(items))
	copy(cp, items)
	for i := range cp {
		// Aggregate counts by tag
		counts[cp[i].Tag]++
		// Enrich text to include the tag keyword for clearer reports
		if cp[i].Text == "" {
			cp[i].Text = cp[i].Tag
		} else {
			cp[i].Text = cp[i].Tag + ": " + cp[i].Text
		}
	}
	// Stable ordering for todos: by file, then line
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].File == cp[j].File {
			return cp[i].Line < cp[j].Line
		}
		return cp[i].File < cp[j].File
	})
	// Build TagStats in alphabetical order with percentages rounded to one decimal place.
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	total := len(cp)
	stats := make([]TagStat, 0, len(keys))
	for _, k := range keys {
		c := counts[k]
		var pct float64
		if total > 0 {
			// one decimal precision
			pct = math.Round((float64(c)*100.0/float64(total))*10) / 10
		}
		stats = append(stats, TagStat{Tag: k, Count: c, Percent: pct})
	}
	return ReportData{
		Todos:    cp,
		Summary:  Summary{Total: total, ByTag: counts},
		TagStats: stats,
	}
}

// GenerateHTMLReportWithWriter allows dependency injection of writers for testing.
func GenerateHTMLReportWithWriter(items []Todo, output string, w FileWriter) error {
	data := buildReportData(items)

	tmpl, candidates, err := parseReportTemplate()
	if err != nil {
		return fmt.Errorf("could not find report.html template in: %v", candidates)
	}

	f, err := w.Create(output)
	if err != nil {
		return err
	}
	defer SafeClose(f, output)

	return tmpl.Execute(f, data)
}

// GenerateJSONReportWithWriter allows dependency injection of writers for testing.
func GenerateJSONReportWithWriter(items []Todo, output string, w FileWriter) error {
	data := buildReportData(items)
	f, err := w.Create(output)
	if err != nil {
		return err
	}
	defer SafeClose(f, output)
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// GenerateMarkdownReport writes a Markdown report to the given output path using the
// default OS-backed writer.
func GenerateMarkdownReport(items []Todo, output string) error {
	return GenerateMarkdownReportWithWriter(items, output, OSFileWriter{})
}

// GenerateMarkdownReportWithWriter allows dependency injection of writers for testing.
func GenerateMarkdownReportWithWriter(items []Todo, output string, w FileWriter) error {
	data := buildReportData(items)
	f, err := w.Create(output)
	if err != nil {
		return err
	}
	defer SafeClose(f, output)

	var b strings.Builder
	// Title
	b.WriteString("# todototum report\n\n")
	// Summary
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- Total: %d\n", data.Summary.Total))
	// Stable list of tags using TagStats (already sorted)
	if len(data.TagStats) > 0 {
		for _, ts := range data.TagStats {
			b.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", ts.Tag, ts.Count, ts.Percent))
		}
	}
	b.WriteString("\n")
	// Todos table
	b.WriteString("## Todos\n\n")
	b.WriteString("| File | Line | Tag | Text |\n")
	b.WriteString("|------|------:|-----:|------|\n")
	for _, t := range data.Todos {
		// Text already includes the tag prefix (via buildReportData)
		b.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", t.File, t.Line, t.Tag, t.Text))
	}

	_, err = io.WriteString(f, b.String())
	return err
}

//go:embed templates/report.html
var templatesFS embed.FS

// parseReportTemplate parses the embedded HTML template.
// The template is compiled into the binary via Go's //go:embed. No filesystem
// lookup or overrides are performed.
func parseReportTemplate() (*template.Template, []string, error) {
	if tmpl, err := template.ParseFS(templatesFS, "templates/report.html"); err == nil {
		return tmpl, []string{"embedded:templates/report.html"}, nil
	}
	return nil, []string{"embedded:templates/report.html"}, fmt.Errorf("template not found")
}
