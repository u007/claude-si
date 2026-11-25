package okf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const ReminderText = `OKF knowledge bundle detected at docs/ (docs/index.md has okf_version). Before starting ANY task in this project, dispatch the okf:context agent (Agent tool, subagent_type okf:context) with the task topic for a lookup digest, and work from that. After a task that produced durable knowledge, dispatch okf:context-writer with a self-contained update brief, then check its work with git diff docs/. Do not edit docs/ yourself. If these agents are unavailable, invoke the okf skill and follow its steps directly.`

func Reminder(projectDir string) (string, bool, error) {
	index := filepath.Join(projectDir, "docs", "index.md")
	raw, err := os.ReadFile(index)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("okf: cannot read %s: %w", index, err)
	}
	if _, found := okfVersion(raw); !found {
		return "", false, nil
	}
	return ReminderText, true, nil
}
