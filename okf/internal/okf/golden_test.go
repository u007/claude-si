package okf

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const localDateToken = "{{LOCAL_DATE}}"

func TestGoldenIndex(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"index-mixed", "index-ignores"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			got, err := RenderIndex(root)
			if err != nil {
				t.Fatal(err)
			}
			want := readExpected(t, name, "docs/index.md")
			if string(got) != string(want) {
				t.Fatalf("index mismatch\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}

func TestGoldenLog(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := readCaseArgs(t, name)
			got, err := RenderLog(root, args[1], args[2], args[3], now)
			if err != nil {
				t.Fatal(err)
			}
			want := normalizeLocalDate(readExpected(t, name, "docs/log.md"), now)
			if string(got) != string(want) {
				t.Fatalf("log mismatch\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}

func fixtureDir(name string) string {
	return filepath.Join("..", "..", "testdata", "perl-golden", name)
}

func materializeFixture(t *testing.T, name string, now time.Time) string {
	t.Helper()
	src := filepath.Join(fixtureDir(name), "input", "docs")
	dst := filepath.Join(t.TempDir(), "bundle")
	date := now.Format("2006-01-02")
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		raw = []byte(strings.ReplaceAll(string(raw), localDateToken, date))
		return os.WriteFile(target, raw, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func readExpected(t *testing.T, name, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir(name), "expected", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func normalizeLocalDate(data []byte, now time.Time) []byte {
	return []byte(strings.ReplaceAll(string(data), localDateToken, now.Format("2006-01-02")))
}

func readCaseArgs(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir(name), "case.json"))
	if err != nil {
		t.Fatal(err)
	}
	var caseFile struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(raw, &caseFile); err != nil {
		t.Fatal(err)
	}
	return caseFile.Args
}

func assertFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s mismatch\nwant:\n%s\ngot:\n%s", path, want, got)
	}
}

func assertExpectedTree(t *testing.T, name, root string) {
	t.Helper()
	expectedRoot := filepath.Join(fixtureDir(name), "expected", "docs")
	wantFiles := make(map[string]struct{})
	err := filepath.WalkDir(expectedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(expectedRoot, path)
		if err != nil {
			return err
		}
		wantFiles[rel] = struct{}{}
		assertFile(t, filepath.Join(root, rel), normalizeLocalDate(readExpected(t, name, filepath.ToSlash(filepath.Join("docs", rel))), time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	gotFiles := make(map[string]struct{})
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == ".okf.lock" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		gotFiles[rel] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(gotFiles) != len(wantFiles) {
		t.Fatalf("file count mismatch: got %d, want %d", len(gotFiles), len(wantFiles))
	}
	for rel := range wantFiles {
		if _, ok := gotFiles[rel]; !ok {
			t.Fatalf("missing file: %s", rel)
		}
	}
}
