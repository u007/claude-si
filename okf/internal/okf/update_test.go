package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogAndIndexGolden(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := readCaseArgs(t, name)
			if err := LogAndIndex(root, args[1], args[2], args[3], now); err != nil {
				t.Fatal(err)
			}
			wantLog := normalizeLocalDate(readExpected(t, name, "docs/log.md"), now)
			assertFile(t, filepath.Join(root, "log.md"), wantLog)
			assertFile(t, filepath.Join(root, "index.md"), readExpected(t, name, "docs/index.md"))
		})
	}
}

func TestNonBundleWritesNothing(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	cases := map[string]func(root string) error{
		"LogAndIndex": func(root string) error {
			return LogAndIndex(root, "Creation", "a.md", "s", now)
		},
		"RegenerateIndex": RegenerateIndex,
	}
	setups := map[string]func(t *testing.T, root string){
		"no index": func(t *testing.T, root string) {
			writeTestFile(t, filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\n"))
		},
		"index without marker": func(t *testing.T, root string) {
			writeTestFile(t, filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\n"))
			writeTestFile(t, filepath.Join(root, "index.md"), []byte("# plain\n"))
		},
	}
	for opName, op := range cases {
		for setupName, setup := range setups {
			t.Run(opName+"/"+setupName, func(t *testing.T) {
				root := t.TempDir()
				setup(t, root)
				if err := op(root); err == nil || !strings.Contains(err.Error(), "okf:") {
					t.Fatalf("want okf error, got %v", err)
				}
				for _, name := range []string{"log.md", ".okf.lock"} {
					if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
						t.Fatalf("%s must not be created in a non-bundle dir (stat err: %v)", name, err)
					}
				}
			})
		}
	}
}

func TestLogAndIndexFailureLeavesLogUnchanged(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	root := t.TempDir()
	logBefore := []byte("# Directory Update Log\n\n## 2026-11-24\n\n* **Creation**: old ([b.md](/b.md))\n")
	writeTestFile(t, filepath.Join(root, "index.md"), []byte("---\nokf_version: 0.1\n---\n"))
	writeTestFile(t, filepath.Join(root, "log.md"), logBefore)
	writeTestFile(t, filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\n"))
	writeTestFile(t, filepath.Join(root, ".okfignore"), []byte("[]\n"))

	if err := LogAndIndex(root, "Creation", "a.md", "s", now); err == nil || !strings.Contains(err.Error(), `"[]"`) {
		t.Fatalf("want invalid pattern error, got %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(logBefore) {
		t.Fatalf("log.md changed after index failure:\n%s", got)
	}
}
