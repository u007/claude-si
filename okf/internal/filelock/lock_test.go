package filelock

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWithFileLockSerializes(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".okf.lock")
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithFileLock(path, time.Second, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	started := time.Now()
	time.AfterFunc(150*time.Millisecond, func() { close(release) })
	err := WithFileLock(path, 2*time.Second, func() error { return nil })
	if err != nil || time.Since(started) < 50*time.Millisecond {
		t.Fatalf("second lock did not wait: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWithFileLockReturnsCallbackError(t *testing.T) {
	want := errors.New("callback failed")
	err := WithFileLock(filepath.Join(t.TempDir(), ".okf.lock"), time.Second, func() error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v; want %v", err, want)
	}
}

func TestWithFileLockTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".okf.lock")
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithFileLock(path, time.Second, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	err := WithFileLock(path, 150*time.Millisecond, func() error { return nil })
	if err == nil {
		t.Fatal("expected timeout")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
