package filelock

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const retryInterval = 100 * time.Millisecond

func WithFileLock(path string, timeout time.Duration, fn func() error) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("okf: cannot open %s: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("okf: close %s: %w", path, closeErr))
		}
	}()

	deadline := time.Now().Add(timeout)
	for {
		lockErr := tryLockFile(file)
		if lockErr == nil {
			break
		}
		if !isLockContention(lockErr) {
			return fmt.Errorf("okf: lock %s: %w", path, lockErr)
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("okf: timed out after %s waiting for %s", timeout, path)
		}
		time.Sleep(retryInterval)
	}
	defer func() {
		if unlockErr := unlockFile(file); unlockErr != nil {
			err = errors.Join(err, fmt.Errorf("okf: unlock %s: %w", path, unlockErr))
		}
	}()
	return fn()
}
