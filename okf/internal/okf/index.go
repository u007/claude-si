package okf

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// warningOut receives non-fatal index warnings. Tests swap it to capture output.
var warningOut io.Writer = os.Stderr

type indexEntry struct {
	path       string
	title      string
	desc       string
	deprecated bool
}

type ignorePatterns struct {
	files []*regexp.Regexp
	dirs  []*regexp.Regexp
}

func compileGlob(pattern string) (*regexp.Regexp, error) {
	var out strings.Builder
	out.WriteByte('^')
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			out.WriteString("[^/]*")
			i++
		case '?':
			out.WriteString("[^/]")
			i++
		case '[':
			end := strings.IndexByte(pattern[i+1:], ']')
			if end < 0 {
				out.WriteString(`\[`)
				i++
				continue
			}
			end += i + 1
			class := pattern[i+1 : end]
			if strings.HasPrefix(class, "^") {
				class = "!" + class[1:]
			}
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			out.WriteByte('[')
			out.WriteString(class)
			out.WriteByte(']')
			i = end + 1
		case '\\':
			if i+1 >= len(pattern) {
				out.WriteString(`\\`)
				i++
				continue
			}
			r, size := utf8.DecodeRuneInString(pattern[i+1:])
			out.WriteString(regexp.QuoteMeta(string(r)))
			i += 1 + size
		default:
			r, size := utf8.DecodeRuneInString(pattern[i:])
			out.WriteString(regexp.QuoteMeta(string(r)))
			i += size
		}
	}
	out.WriteByte('$')
	return regexp.Compile(out.String())
}

func RenderIndex(root string) ([]byte, error) {
	bundle, err := LoadBundle(root)
	if err != nil {
		return nil, err
	}
	root = bundle.Root
	ignores, err := readIgnorePatterns(root)
	if err != nil {
		return nil, err
	}
	defaults := make([]*regexp.Regexp, 0, 2)
	for _, pattern := range []string{"PLAN-*.md", "*.OCODE.md"} {
		re, err := compileGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("okf: default pattern %q: %w", pattern, err)
		}
		defaults = append(defaults, re)
	}

	paths := make([]string, 0)
	err = fs.WalkDir(os.DirFS(root), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() && errors.Is(walkErr, fs.ErrPermission) {
				fmt.Fprintf(warningOut, "okf: warning: skipping %s: %v\n", path, walkErr)
				return fs.SkipDir
			}
			return walkErr
		}
		if path == "." {
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(name, ".") || ignores.matchesDir(name, path) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			return nil
		}
		if name == "index.md" || name == "log.md" {
			return nil
		}
		for _, pattern := range defaults {
			if pattern.MatchString(name) {
				return nil
			}
		}
		if ignores.matchesFile(name, path) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("okf: cannot walk %s: %w", root, err)
	}
	slices.Sort(paths)

	sections := make(map[string][]indexEntry)
	unclassified := make([]string, 0)
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("okf: cannot read %s: %w", filepath.Join(root, filepath.FromSlash(rel)), err)
		}
		fields, ok := parseFrontmatter(raw)
		if !ok {
			unclassified = append(unclassified, rel)
			continue
		}
		dir := ""
		if slash := strings.IndexByte(rel, '/'); slash >= 0 {
			dir = rel[:slash]
		}
		sections[dir] = append(sections[dir], indexEntry{
			path:       rel,
			title:      fields["title"],
			desc:       fields["description"],
			deprecated: fields["status"] == "deprecated",
		})
	}
	slices.Sort(unclassified)

	var out strings.Builder
	fmt.Fprintf(&out, "---\nokf_version: %s\n---\n\n", bundle.Version)
	for _, dir := range slices.Sorted(maps.Keys(sections)) {
		heading := dir
		if heading == "" {
			heading = "Concepts"
		}
		fmt.Fprintf(&out, "# %s\n\n", heading)
		entries := sections[dir]
		slices.SortFunc(entries, func(a, b indexEntry) int {
			if order := cmp.Compare(lowerASCII(a.title), lowerASCII(b.title)); order != 0 {
				return order
			}
			return cmp.Compare(a.path, b.path)
		})
		for _, entry := range entries {
			title := entry.title
			if title == "" {
				title = entry.path
			}
			fmt.Fprintf(&out, "- [%s](%s)", title, entry.path)
			if entry.desc != "" {
				fmt.Fprintf(&out, " - %s", entry.desc)
			}
			if entry.deprecated {
				out.WriteString(" (deprecated)")
			}
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
	}
	if len(unclassified) > 0 {
		out.WriteString("# Unclassified\n\n")
		for _, rel := range unclassified {
			name := rel
			if slash := strings.LastIndexByte(rel, '/'); slash >= 0 {
				name = rel[slash+1:]
			}
			fmt.Fprintf(&out, "- [%s](%s)\n", name, rel)
		}
		out.WriteByte('\n')
	}
	return []byte(out.String()), nil
}

func readIgnorePatterns(root string) (ignorePatterns, error) {
	path := filepath.Join(root, ".okfignore")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ignorePatterns{}, nil
		}
		return ignorePatterns{}, fmt.Errorf("okf: cannot read %s: %w", path, err)
	}
	var patterns ignorePatterns
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = trimPerlSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fileRe, err := compileGlob(line)
		if err != nil {
			return ignorePatterns{}, fmt.Errorf("okf: invalid .okfignore pattern %q: %w", line, err)
		}
		dirRe, err := compileGlob(strings.TrimSuffix(line, "/"))
		if err != nil {
			return ignorePatterns{}, fmt.Errorf("okf: invalid .okfignore pattern %q: %w", line, err)
		}
		patterns.files = append(patterns.files, fileRe)
		patterns.dirs = append(patterns.dirs, dirRe)
	}
	return patterns, nil
}

func (patterns ignorePatterns) matchesFile(name, rel string) bool {
	for _, pattern := range patterns.files {
		if pattern.MatchString(name) || pattern.MatchString(rel) {
			return true
		}
	}
	return false
}

func (patterns ignorePatterns) matchesDir(name, rel string) bool {
	for _, pattern := range patterns.dirs {
		if pattern.MatchString(name) || pattern.MatchString(rel) {
			return true
		}
	}
	return false
}

func lowerASCII(value string) string {
	raw := []byte(value)
	for i, char := range raw {
		if char >= 'A' && char <= 'Z' {
			raw[i] = char + ('a' - 'A')
		}
	}
	return string(raw)
}
