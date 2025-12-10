package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/valerioTomassi/todototum/internal/todo"
)

var (
	path   string
	report string
	out    string
	ignore string
	outDir string
	serve  bool
)

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringVarP(&path, "path", "p", ".", "Directory path to scan")
	scanCmd.Flags().StringVar(&report, "report", "table", "Output format: one of table, html, json, md")
	scanCmd.Flags().StringVar(&out, "out", "", "Output filename when --report is html|json|md; defaults: report.html/report.json/report.md. Use with --out-dir to control directory")
	scanCmd.Flags().StringVar(&ignore, "ignore", "", "Comma-separated list of directory names to skip")
	scanCmd.Flags().StringVar(&outDir, "out-dir", "", "Directory where report is written when using --report html/json/md; if file path is relative it will be placed inside this directory")
	scanCmd.Flags().BoolVar(&serve, "serve", false, "Generate an HTML report and open it in your default browser (ignores --report value)")
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a directory for TODO, FIXME, BUG, NOTE comments",
	Long:  `Recursively searches a folder for common task markers inside code comments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure flags don't leak between test runs/executions by resetting Changed at exit.
		defer func() {
			if f := cmd.Flags().Lookup("report"); f != nil {
				f.Changed = false
				_ = f.Value.Set("table")
			}
			if f := cmd.Flags().Lookup("out"); f != nil {
				f.Changed = false
				_ = f.Value.Set("")
			}
			if f := cmd.Flags().Lookup("out-dir"); f != nil {
				f.Changed = false
				_ = f.Value.Set("")
			}
			if f := cmd.Flags().Lookup("serve"); f != nil {
				f.Changed = false
				_ = f.Value.Set("false")
			}
		}()

		// Read flag values at runtime
		p, _ := cmd.Flags().GetString("path")
		i, _ := cmd.Flags().GetString("ignore")
		r, _ := cmd.Flags().GetString("report")
		outName, _ := cmd.Flags().GetString("out")
		od, _ := cmd.Flags().GetString("out-dir")
		serveFlag, _ := cmd.Flags().GetBool("serve")

		r = strings.ToLower(strings.TrimSpace(r))
		if serveFlag {
			// --serve forces HTML generation and browser open regardless of --report
			r = "html"
		}
		switch r {
		case "", "table":
			// default
			r = "table"
		case "html", "json", "md":
			// ok
		default:
			return errors.New("invalid --report value; must be one of: table, html, json, md")
		}

		ignoreList := buildIgnoreList(i)

		items, err := todo.ScanDir(p, ignoreList)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No TODOs found.")
			return nil
		}

		if r == "table" {
			// print to terminal as a table then a short summary.
			renderTable(os.Stdout, items)
			printSummary(items)
			return nil
		}

		// For file-based reports, choose default output filename when not provided
		if strings.TrimSpace(outName) == "" {
			switch r {
			case "html":
				outName = "report.html"
			case "json":
				outName = "report.json"
			case "md":
				outName = "report.md"
			}
		}
		outPath := resolveOutputPath(outName, od)
		if err := ensureParentDir(outPath); err != nil {
			return err
		}

		switch r {
		case "html":
			if err := todo.GenerateHTMLReport(items, outPath); err != nil {
				return err
			}
			fmt.Printf("HTML report written to %s\n", outPath)
			if serveFlag {
				if err := browserOpen(outPath); err != nil {
					return fmt.Errorf("failed to open browser: %w", err)
				}
				fmt.Println("Opened in your default browser.")
			}
		case "json":
			if err := todo.GenerateJSONReport(items, outPath); err != nil {
				return err
			}
			fmt.Printf("JSON report written to %s\n", outPath)
		case "md":
			if err := todo.GenerateMarkdownReport(items, outPath); err != nil {
				return err
			}
			fmt.Printf("Markdown report written to %s\n", outPath)
		}
		return nil
	},
}

// browserOpen is a package-level function variable to allow tests to stub the opener.
var browserOpen = openInBrowser

// openInBrowser attempts to open the given file path in the user's default browser
// without blocking. It uses platform-native commands.
func openInBrowser(path string) error {
	// Ensure absolute path for better OS handling
	abs := path
	if !filepath.IsAbs(abs) {
		if a, err := filepath.Abs(abs); err == nil {
			abs = a
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", abs).Start()
	case "linux":
		return exec.Command("xdg-open", abs).Start()
	case "windows":
		// Using rundll32 for compatibility
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", abs).Start()
	default:
		// Fallback: try xdg-open, then open
		if err := exec.Command("xdg-open", abs).Start(); err == nil {
			return nil
		}
		return exec.Command("open", abs).Start()
	}
}

// buildIgnoreList parses a comma-separated ignore string into a slice, trimming spaces.
func buildIgnoreList(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// renderTable writes the TODO items as a table to the provided writer.
func renderTable(w *os.File, items []todo.Todo) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"File", "Line", "Tag", "Text"})
	for _, t := range items {
		coloredTag := t.Tag
		switch strings.ToUpper(t.Tag) {
		case "TODO":
			coloredTag = color.New(color.FgYellow).Sprint(t.Tag)
		case "FIXME":
			coloredTag = color.New(color.FgRed).Sprint(t.Tag)
		case "BUG":
			coloredTag = color.New(color.FgHiRed).Sprint(t.Tag)
		case "NOTE":
			coloredTag = color.New(color.FgCyan).Sprint(t.Tag)
		}
		// Include the tag within the text column for clearer context
		text := t.Tag
		if strings.TrimSpace(t.Text) != "" {
			text = t.Tag + ": " + t.Text
		}
		table.Append([]string{t.File, fmt.Sprintf("%d", t.Line), coloredTag, text})
	}
	table.Render()
}

// resolveOutputPath determines the final output path based on the provided
// filename and optional outDir. If filename is absolute, outDir is ignored.
// If filename is relative and outDir is provided, the two are joined.
func resolveOutputPath(filename, outDir string) string {
	if filename == "" {
		return filename
	}
	if filepath.IsAbs(filename) || outDir == "" {
		return filename
	}
	return filepath.Join(outDir, filename)
}

// ensureParentDir makes sure the directory for the given file path exists.
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// printSummary prints a simple summary of counts by tag.
func printSummary(items []todo.Todo) {
	counts := make(map[string]int)
	for _, t := range items {
		counts[strings.ToUpper(t.Tag)]++
	}
	fmt.Println()
	fmt.Println(color.New(color.FgGreen, color.Bold).Sprint("Summary:"))
	fmt.Printf("  Total: %d\n", len(items))
	// Stable order for readability in tests and humans
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, tag := range keys {
		fmt.Printf("  %s: %d\n", tag, counts[tag])
	}
}
