package todo

import (
	"bufio"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Todo represents a single annotated task found in source files.
// Fields are intentionally simple to support plain table and HTML rendering.
type Todo struct {
	File string
	Line int
	Tag  string
	Text string
}

// pattern matches TODO-like markers, case-insensitively, capturing tag and text.
var pattern = regexp.MustCompile(`(?i)\b(TODO|FIXME|BUG|NOTE)\b:?(.+)?`)

// ScanDir walks a directory tree using the real OS reader and collects todos.
func ScanDir(root string, ignoreDirs []string) ([]Todo, error) {
	return ScanDirWithReader(root, ignoreDirs, OSFileReader{})
}

// ScanDirWithReader is like ScanDir but allows injection of a custom FileReader
// for testing or alternate backends. Behavior and output are identical.
func ScanDirWithReader(root string, ignoreDirs []string, reader FileReader) ([]Todo, error) {
	// Prepare ignore set
	skip := make(map[string]bool)
	for _, d := range ignoreDirs {
		skip[strings.TrimSpace(d)] = true
	}

	// Determine repo root and load .gitignore rules if available.
	repoRoot := findRepoRoot(root)
	gi, _ := loadGitIgnore(repoRoot)

	// Bounded worker pool to scan files in parallel.
	type fileJob struct {
		rel  string
		open string
	}

	jobs := make(chan fileJob, 64)
	var todos []Todo
	var mu sync.Mutex

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				fileTodos, err := scanFileWithReader(job.open, reader)
				if err == nil && len(fileTodos) > 0 {
					for i := range fileTodos {
						fileTodos[i].File = job.rel
					}
					mu.Lock()
					todos = append(todos, fileTodos...)
					mu.Unlock()
				}
			}
		}()
	}

	// Walk directory and dispatch files to workers.
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Ignore traversal errors for individual entries; continue walking.
			return nil
		}
		if d.IsDir() {
			// Always skip VCS metadata directories
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			// Skip by explicit directory name
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			// Skip by .gitignore rules when inside a git repo
			if gi != nil {
				relRepo, _ := filepath.Rel(repoRoot, path)
				if gi.match(relRepo, true) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Normalize to relative path for nicer display and stable output.
		relPath, _ := filepath.Rel(root, path)

		// Check .gitignore rules for files
		if gi != nil {
			relRepo, _ := filepath.Rel(repoRoot, path)
			if gi.match(relRepo, false) {
				return nil
			}
		}

		// Use full path when reading real files; relative for mocks.
		openPath := relPath
		if _, ok := reader.(OSFileReader); ok {
			openPath = path
		}

		jobs <- fileJob{rel: relPath, open: openPath}
		return nil
	})

	close(jobs)
	wg.Wait()

	return todos, err
}

// scanFileWithReader scans a single file using the provided reader.
// It returns any matching TODO-like items found line by line.
func scanFileWithReader(path string, reader FileReader) ([]Todo, error) {
	f, err := reader.Open(path)
	if err != nil {
		return nil, err
	}
	defer SafeClose(f, path)

	var todos []Todo
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if m := pattern.FindStringSubmatch(line); m != nil {
			todos = append(todos, Todo{
				File: path,
				Line: lineNum,
				Tag:  strings.ToUpper(m[1]),
				Text: strings.TrimSpace(m[2]),
			})
		}
	}
	return todos, sc.Err()
}
