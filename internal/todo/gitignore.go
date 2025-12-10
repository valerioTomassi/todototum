package todo

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// gitIgnoreRule represents a single .gitignore rule.
// This is a lightweight approximation that covers common cases used in repos:
// - comments starting with '#', blank lines ignored
// - negation with leading '!'
// - trailing '/' means directory-only rule
// - leading '/' anchors the pattern to the repository root
// - patterns without '/' match against the basename
// - patterns with '/' can match from any path segment downwards
// - globbing uses path.Match semantics with forward slashes
// It is not a full .gitignore implementation, but adequate for typical setups
// (e.g., node_modules/, vendor/, *.tmp, build/**, etc.).

type gitIgnoreRule struct {
	pattern  string
	negative bool
	anchored bool
	dirOnly  bool
	// hasSlash precomputed for performance
	hasSlash bool
}

type gitIgnore struct {
	root  string // repository root used for anchoring
	rules []gitIgnoreRule
}

// findRepoRoot returns the nearest ancestor directory that contains a .git directory.
// If none is found, it returns the input dir.
func findRepoRoot(start string) string {
	d := start
	for {
		if fi, err := os.Stat(filepath.Join(d, ".git")); err == nil && fi.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d { // reached filesystem root
			return start
		}
		d = parent
	}
}

// loadGitIgnore loads rules from a .gitignore file at base. If not present, returns nil.
func loadGitIgnore(base string) (*gitIgnore, error) {
	p := filepath.Join(base, ".gitignore")
	f, err := os.Open(p)
	if err != nil {
		return nil, nil // no .gitignore is fine
	}
	defer SafeClose(f, p)

	rules := make([]gitIgnoreRule, 0, 16)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		neg := false
		if strings.HasPrefix(line, "!") {
			neg = true
			line = strings.TrimSpace(line[1:])
			if line == "" { // a bare '!' line is ignored
				continue
			}
		}
		dirOnly := false
		if strings.HasSuffix(line, "/") {
			dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		anchored := false
		if strings.HasPrefix(line, "/") {
			anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		if line == "" {
			continue
		}
		rules = append(rules, gitIgnoreRule{
			pattern:  line,
			negative: neg,
			anchored: anchored,
			dirOnly:  dirOnly,
			hasSlash: strings.Contains(line, "/"),
		})
	}
	// ignore scanner error silently (non-critical)
	return &gitIgnore{root: base, rules: rules}, nil
}

// normalizePath converts OS-specific separators to '/' for matching.
func normalizePath(p string) string {
	return strings.ReplaceAll(p, string(os.PathSeparator), "/")
}

// match applies gitignore rules to a path relative to repo root.
// isDir indicates whether the path is a directory.
func (g *gitIgnore) match(rel string, isDir bool) bool {
	if g == nil {
		return false
	}
	rel = normalizePath(rel)
	// Track last match state to allow later rules to override earlier ones.
	matched := false
	for _, r := range g.rules {
		if r.dirOnly && !isDir {
			continue
		}
		if r.anchored {
			if matchPattern(r.pattern, rel) {
				matched = !r.negative
			}
			continue
		}
		// Unanchored
		if !r.hasSlash {
			// Match against basename
			base := path.Base(rel)
			if matchPattern(r.pattern, base) {
				matched = !r.negative
			}
			// Additionally, for directory-only patterns like "vendor"
			if isDir && (r.pattern == base) {
				matched = !r.negative
			}
			continue
		}
		// Pattern has slash but is unanchored: allow match from any segment downward.
		// We check the full rel and each suffix after a '/'.
		if matchPattern(r.pattern, rel) {
			matched = !r.negative
			continue
		}
		for i := 0; i < len(rel); i++ {
			if rel[i] == '/' && i+1 < len(rel) {
				suf := rel[i+1:]
				if matchPattern(r.pattern, suf) {
					matched = !r.negative
					break
				}
			}
		}
	}
	return matched
}

func matchPattern(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	if err != nil {
		// In case of invalid pattern, fall back to simple equality
		return pattern == name
	}
	return ok
}
