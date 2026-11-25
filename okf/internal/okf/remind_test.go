package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReminder(t *testing.T) {
	cases := []struct {
		name   string
		index  string
		want   bool
		err    bool
		absent bool
	}{
		{"LF bundle", "---\nokf_version: 0.1\n---\n", true, false, false},
		{"CRLF bundle", "---\r\nokf_version: 0.1\r\n---\r\n", true, false, false},
		{"empty marker counts as detected", "---\nokf_version:\n---\n", true, false, false},
		{"marker outside frontmatter", "# okf_version: 0.1\n", false, false, false},
		{"unclosed frontmatter", "---\nokf_version: 0.1\n", false, false, false},
		{"no bundle", "", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if !tc.absent {
				writeTestFile(t, filepath.Join(root, "docs", "index.md"), []byte(tc.index))
			}
			text, found, err := Reminder(root)
			if (err != nil) != tc.err || found != tc.want {
				t.Fatalf("got %q, %v, %v; want found=%v err=%v", text, found, err, tc.want, tc.err)
			}
			if found && text != ReminderText {
				t.Fatalf("unexpected reminder text: %q", text)
			}
			if !found && text != "" {
				t.Fatalf("unexpected text when no bundle: %q", text)
			}
		})
	}
}

func TestReminderUnreadableIndexIsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "docs"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Reminder(root)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestReminderTextMatchesShell(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "okf-reminder.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var shellText string
	for line := range strings.SplitSeq(string(raw), "\n") {
		if value, ok := strings.CutPrefix(line, `echo "`); ok {
			shellText, _ = strings.CutSuffix(value, `"`)
			break
		}
	}
	if shellText == "" {
		t.Fatal("shell reminder text not found")
	}
	if ReminderText != shellText {
		t.Fatalf("ReminderText drifted\nwant: %q\ngot:  %q", shellText, ReminderText)
	}
}
