package filelock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func WriteFileAtomic(target string, data []byte) error {
	return writeFileAtomic(target, data, replaceFile)
}

func writeFileAtomic(target string, data []byte, replace func(string, string) error) (err error) {
	temp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp.*")
	if err != nil {
		return fmt.Errorf("okf: create temporary file for %s: %w", target, err)
	}
	name := temp.Name()
	closed := false
	replaced := false
	defer func() {
		if !closed {
			if closeErr := temp.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("okf: close %s: %w", name, closeErr))
			}
		}
		if !replaced {
			if removeErr := os.Remove(name); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("okf: remove %s: %w", name, removeErr))
			}
		}
	}()

	if _, err = temp.Write(data); err != nil {
		return fmt.Errorf("okf: write %s: %w", name, err)
	}
	if err = temp.Sync(); err != nil {
		return fmt.Errorf("okf: sync %s: %w", name, err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("okf: close %s: %w", name, err)
	}
	closed = true
	if err = replace(name, target); err != nil {
		return fmt.Errorf("okf: replace %s: %w", target, err)
	}
	replaced = true
	return nil
}
