package okf

import (
	"path/filepath"
	"time"

	"github.com/u007/claude-plugins/okf/internal/filelock"
)

const lockTimeout = 60 * time.Second

func RegenerateIndex(root string) error {
	bundle, err := LoadBundle(root)
	if err != nil {
		return err
	}
	return filelock.WithFileLock(filepath.Join(bundle.Root, ".okf.lock"), lockTimeout, func() error {
		data, err := RenderIndex(bundle.Root)
		if err != nil {
			return err
		}
		return filelock.WriteFileAtomic(filepath.Join(bundle.Root, "index.md"), data)
	})
}

func LogAndIndex(root, action, docPath, summary string, now time.Time) error {
	bundle, err := LoadBundle(root)
	if err != nil {
		return err
	}
	return filelock.WithFileLock(filepath.Join(bundle.Root, ".okf.lock"), lockTimeout, func() error {
		logData, err := RenderLog(bundle.Root, action, docPath, summary, now)
		if err != nil {
			return err
		}
		indexData, err := RenderIndex(bundle.Root)
		if err != nil {
			return err
		}
		if err := filelock.WriteFileAtomic(filepath.Join(bundle.Root, "log.md"), logData); err != nil {
			return err
		}
		return filelock.WriteFileAtomic(filepath.Join(bundle.Root, "index.md"), indexData)
	})
}
