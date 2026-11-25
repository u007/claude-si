package okf

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunGolden(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"index-mixed", "index-ignores", "log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := append([]string{root}, readCaseArgs(t, name)...)
			var stdout, stderr bytes.Buffer
			code := run(args, &stdout, &stderr, now)
			wantCode, err := strconv.Atoi(strings.TrimSpace(string(readExpected(t, name, "exit.txt"))))
			if err != nil {
				t.Fatal(err)
			}
			if code != wantCode || stdout.String() != string(readExpected(t, name, "stdout.txt")) || stderr.String() != string(readExpected(t, name, "stderr.txt")) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			assertExpectedTree(t, name, root)
		})
	}
}

func TestRunReminder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
	t.Setenv("CLAUDE_PROJECT_DIR", root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"remind"}, &stdout, &stderr, time.Now())
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if stdout.String() != ReminderText+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunReminderAbsentBundle(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run([]string{"remind"}, &stdout, &stderr, time.Now())
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunUsageErrors(t *testing.T) {
	root := t.TempDir()
	cases := [][]string{
		nil,
		{root},
		{root, "unknown"},
		{"remind", "extra"},
		{root, "log", "Update", "alpha.md"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr, time.Now())
		if code != 2 || !strings.Contains(stderr.String(), "usage:") {
			t.Fatalf("args=%q code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestRunRuntimeErrors(t *testing.T) {
	cases := []struct {
		name string
		args func(string) []string
		want string
	}{
		{"invalid action", func(root string) []string {
			return []string{root, "log", "create", "alpha.md", "Summary"}
		}, "action must be"},
		{"traversal", func(root string) []string {
			return []string{root, "log", "Update", "../alpha.md", "Summary"}
		}, "stay inside"},
		{"reserved", func(root string) []string {
			return []string{root, "log", "Update", "index.md", "Summary"}
		}, "must not be"},
		{"multiline", func(root string) []string {
			return []string{root, "log", "Update", "alpha.md", "two\nlines"}
		}, "one line"},
		{"missing document", func(root string) []string {
			return []string{root, "log", "Update", "missing.md", "Summary"}
		}, "does not exist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
			writeTestFile(t, filepath.Join(root, "docs", "alpha.md"), []byte("---\ntitle: Alpha\n---\n"))
			var stdout, stderr bytes.Buffer
			code := run(tc.args(filepath.Join(root, "docs")), &stdout, &stderr, time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local))
			if code != 1 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunReminderUsesWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"remind"}, &stdout, &stderr, time.Now())
	if code != 0 || !strings.Contains(stdout.String(), ReminderText) || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunReminderUnreadableBundle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "docs"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PROJECT_DIR", root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"remind"}, &stdout, &stderr, time.Now())
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
