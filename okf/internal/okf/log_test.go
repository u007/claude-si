package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderLogValidation(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	cases := []struct {
		name    string
		action  string
		docPath string
		summary string
		want    string
	}{
		{"invalid action", "create", "alpha.md", "Summary", "action must be"},
		{"parent segment", "Update", "../alpha.md", "Summary", "stay inside"},
		{"windows parent segment", "Update", `..\alpha.md`, "Summary", "stay inside"},
		{"windows rooted path", "Update", `\alpha.md`, "Summary", "stay inside"},
		{"index path", "Update", "index.md", "Summary", "must not be"},
		{"log path", "Update", "log.md", "Summary", "must not be"},
		{"missing document", "Update", "missing.md", "Summary", "does not exist"},
		{"newline summary", "Update", "alpha.md", "two\nlines", "one line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
			_, err := RenderLog(root, tc.action, tc.docPath, tc.summary, now)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRenderLogDeletionAllowsMissingDocument(t *testing.T) {
	root := t.TempDir()
	got, err := RenderLog(root, "Deletion", "missing.md", "Removed", time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "* **Deletion**: Removed ([missing.md](/missing.md))") {
		t.Fatalf("unexpected log: %s", got)
	}
}

func TestRenderLogRequiresTitle(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, content := range []string{"", "# Other\n"} {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
		writeTestFile(t, filepath.Join(root, "log.md"), []byte(content))
		_, err := RenderLog(root, "Update", "alpha.md", "Summary", now)
		if err == nil || !strings.Contains(err.Error(), "no '# Directory Update Log' heading") {
			t.Fatalf("unexpected error for %q: %v", content, err)
		}
	}
}

func TestRenderLogPreservesExistingCRBytes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
	writeTestFile(t, filepath.Join(root, "log.md"), []byte("# Directory Update Log\r\n\r\n## 2026-11-25\r\n\r\n* old\r\n"))
	got, err := RenderLog(root, "Update", "alpha.md", "New", time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Directory Update Log\r\n\r\n## 2026-11-25\r\n\r\n* **Update**: New ([alpha.md](/alpha.md))\n* old\r\n"
	if string(got) != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestRenderLogDoesNotWriteLog(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "log.md")
	raw := []byte("# Directory Update Log\n")
	writeTestFile(t, filepath.Join(root, "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
	writeTestFile(t, logPath, raw)
	if _, err := RenderLog(root, "Update", "alpha.md", "New", time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("RenderLog wrote to disk: %q", got)
	}
}
