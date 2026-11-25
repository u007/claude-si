package okf

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadBundle(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version: \"0.3\"\n---\n"))
		bundle, err := LoadBundle(filepath.Join(root, "docs"))
		if err != nil {
			t.Fatal(err)
		}
		if bundle.Root != filepath.Join(root, "docs") || bundle.Version != "0.3" {
			t.Fatalf("unexpected bundle: %#v", bundle)
		}
	})

	t.Run("missing index", func(t *testing.T) {
		_, err := LoadBundle(filepath.Join(t.TempDir(), "docs"))
		if err == nil || !strings.Contains(err.Error(), "cannot read") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty version", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version:\n---\n"))
		_, err := LoadBundle(filepath.Join(root, "docs"))
		if err == nil || !strings.Contains(err.Error(), "no okf_version marker") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRenderIndexRequiresVersion(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("# no frontmatter\n"))
	_, err := RenderIndex(filepath.Join(root, "docs"))
	if err == nil || !strings.Contains(err.Error(), "no okf_version marker") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderIndexDoesNotWriteIndex(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "docs", "index.md")
	raw := []byte("---\nokf_version: 0.1\n---\n\n# Old\n")
	writeTestFile(t, indexPath, raw)
	writeTestFile(t, filepath.Join(root, "docs", "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
	if _, err := RenderIndex(filepath.Join(root, "docs")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("RenderIndex wrote to disk: %q", got)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRenderIndexRejectsInvalidIgnorePattern(t *testing.T) {
	for _, pattern := range []string{"[]", "[!]", "[^]", "[z-a]"} {
		t.Run(pattern, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
			writeTestFile(t, filepath.Join(root, "docs", ".okfignore"), []byte(pattern+"\n"))
			_, err := RenderIndex(filepath.Join(root, "docs"))
			want := `okf: invalid .okfignore pattern "` + pattern + `"`
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("want error starting with %q, got %v", want, err)
			}
		})
	}
}

func TestRenderIndexSkipsUnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict directory reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	docs := filepath.Join(t.TempDir(), "docs")
	writeTestFile(t, filepath.Join(docs, "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
	writeTestFile(t, filepath.Join(docs, "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
	writeTestFile(t, filepath.Join(docs, "locked", "hidden.md"), []byte("---\ntitle: Hidden\n---\n"))
	locked := filepath.Join(docs, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0700); err != nil {
			t.Error(err)
		}
	})

	var warn bytes.Buffer
	previous := warningOut
	warningOut = &warn
	t.Cleanup(func() { warningOut = previous })

	got, err := RenderIndex(docs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "[Alpha](alpha.md)") {
		t.Fatalf("index lost sibling doc:\n%s", got)
	}
	if strings.Contains(string(got), "Hidden") {
		t.Fatalf("index includes doc from unreadable dir:\n%s", got)
	}
	if !strings.HasPrefix(warn.String(), "okf: warning: skipping locked: ") {
		t.Fatalf("missing skip warning, got %q", warn.String())
	}
}
