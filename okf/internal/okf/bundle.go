package okf

import (
	"fmt"
	"os"
	"path/filepath"
)

type Bundle struct {
	Root    string
	Version string
}

func LoadBundle(root string) (*Bundle, error) {
	root = filepath.Clean(root)
	index := filepath.Join(root, "index.md")
	raw, err := os.ReadFile(index)
	if err != nil {
		return nil, fmt.Errorf("okf: cannot read %s: %w", index, err)
	}
	version, found := okfVersion(raw)
	if !found || version == "" {
		return nil, fmt.Errorf("okf: %s has no okf_version marker; not a bundle", index)
	}
	return &Bundle{Root: root, Version: version}, nil
}
