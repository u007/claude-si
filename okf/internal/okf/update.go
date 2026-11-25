package okf

import (
	"path/filepath"
	"time"

	"github.com/u007/claude-plugins/okf/internal/filelock"
)

const lockTimeout = 60 * time.Second

func RegenerateIndex(root string) error {
	return filelock.WithFileLock(filepath.Join(root, ".okf.lock"), lockTimeout, func() error {
		data, err := RenderIndex(root)
		if err != nil {
			return err
		}
		return filelock.WriteFileAtomic(filepath.Join(root, "index.md"), data)
	})
}

func LogAndIndex(root, action, docPath, summary string, now time.Time) error {
	return filelock.WithFileLock(filepath.Join(root, ".okf.lock"), lockTimeout, func() error {
		logData, err := RenderLog(root, action, docPath, summary, now)
		if err != nil {
			return err
		}
		if err := filelock.WriteFileAtomic(filepath.Join(root, "log.md"), logData); err != nil {
			return err
		}
		indexData, err := RenderIndex(root)
		if err != nil {
			return err
		}
		return filelock.WriteFileAtomic(filepath.Join(root, "index.md"), indexData)
	})
}
