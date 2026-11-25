package main

import (
	"os"

	"github.com/u007/claude-plugins/okf/internal/okf"
)

func main() {
	os.Exit(okf.Run(os.Args[1:], os.Stdout, os.Stderr))
}
