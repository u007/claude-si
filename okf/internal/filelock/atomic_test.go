package filelock

import (
	"errors"
	"os"
	"path/filepath"
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
