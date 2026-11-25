package okf

import (
	"fmt"
	"io"
	"os"
	"time"
)

const usageText = "usage: okf <docs-dir> index | log <Action> <doc-path> <summary> | remind\n"

func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, time.Now())
}

func run(args []string, stdout, stderr io.Writer, now time.Time) int {
	if len(args) == 1 && args[0] == "remind" {
		return runReminder(stdout, stderr)
	}
	if len(args) == 2 && args[1] == "index" {
		if err := RegenerateIndex(args[0]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "okf: index.md regenerated")
		return 0
	}
	if len(args) == 5 && args[1] == "log" {
		if err := LogAndIndex(args[0], args[2], args[3], args[4], now); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "okf: index.md regenerated")
		return 0
	}
	fmt.Fprint(stderr, usageText)
	return 2
}

func runReminder(stdout, stderr io.Writer) int {
	projectDir, ok := os.LookupEnv("CLAUDE_PROJECT_DIR")
	if !ok {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "okf: cannot determine project directory: %v\n", err)
			return 1
		}
	}
	text, found, err := Reminder(projectDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if found {
		fmt.Fprintln(stdout, text)
	}
	return 0
}
