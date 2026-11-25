package filelock

import (
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	replaceAttempts   = 10
	replaceRetryDelay = 50 * time.Millisecond
)

// retryReplace runs attempt until it succeeds, fails with an error retryable
// rejects, or replaceAttempts is reached. Every retry is announced on stderr so
// it is never silent. The announcement names the attempt about to run, so
// "retrying (2/10)" is the second of at most ten attempts.
func retryReplace(target string, stderr io.Writer, retryable func(error) bool, sleep func(time.Duration), attempt func() error) error {
	for n := 1; ; n++ {
		err := attempt()
		if err == nil {
			return nil
		}
		if n >= replaceAttempts || !retryable(err) {
			return err
		}
		next := n + 1
		if _, writeErr := fmt.Fprintf(stderr, "okf: replace %s: %v; retrying (%d/%d)\n", target, err, next, replaceAttempts); writeErr != nil {
			return errors.Join(err, fmt.Errorf("okf: write retry notice for %s: %w", target, writeErr))
		}
		sleep(replaceRetryDelay)
	}
}
