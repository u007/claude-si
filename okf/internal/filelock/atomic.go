package filelock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// newFileMode is the mode for a file that does not exist yet. os.CreateTemp
// would make it 0600.
const newFileMode fs.FileMode = 0644

func WriteFileAtomic(target string, data []byte) error {
	return writeFileAtomic(target, data, replaceFile)
}

func writeFileAtomic(target string, data []byte, replace func(string, string) error) (err error) {
	mode, err := targetMode(target)
	if err != nil {
		return err
	}
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
	// os.CreateTemp makes the file 0600. Give it the target's mode so the
	// replacement does not change permissions.
	if err = os.Chmod(name, mode); err != nil {
		return fmt.Errorf("okf: chmod %s: %w", name, err)
	}
	if err = replace(name, target); err != nil {
		return fmt.Errorf("okf: replace %s: %w", target, err)
	}
	replaced = true
	return nil
}

// targetMode returns the permission bits the replacement file should have: the
// existing target's mode, or 0644 when the target does not exist yet. Any other
// stat error is returned rather than guessed around.
func targetMode(target string) (fs.FileMode, error) {
	info, err := os.Stat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return newFileMode, nil
	}
	if err != nil {
		return 0, fmt.Errorf("okf: stat %s: %w", target, err)
	}
	return info.Mode().Perm(), nil
}
