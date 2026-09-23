package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// glob.go — pattern expansion for the CLI. ESLint resolves patterns through
// globby/minimatch; this is a small, explicit subset: `**`, `*`, `?`, literal
// paths, existing files and existing directories. Anything outside the subset
// is treated literally rather than silently mis-matched.

// defaultExtensions are the extensions ESLint lints by default (.js), with
// --ext adding more.
var defaultExtensions = []string{".js"}

// globToRegexp translates a glob pattern into an anchored regexp over
// slash-separated relative paths.
func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// "**/" matches zero or more directories.
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 2
					continue
				}
				b.WriteString(".*")
				i++
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '[':
			j := i + 1
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				b.WriteString(regexp.QuoteMeta("["))
				continue
			}
			cls := pattern[i+1 : j]
			if strings.HasPrefix(cls, "!") {
				cls = "^" + cls[1:]
			}
			b.WriteString("[" + cls + "]")
			i = j
		case '\\':
			if i+1 < len(pattern) {
				b.WriteString(regexp.QuoteMeta(string(pattern[i+1])))
				i++
				continue
			}
			b.WriteString(regexp.QuoteMeta("\\"))
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// globMatcher matches slash-separated relative paths against patterns.
type globMatcher struct {
	patterns []*regexp.Regexp
	raw      []string
}

func newGlobMatcher(patterns []string) *globMatcher {
	m := &globMatcher{}
	for _, p := range patterns {
		p = strings.TrimPrefix(filepath.ToSlash(p), "./")
		re, err := globToRegexp(p)
		if err != nil {
			// A pattern we cannot compile is kept literally so it never
			// silently matches everything.
			re = regexp.MustCompile("^" + regexp.QuoteMeta(p) + "$")
		}
		m.patterns = append(m.patterns, re)
		m.raw = append(m.raw, p)
	}
	return m
}

// matches reports whether a relative path matches any pattern. Directory
// patterns ("dir") also match everything under them.
func (m *globMatcher) matches(rel string) bool {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	for i, re := range m.patterns {
		if re.MatchString(rel) {
			return true
		}
		// "dir" is treated as "dir/**" by ESLint's ignore handling.
		p := strings.TrimSuffix(m.raw[i], "/")
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

// empty reports whether the matcher has no patterns.
func (m *globMatcher) empty() bool { return m == nil || len(m.patterns) == 0 }

// hasExtension reports whether a path has one of the given extensions.
func hasExtension(path string, exts []string) bool {
	lower := strings.ToLower(path)
	for _, e := range exts {
		if strings.HasSuffix(lower, strings.ToLower(e)) {
			return true
		}
	}
	return false
}

// collectFiles expands CLI patterns into a sorted list of files to lint,
// skipping ignored paths. It mirrors ESLint's behaviour for file paths,
// directories (walked for the configured extensions) and glob patterns.
func collectFiles(patterns []string, exts []string, ignore *globMatcher, cwd string) ([]string, error) {
	seen := map[string]bool{}
	var out []string

	add := func(path string) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if seen[abs] {
			return
		}
		rel, err := filepath.Rel(cwd, abs)
		if err != nil {
			rel = abs
		}
		if !ignore.empty() && ignore.matches(rel) {
			return
		}
		seen[abs] = true
		out = append(out, abs)
	}

	for _, pattern := range patterns {
		info, err := os.Stat(pattern)
		switch {
		case err == nil && info.IsDir():
			if err := walkDir(pattern, exts, ignore, cwd, add); err != nil {
				return nil, err
			}
		case err == nil && !info.IsDir():
			add(pattern)
		default:
			// Treat as a glob relative to cwd.
			root := globRoot(pattern)
			if err := walkDir(root, exts, ignore, cwd, func(path string) {
				rel, err := filepath.Rel(cwd, path)
				if err != nil {
					rel = path
				}
				re, cerr := globToRegexp(strings.TrimPrefix(filepath.ToSlash(pattern), "./"))
				if cerr == nil && re.MatchString(filepath.ToSlash(rel)) {
					add(path)
				}
			}); err != nil {
				return nil, err
			}
		}
	}

	sort.Strings(out)
	return out, nil
}

// globRoot returns the longest literal directory prefix of a glob pattern.
func globRoot(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[")
	if idx < 0 {
		dir := filepath.Dir(pattern)
		if dir == "" {
			return "."
		}
		return dir
	}
	prefix := pattern[:idx]
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		prefix = prefix[:i]
	} else {
		prefix = "."
	}
	if prefix == "" {
		prefix = "."
	}
	return prefix
}

// walkDir walks a directory, calling fn for every file with a matching
// extension that is not ignored.
func walkDir(root string, exts []string, ignore *globMatcher, cwd string, fn func(string)) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, as globby does
		}
		rel, rerr := filepath.Rel(cwd, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "node_modules" || (strings.HasPrefix(base, ".") && base != "." && base != "..") {
				return filepath.SkipDir
			}
			if !ignore.empty() && ignore.matches(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !hasExtension(path, exts) {
			return nil
		}
		if !ignore.empty() && ignore.matches(rel) {
			return nil
		}
		fn(path)
		return nil
	})
}
