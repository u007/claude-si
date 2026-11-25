package okf

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLogAndIndexGolden(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := readCaseArgs(t, name)
			if err := LogAndIndex(root, args[1], args[2], args[3], now); err != nil {
				t.Fatal(err)
			}
			wantLog := normalizeLocalDate(readExpected(t, name, "docs/log.md"), now)
			assertFile(t, filepath.Join(root, "log.md"), wantLog)
			assertFile(t, filepath.Join(root, "index.md"), readExpected(t, name, "docs/index.md"))
		})
	}
}
