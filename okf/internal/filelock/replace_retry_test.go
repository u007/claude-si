package filelock

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

var errContention = errors.New("contention")

func noSleep(time.Duration) {}

func TestRetryReplaceSucceedsAfterContention(t *testing.T) {
	var stderr bytes.Buffer
	calls := 0
	err := retryReplace("docs/index.md", &stderr, func(error) bool { return true }, noSleep, func() error {
		calls++
		if calls < 3 {
			return errContention
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d; want 3", calls)
	}
	out := stderr.String()
	for _, want := range []string{"retrying (2/10)", "retrying (3/10)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stderr missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "retrying (4/10)") {
		t.Fatalf("unexpected extra retry notice:\n%s", out)
	}
}

func TestRetryReplaceGivesUpAfterTenAttempts(t *testing.T) {
	var stderr bytes.Buffer
	calls := 0
	err := retryReplace("docs/index.md", &stderr, func(error) bool { return true }, noSleep, func() error {
		calls++
		return errContention
	})
	if !errors.Is(err, errContention) {
		t.Fatalf("err = %v; want contention error", err)
	}
	if calls != replaceAttempts {
		t.Fatalf("calls = %d; want %d", calls, replaceAttempts)
	}
	if got := strings.Count(stderr.String(), "retrying ("); got != replaceAttempts-1 {
		t.Fatalf("retry notices = %d; want %d", got, replaceAttempts-1)
	}
}

func TestRetryReplaceDoesNotRetryOtherErrors(t *testing.T) {
	var stderr bytes.Buffer
	calls := 0
	err := retryReplace("docs/index.md", &stderr, func(error) bool { return false }, noSleep, func() error {
		calls++
		return errContention
	})
	if !errors.Is(err, errContention) {
		t.Fatalf("err = %v; want contention error", err)
	}
	if calls != 1 || stderr.Len() != 0 {
		t.Fatalf("calls = %d, stderr = %q; want one silent attempt", calls, stderr.String())
	}
}

func TestRetryReplaceFailsWhenNoticeCannotBeWritten(t *testing.T) {
	calls := 0
	err := retryReplace("docs/index.md", failingWriter{}, func(error) bool { return true }, noSleep, func() error {
		calls++
		return errContention
	})
	if err == nil || !strings.Contains(err.Error(), "write retry notice") {
		t.Fatalf("err = %v; want notice write failure", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d; want 1 (no silent retry)", calls)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }
