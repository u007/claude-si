package scripts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLauncherPathsWithSpaces(t *testing.T) {
	plugin := filepath.Join(t.TempDir(), "plugin with spaces")
	copyLauncher(t, plugin)
	copyHostBinary(t, plugin)
	project := filepath.Join(t.TempDir(), "project with spaces")
	makeBundle(t, project)

	shells := []string{"bash"}
	if runtime.GOOS != "windows" {
		shells = append(shells, "sh")
	}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			cmd := launcherCommandShell(shell, filepath.Join(plugin, "scripts", "okf.cmd"), "remind")
			cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+project)
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), "OKF knowledge bundle detected") {
				t.Fatalf("launcher failed: %v\n%s", err, out)
			}
		})
	}
}

func TestLauncherStructure(t *testing.T) {
	raw, err := os.ReadFile("okf.cmd")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("launcher must not have a BOM")
	}
	if bytes.ContainsRune(raw, '\r') {
		t.Fatal("launcher must use LF line endings")
	}
	lines := strings.Split(string(raw), "\n")
	if lines[0] != ": << 'CMDBLOCK'" {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	terminator := -1
	for i, line := range lines {
		if line == "CMDBLOCK" {
			terminator = i
			break
		}
	}
	if terminator < 0 {
		t.Fatal("missing CMDBLOCK terminator")
	}
	exitAt := -1
	for i := 0; i < terminator; i++ {
		if strings.Contains(lines[i], "exit /b") {
			exitAt = i
		}
	}
	if exitAt < 0 {
		t.Fatal("missing batch exit /b before terminator")
	}
}

func TestLauncherMissingBinaryFails(t *testing.T) {
	plugin := t.TempDir()
	copyLauncher(t, plugin)
	cmd := launcherCommand(filepath.Join(plugin, "scripts", "okf.cmd"), "remind")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "missing binary") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestLauncherPropagatesExitCode(t *testing.T) {
	plugin := t.TempDir()
	copyLauncher(t, plugin)
	copyHostBinary(t, plugin)
	err := launcherCommand(filepath.Join(plugin, "scripts", "okf.cmd")).Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("got %v; want exit code 2", err)
	}
}

func TestLauncherSummarySpecialCharacters(t *testing.T) {
	plugin := t.TempDir()
	copyLauncher(t, plugin)
	copyHostBinary(t, plugin)
	project := t.TempDir()
	makeBundle(t, project)
	summary := `Words "quoted" & Unicode α`
	cmd := launcherCommand(filepath.Join(plugin, "scripts", "okf.cmd"), filepath.Join(project, "docs"), "log", "Update", "alpha.md", summary)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launcher failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(filepath.Join(project, "docs", "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), summary) {
		t.Fatalf("summary missing from log:\n%s", raw)
	}
}

func launcherCommand(launcher string, args ...string) *exec.Cmd {
	return launcherCommandShell("bash", launcher, args...)
}

func launcherCommandShell(shell, launcher string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return launcherCommandWindows(launcher, args...)
	}
	return exec.Command(shell, "-c", shellCommand(launcher, args...))
}

func shellCommand(launcher string, args ...string) string {
	parts := []string{shellQuote(launcher)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// windowsCommandArgs returns the argument part of the cmd.exe command line that
// runs the launcher with args. Each argument is quoted for Go's argv parser,
// which is what okf.exe uses to split %* in the launcher. The parser reads \"
// as a literal quote; it does not read "" as one, so \" is used. Panics when
// the result would let cmd.exe treat a metacharacter as syntax, because caret
// escaping cannot be applied safely across both cmd.exe passes (the /S /C
// parse and the launcher's own %* re-parse).
func windowsCommandArgs(launcher string, args ...string) string {
	parts := []string{msvcrtQuote(launcher)}
	for _, arg := range args {
		if strings.ContainsRune(arg, '%') {
			panic(fmt.Sprintf("test argument %q contains %%, which cmd.exe expands", arg))
		}
		parts = append(parts, msvcrtQuote(arg))
	}
	line := strings.Join(parts, " ")
	if i := cmdUnquotedMetaIndex(line); i >= 0 {
		panic(fmt.Sprintf("cmd.exe metacharacter %q is outside quotes in %s", line[i], line))
	}
	return line
}

// msvcrtQuote wraps value in double quotes in the form Go's argv parser reads
// back unchanged: a backslash run before a quote is doubled, and the quote
// itself becomes \".
func msvcrtQuote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat(`\`, backslashes*2+1))
			b.WriteByte('"')
			backslashes = 0
		default:
			b.WriteString(strings.Repeat(`\`, backslashes))
			b.WriteByte(value[i])
			backslashes = 0
		}
	}
	b.WriteString(strings.Repeat(`\`, backslashes*2))
	b.WriteByte('"')
	return b.String()
}

// cmdUnquotedMetaIndex returns the index of the first cmd.exe metacharacter
// outside double quotes, or -1. cmd.exe toggles its quote state on every double
// quote, including the ones that Go's argv rules escape with a backslash.
func cmdUnquotedMetaIndex(line string) int {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case c == '"':
			inQuote = !inQuote
		case !inQuote && strings.IndexByte("&|<>^", c) >= 0:
			return i
		}
	}
	return -1
}

func copyLauncher(t *testing.T, plugin string) {
	t.Helper()
	source, err := os.ReadFile("okf.cmd")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(plugin, "scripts", "okf.cmd")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, source, 0755); err != nil {
		t.Fatal(err)
	}
}

func copyHostBinary(t *testing.T, plugin string) {
	t.Helper()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	relative := filepath.Join("..", "bin", runtime.GOOS, runtime.GOARCH, "okf"+suffix)
	source, err := os.ReadFile(relative)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(plugin, "bin", runtime.GOOS, runtime.GOARCH, "okf"+suffix)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, source, 0755); err != nil {
		t.Fatal(err)
	}
}

func makeBundle(t *testing.T, project string) {
	t.Helper()
	docs := filepath.Join(project, "docs")
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "index.md"), []byte("---\nokf_version: 0.1\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "alpha.md"), []byte("---\ntitle: Alpha\ndescription: Alpha description\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsCommandArgsKeepsArgumentsIntact(t *testing.T) {
	got := windowsCommandArgs(`C:\plugin with spaces\okf.cmd`, `Words "quoted" & Unicode α`)
	want := `"C:\plugin with spaces\okf.cmd" "Words \"quoted\" & Unicode α"`
	if got != want {
		t.Fatalf("command args = %s\nwant %s", got, want)
	}
}

func TestWindowsCommandArgsRefusesUnsafeMetacharacters(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for a metacharacter outside quotes")
		}
	}()
	windowsCommandArgs("okf.cmd", `a "b & c" d`)
}
