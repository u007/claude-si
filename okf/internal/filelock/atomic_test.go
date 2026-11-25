package filelock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileAtomicFailurePreservesTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "index.md")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomic(target, []byte("new"), func(string, string) error {
		return errors.New("injected replacement failure")
	})
	if err == nil {
		t.Fatal("expected replacement failure")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "old" {
		t.Fatalf("target changed: %q, %v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp.*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, %v", matches, err)
	}
}

func TestWriteFileAtomicReplacesTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "index.md")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("got %q; want new", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp.*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, %v", matches, err)
	}
}

func TestWriteFileAtomicNewFileIsWorldReadable(t *testing.T) {
	target := filepath.Join(t.TempDir(), "index.md")
	if err := WriteFileAtomic(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	assertMode(t, target, 0644)
}

func TestWriteFileAtomicKeepsExistingMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "log.md")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set the mode explicitly so the umask cannot change the fixture.
	if err := os.Chmod(target, 0640); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	assertMode(t, target, 0640)
}

func assertMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode of %s = %o; want %o", path, got, want)
	}
}
