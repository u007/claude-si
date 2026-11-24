# OKF Go Cross-Platform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (default) or superpowers:subagent-driven-development (long plans of independent tasks) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the OKF Perl updater and POSIX reminder with one standalone Go CLI that runs natively on macOS, Linux, and Windows for Claude Code and ocode.

**Architecture:** A standard-library-only Go module owns parsing, index rendering, log insertion, reminder output, locking, and atomic replacement. A thin shell/batch polyglot selects one of six committed binaries. Claude Code and ocode invoke the same launcher command.

**Tech Stack:** Go 1.24.0, POSIX `sh`, Windows `cmd.exe`, GitHub Actions, existing Python unittest suite.

**Spec:** `docs/superpowers/specs/2026-11-25-okf-go-cross-platform-design.md`

**Detail:** decisions, not code — the implementer writes the implementation.

## Global Constraints

- Native Windows must work without Git Bash.
- Both hosts use `"${CLAUDE_PLUGIN_ROOT}/scripts/okf.cmd" remind`.
- Support only `{darwin,linux,windows} × {amd64,arm64}`.
- Use Go 1.24.0 exactly; set `GOTOOLCHAIN=go1.24.0` for local builds.
- Use only the Go standard library.
- Preserve successful Perl output bytes and the exact reminder text.
- Keep `.okf.lock` compatible with ocode's `knowledge.WithBundleLock`.
- Use a 60-second lock timeout with 100 ms contention retries.
- Temporary files use `0600` semantics and same-directory atomic replacement.
- No runtime fallback, silent retry, or hand-edit fallback is permitted.
- Pin `okf/scripts/okf.cmd` to LF and mark `okf/bin/**` binary.
- Use no shebang or BOM; the launcher's first line is exactly `: << 'CMDBLOCK'`.
- Commit golden fixtures so Windows CI never needs Perl.
- Preserve peer WIP; re-read shared files before editing and commit only explicit paths.
- Run the Modern Go Guidelines CLI for Go 1.24 before editing each new Go file.
- Skip only guidelines that require external dependencies; the standard-library-only constraint takes precedence.

## Review Focus

- Windows batch parsing and LF preservation can break the launcher; Task 4 tests structure, Bash, `cmd.exe`, and PowerShell invocation.
- Windows lock flags and byte range must match ocode; Task 3 tests contention on every CI operating system.
- Shell and batch argument quoting can corrupt log summaries; Task 4 tests spaces, quotes, ampersands, and Unicode.
- Date-dependent golden files can rot; Task 1 uses `{{LOCAL_DATE}}` and Task 3 substitutes a fixed clock.
- Failed writes can leave partial files or temporary debris; Task 3 tests target preservation and cleanup.

---

### Task 1: Capture Perl Golden Fixtures

**Risk:** high — this characterizes untested behavior before replacement.

**Files:**
- Create: `okf/go.mod`
- Create: `okf/testdata/perl-golden/README.md`
- Create: `okf/testdata/perl-golden/<case>/input/docs/**`
- Create: `okf/testdata/perl-golden/<case>/expected/**`
- Create: `okf/testdata/perl-golden/<case>/expected/stdout.txt`
- Create: `okf/testdata/perl-golden/<case>/expected/stderr.txt`
- Create: `okf/testdata/perl-golden/<case>/expected/exit.txt`
- Create: `okf/testdata/perl-golden/<case>/case.json`
- Create: `okf/internal/okf/golden_test.go`

**Interfaces:**
- Produces: `RenderIndex(root string) ([]byte, error)` contract exercised by tests.
- Produces: `RenderLog(root, action, docPath, summary string, now time.Time) ([]byte, error)` contract exercised by tests.
- Produces: test helpers `materializeFixture(t *testing.T, name string, now time.Time) string`, `readExpected(t *testing.T, name, rel string) []byte`, `normalizeLocalDate(data []byte, now time.Time) []byte`, `readCaseArgs(t *testing.T, name string) []string`, `assertFile(t *testing.T, path string, want []byte)`, and `assertExpectedTree(t *testing.T, name, root string)`.

- [ ] **Step 1: Create the Go module**

Create `okf/go.mod` with module `github.com/u007/claude-plugins/okf`, `go 1.24.0`, and no dependencies.

- [ ] **Step 2: Create fixture inputs**

Create these cases under `okf/testdata/perl-golden/`:

- `index-mixed`: root and nested docs, deprecated doc, unclassified doc, hidden paths, quoted and multiline scalars, CRLF input, tied titles, Unicode titles.
- `index-ignores`: `.okfignore` file and directory patterns plus default exclusions.
- `log-new`: valid bundle with no `log.md`.
- `log-existing-today`: `log.md` containing `{{LOCAL_DATE}}`.
- `log-existing-older`: `log.md` with only older date headings.
- `log-deletion`: `Deletion` for a path that does not exist.

Every input bundle contains a valid `docs/index.md` with `okf_version`.

Each case also contains `case.json` with its CLI arguments after the docs directory, such as `{"args":["index"]}` or `{"args":["log","Creation","concepts/alpha.md","Created alpha"]}`.

- [ ] **Step 3: Capture Perl results**

Run the current `okf/skills/okf/scripts/okf-update.pl` against temporary copies of each input bundle. Replace `{{LOCAL_DATE}}` in copied inputs with the capture date before execution. Store stdout, stderr, exit code, and every resulting `docs/**` file except `.okf.lock` under `expected/`. Replace the capture date in expected log files with `{{LOCAL_DATE}}`. Implement `assertExpectedTree` to walk `expected/docs/**` and apply `normalizeLocalDate` before comparing each file.

- [ ] **Step 4: Document fixture provenance**

State in `README.md` that fixtures were generated from Perl commit `f74ffe0`, list the covered behaviors, and explain the date token.

- [ ] **Step 5: Write failing golden tests**

Implement all fixture helpers in `golden_test.go`; only `RenderIndex` and `RenderLog` remain undefined.

```go
func TestGoldenIndex(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"index-mixed", "index-ignores"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			got, err := RenderIndex(root)
			if err != nil {
				t.Fatal(err)
			}
			want := readExpected(t, name, "docs/index.md")
			if string(got) != string(want) {
				t.Fatalf("index mismatch\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}

func TestGoldenLog(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := readCaseArgs(t, name)
			got, err := RenderLog(root, args[1], args[2], args[3], now)
			if err != nil {
				t.Fatal(err)
			}
			want := normalizeLocalDate(readExpected(t, name, "docs/log.md"), now)
			if string(got) != string(want) {
				t.Fatalf("log mismatch\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}
```

- [ ] **Step 6: Verify the red state**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./internal/okf`

Expected: FAIL because `RenderIndex` and `RenderLog` do not exist.

---

### Task 2: Implement Parsing and Pure Renderers

**Risk:** high — parsing, glob translation, and ordering have edge cases.

**Files:**
- Create: `okf/internal/okf/frontmatter.go`
- Create: `okf/internal/okf/bundle.go`
- Create: `okf/internal/okf/index.go`
- Create: `okf/internal/okf/log.go`
- Create: `okf/internal/okf/remind.go`
- Test: `okf/internal/okf/frontmatter_test.go`
- Test: `okf/internal/okf/index_test.go`
- Test: `okf/internal/okf/log_test.go`
- Test: `okf/internal/okf/remind_test.go`

**Interfaces:**
- Produces: `type Bundle struct { Root string; Version string }`.
- Produces: `func LoadBundle(root string) (*Bundle, error)`.
- Produces: `func parseFrontmatter(raw []byte) (map[string]string, bool)`.
- Produces: `func compileGlob(pattern string) *regexp.Regexp`.
- Produces: `func RenderIndex(root string) ([]byte, error)`.
- Produces: `func RenderLog(root, action, docPath, summary string, now time.Time) ([]byte, error)`.
- Produces: `func Reminder(projectDir string) (string, bool, error)`.
- Produces: `const ReminderText` matching the shell message without its trailing newline; `Run` writes exactly one newline.

- [ ] **Step 1: Read the Go guidelines**

Run the Modern Go Guidelines CLI with `--go-version 1.24` for each new Go file before editing it. Apply every relevant guideline.

- [ ] **Step 2: Write parser and glob tests**

```go
func TestParseFrontmatterScalars(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{"plain", "---\ntitle: Alpha\ndescription: First # note\n---\n", map[string]string{"title": "Alpha", "description": "First"}},
		{"single quoted", "---\ntitle: 'It''s alpha'\n---\n", map[string]string{"title": "It's alpha"}},
		{"double quoted", "---\ntitle: \"Alpha\\nBeta\"\n---\n", map[string]string{"title": "Alpha\nBeta"}},
		{"literal", "---\ndescription: |\n  Alpha\n  Beta\n---\n", map[string]string{"description": "Alpha\nBeta\n"}},
		{"folded", "---\ndescription: >\n  Alpha\n  Beta\n---\n", map[string]string{"description": "Alpha Beta\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseFrontmatter([]byte(tc.raw))
			if !ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, %v; want %#v", got, ok, tc.want)
			}
		})
	}
}

func TestCompileGlobUnmatchedBracketIsLiteral(t *testing.T) {
	re := compileGlob("draft[.md")
	if !re.MatchString("draft[.md") || re.MatchString("draftx.md") {
		t.Fatal("unmatched bracket must be literal")
	}
}
```

- [ ] **Step 3: Verify parser tests fail**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./internal/okf -run 'TestParseFrontmatterScalars|TestCompileGlob'`

Expected: FAIL because the parser and glob compiler do not exist.

- [ ] **Step 4: Implement `LoadBundle` and frontmatter parsing**

Implement `Bundle`, `LoadBundle`, `parseFrontmatter`, `okfVersion`, scalar decoding, line folding, and `compileGlob` in the named files. Port the Perl semantics pinned by the spec and fixtures. Implement `materializeFixture` by copying the input tree and replacing `{{LOCAL_DATE}}` with `now.Format("2006-01-02")` in every copied text file.

- [ ] **Step 5: Implement `RenderIndex`**

Implement bundle walking, exclusions, `.okfignore`, classification, section grouping, deterministic sorting, and exact Markdown rendering. Return bytes without writing files.

- [ ] **Step 6: Implement `RenderLog`**

Implement action validation, path normalization and rejection, document existence checks, single-line summary validation, local-date heading insertion, and exact log rendering. Trim leading `/` first, matching Perl. Then normalize accepted paths to slash form and reject `filepath.IsAbs`, non-empty `filepath.VolumeName`, and `..` segments. Return bytes without writing files.

- [ ] **Step 7: Implement `Reminder`**

Detect `docs/index.md`, require `okf_version` inside leading frontmatter, return `ReminderText` and true for a bundle, and return empty/false for a missing bundle. Return an error for unreadable files.

- [ ] **Step 8: Verify renderer tests pass**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./internal/okf`

Expected: PASS, including all golden index and log cases.

---

### Task 3: Implement Locking, Atomic Writes, and CLI

**Risk:** high — concurrency, persistence, Windows APIs, and public command behavior.

**Files:**
- Create: `okf/internal/filelock/lock.go`
- Create: `okf/internal/filelock/atomic.go`
- Create: `okf/internal/filelock/lock_unix.go`
- Create: `okf/internal/filelock/lock_windows.go`
- Create: `okf/internal/filelock/replace_unix.go`
- Create: `okf/internal/filelock/replace_windows.go`
- Create: `okf/internal/okf/update.go`
- Create: `okf/internal/okf/cli.go`
- Create: `okf/cmd/okf/main.go`
- Test: `okf/internal/filelock/lock_test.go`
- Test: `okf/internal/filelock/atomic_test.go`
- Test: `okf/internal/okf/update_test.go`
- Test: `okf/internal/okf/cli_test.go`

**Interfaces:**
- Produces: `func WithFileLock(path string, timeout time.Duration, fn func() error) error`.
- Produces: platform functions `tryLockFile(*os.File) error` and `unlockFile(*os.File) error`.
- Produces: `func WriteFileAtomic(target string, data []byte) error`.
- Produces: unexported `writeFileAtomic(target string, data []byte, replace func(string, string) error) error` for deterministic failure tests.
- Produces: platform function `replaceFile(oldPath, newPath string) error`.
- Produces: `func RegenerateIndex(root string) error`.
- Produces: `func LogAndIndex(root, action, docPath, summary string, now time.Time) error`.
- Produces: `func Run(args []string, stdout, stderr io.Writer) int`.
- Produces: unexported `run(args []string, stdout, stderr io.Writer, now time.Time) int` for deterministic CLI tests.

- [ ] **Step 1: Read the Go guidelines**

Run the Modern Go Guidelines CLI for Go 1.24 on all new Go files.

- [ ] **Step 2: Write lock and atomicity tests**

```go
func TestWithFileLockSerializes(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".okf.lock")
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithFileLock(path, time.Second, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	started := time.Now()
	time.AfterFunc(150*time.Millisecond, func() { close(release) })
	err := WithFileLock(path, 2*time.Second, func() error { return nil })
	if err != nil || time.Since(started) < 50*time.Millisecond {
		t.Fatalf("second lock did not wait: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWriteFileAtomicFailurePreservesTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "index.md")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomic(target, []byte("new"), func(string, string) error {
		return errors.New("injected replacement failure")
	})
	if err == nil {
		t.Fatal("expected replacement failure")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "old" {
		t.Fatalf("target changed: %q, %v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp.*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, %v", matches, err)
	}
}
```

- [ ] **Step 3: Verify lock tests fail**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./internal/filelock`

Expected: FAIL because the filelock package does not exist.

- [ ] **Step 4: Implement cross-platform locking**

Implement `WithFileLock` with a 100 ms retry loop and a `0644` lock file. Use `syscall.Flock` on Unix. On Windows, call `LockFileEx(handle, 0x2|0x1, 0, 1, 0, &overlapped)` and `UnlockFileEx(handle, 0, 1, 0, &overlapped)` through `syscall.NewLazyDLL("kernel32.dll")` with a zero `syscall.Overlapped`. Retry only `EAGAIN`/`EWOULDBLOCK` on Unix and Windows error 33 (`ERROR_LOCK_VIOLATION`).

- [ ] **Step 5: Implement atomic replacement**

Implement `WriteFileAtomic` as a wrapper around `writeFileAtomic` with the platform `replaceFile` function. The shared helper uses `os.CreateTemp` with pattern `.<base>.tmp.*` in the target directory, `0600` permissions, write, `Sync`, close, replacement, and cleanup on failure. Use `os.Rename` on Unix. On Windows, convert both paths with `syscall.UTF16PtrFromString` and call `MoveFileEx(from, to, 0x1|0x8)` through `kernel32.dll`.

- [ ] **Step 6: Write update and CLI tests**

```go
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

func TestRunGolden(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"index-mixed", "index-ignores", "log-new", "log-existing-today", "log-existing-older", "log-deletion"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(materializeFixture(t, name, now), "docs")
			args := append([]string{root}, readCaseArgs(t, name)...)
			var stdout, stderr bytes.Buffer
			code := run(args, &stdout, &stderr, now)
			wantCode, err := strconv.Atoi(strings.TrimSpace(string(readExpected(t, name, "exit.txt"))))
			if err != nil {
				t.Fatal(err)
			}
			if code != wantCode || stdout.String() != string(readExpected(t, name, "stdout.txt")) || stderr.String() != string(readExpected(t, name, "stderr.txt")) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			assertExpectedTree(t, name, root)
		})
	}
}
```

Add table tests for usage errors, invalid actions, traversal paths, reserved paths, multiline summaries, missing documents, reminder output, and absent bundles.

- [ ] **Step 7: Verify update tests fail**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./internal/okf -run 'TestLogAndIndexGolden|TestRunGolden'`

Expected: FAIL because update and CLI functions do not exist.

- [ ] **Step 8: Implement update transactions**

Implement `RegenerateIndex` and `LogAndIndex`. Both acquire `.okf.lock`. `LogAndIndex` renders and replaces `log.md`, then renders and replaces `index.md` without releasing the lock. Do not implement multi-file rollback.

- [ ] **Step 9: Implement the CLI**

Implement `run` with the three command shapes, an injected clock, stdout success text, stderr errors, usage exit code 2, and runtime exit code 1. Resolve the reminder project from `CLAUDE_PROJECT_DIR`, falling back to `os.Getwd` only when the variable is unset. Implement `Run` as the `time.Now()` wrapper. Add `cmd/okf/main.go` as the process entry point.

- [ ] **Step 10: Verify transaction tests pass**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./...`

Expected: PASS.

---

### Task 4: Add Launcher, Checkout Attributes, and Binaries

**Risk:** high — shell/batch parsing, architecture selection, quoting, and release artifacts.

**Files:**
- Create: `.gitattributes`
- Create: `okf/scripts/okf.cmd`
- Create: `okf/scripts/build.sh`
- Create: `okf/scripts/launcher_test.go`
- Create: `okf/bin/darwin/amd64/okf`
- Create: `okf/bin/darwin/arm64/okf`
- Create: `okf/bin/linux/amd64/okf`
- Create: `okf/bin/linux/arm64/okf`
- Create: `okf/bin/windows/amd64/okf.exe`
- Create: `okf/bin/windows/arm64/okf.exe`

**Interfaces:**
- Consumes: compiled `cmd/okf` binary.
- Produces: `okf/scripts/okf.cmd` with identical CLI behavior on every host.
- Produces: `okf/scripts/build.sh` with no flags or optional behavior.
- Produces: test helpers `launcherCommand(launcher string, args ...string) *exec.Cmd`, `copyLauncher`, `copyHostBinary`, and `makeBundle`.

- [ ] **Step 1: Add checkout attributes**

Create root `.gitattributes` with:

```gitattributes
okf/scripts/okf.cmd text eol=lf
okf/scripts/build.sh text eol=lf
okf/testdata/perl-golden/** -text
okf/bin/** binary
```

- [ ] **Step 2: Write launcher integration tests**

```go
func TestLauncherPathsWithSpaces(t *testing.T) {
	plugin := filepath.Join(t.TempDir(), "plugin with spaces")
	copyLauncher(t, plugin)
	copyHostBinary(t, plugin)
	project := filepath.Join(t.TempDir(), "project with spaces")
	makeBundle(t, project)
	cmd := launcherCommand(filepath.Join(plugin, "scripts", "okf.cmd"), "remind")
	cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+project)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), ReminderText) {
		t.Fatalf("launcher failed: %v\n%s", err, out)
	}
}
```

Add `TestLauncherStructure`: assert the first line is exactly `: << 'CMDBLOCK'`, the file contains no CR bytes, the terminator line is exactly `CMDBLOCK`, and `exit /b` appears before it. Add `TestLauncherPropagatesExitCode` using invalid CLI arguments. On Unix, run the same integration case through both `bash -c` and `sh -c`.

Add `TestLauncherMissingBinaryFails`: copy only the launcher, run `remind`, and assert non-zero status plus a clear stderr message.

Add a test that forwards a summary containing spaces, double quotes, `&`, and Unicode through `log`, then verifies the resulting `log.md` text.

- [ ] **Step 3: Verify launcher tests fail**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./scripts`

Expected: FAIL because the launcher and host binary do not exist.

- [ ] **Step 4: Implement the polyglot launcher**

Create `okf/scripts/okf.cmd` with first line exactly `: << 'CMDBLOCK'`, no shebang or BOM, and LF endings. Put `@echo off`, binary selection and invocation with `%*`, and `exit /b %ERRORLEVEL%` before the `CMDBLOCK` terminator. After the terminator, implement the POSIX branch. Resolve the plugin root with `%~dp0..` on Windows and `CDPATH= cd -- "$(dirname -- "$0")/.."` on Unix. Select Windows from `%OS%`, otherwise map `uname -s`; map `%PROCESSOR_ARCHITECTURE%` or `uname -m` to `amd64` or `arm64`. Treat `MINGW*` and `MSYS*` as Windows in the POSIX branch. Append `.exe` on Windows. Use `exec "$bin" "$@"` on Unix and `"%bin%" %*` on Windows. Fail with a stderr message and non-zero status for unsupported platforms, architectures, or missing binaries.

Implement `launcherCommand` with `exec.Command("bash", "-c", shellQuote(launcher)+" "+shellQuoteArgs(args))` on Unix. On Windows, pass `cmd /C` one command string containing the quoted launcher and quoted arguments. This models the verified host executors rather than bare `os/exec`.

- [ ] **Step 5: Make the launcher executable**

Run: `chmod 0755 okf/scripts/okf.cmd`

- [ ] **Step 6: Implement the build script**

Create `okf/scripts/build.sh`. Export `CGO_ENABLED=0` and `GOTOOLCHAIN=go1.24.0`. Build all six targets with `-trimpath -buildvcs=false -ldflags="-s -w -buildid="`. Write binaries to the paths pinned by the spec.

- [ ] **Step 7: Build binaries and verify reproducibility**

Run: `sh okf/scripts/build.sh`

Record all six SHA-256 hashes, run the script again, and confirm every hash is unchanged.

Generate a temporary 1,200-document bundle. Time three Perl index runs and three Go index runs, then record the medians in the task report. Treat this as measurement, not a flaky pass threshold.

- [ ] **Step 8: Verify launcher tests pass**

Run: `cd okf && GOTOOLCHAIN=go1.24.0 go test ./...`

Expected: PASS on the host platform.

---

### Task 5: Create Cross-Platform CI

**Risk:** standard — new build and release validation workflow.

**Files:**
- Create: `.github/workflows/okf.yml`

**Interfaces:**
- Consumes: Go tests, launcher tests, build script, committed binaries, and plugin manifest.
- Produces: six native test jobs, one reproducibility job, and one plugin-validation job.

- [ ] **Step 1: Create the workflow skeleton**

Create `.github/workflows/okf.yml` with `pull_request` and `push` triggers for `main`.

- [ ] **Step 2: Add the native matrix**

Use these runner labels:

- `macos-15-intel`
- `macos-15`
- `ubuntu-24.04`
- `ubuntu-24.04-arm`
- `windows-2025`
- `windows-11-arm`

Each job checks out the repository, sets up Go 1.24.0, runs `go test ./...` and `go vet ./...` from `okf`, and runs the launcher integration tests.

- [ ] **Step 3: Add explicit Windows shell coverage**

On both Windows runners, create a temporary OKF bundle and run the launcher through `cmd.exe /C` and `powershell.exe -NoProfile -Command`. Assert both outputs contain the reminder text.

- [ ] **Step 4: Add binary drift detection**

Add an `ubuntu-24.04` job that runs `sh okf/scripts/build.sh`, then `git diff --exit-code -- okf/bin`. The job fails if any committed binary differs from a fresh build.

- [ ] **Step 5: Add plugin validation**

Add an `ubuntu-24.04` job that runs `npx --yes @anthropic-ai/claude-code plugin validate .`.

- [ ] **Step 6: Validate workflow syntax locally**

Run: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/okf.yml`

Expected: PASS with no findings.

---

### Task 6: Cut Over Hosts and Remove Perl

**Risk:** high — public plugin behavior, documentation, version identity, and script removal.

**Files:**
- Modify: `okf/hooks/hooks.json`
- Modify: `okf/skills/okf/SKILL.md`
- Modify: `okf/agents/context-writer.md`
- Modify: `okf/README.md`
- Modify: `CLAUDE.md`
- Modify: `okf/.claude-plugin/plugin.json`
- Modify: `.claude-plugin/marketplace.json`
- Delete: `okf/skills/okf/scripts/okf-update.pl`
- Delete: `okf/scripts/okf-reminder.sh`
- Create: `tests/test_okf_plugin.py`

**Interfaces:**
- Consumes: launcher, binaries, CI, and golden-proven CLI.
- Produces: OKF plugin version `0.3.0` with no Perl or POSIX-reminder runtime dependency.

- [ ] **Step 1: Re-read shared WIP files**

Read the current working-tree versions of `okf/README.md`, `okf/.claude-plugin/plugin.json`, `CLAUDE.md`, and `.claude-plugin/marketplace.json`. Preserve all unrelated hunks.

- [ ] **Step 2: Write failing packaging tests**

```python
def test_okf_uses_cross_platform_launcher(self):
    hooks = json.loads((ROOT / "okf/hooks/hooks.json").read_text())
    command = hooks["hooks"]["SessionStart"][0]["hooks"][0]["command"]
    self.assertIn('scripts/okf.cmd" remind', command)
    self.assertNotIn("sh ", command)

def test_okf_has_no_perl_runtime_references(self):
    for path in (
        ROOT / "okf/README.md",
        ROOT / "okf/skills/okf/SKILL.md",
        ROOT / "okf/agents/context-writer.md",
    ):
        self.assertNotIn("okf-update.pl", path.read_text())
    self.assertFalse((ROOT / "okf/skills/okf/scripts/okf-update.pl").exists())
    self.assertFalse((ROOT / "okf/scripts/okf-reminder.sh").exists())

def test_okf_versions_match(self):
    plugin = json.loads((ROOT / "okf/.claude-plugin/plugin.json").read_text())
    market = json.loads((ROOT / ".claude-plugin/marketplace.json").read_text())
    entry = next(p for p in market["plugins"] if p["name"] == "okf")
    self.assertEqual(plugin["version"], "0.3.0")
    self.assertEqual(entry["version"], plugin["version"])

def test_okf_hook_runs_on_host(self):
    hooks = json.loads((ROOT / "okf/hooks/hooks.json").read_text())
    command = hooks["hooks"]["SessionStart"][0]["hooks"][0]["command"].replace(
        "${CLAUDE_PLUGIN_ROOT}", str(ROOT / "okf")
    )
    with tempfile.TemporaryDirectory() as project:
        docs = Path(project) / "docs"
        docs.mkdir()
        (docs / "index.md").write_text("---\nokf_version: \"0.1\"\n---\n")
        env = dict(os.environ, CLAUDE_PROJECT_DIR=project)
        result = subprocess.run(command, shell=True, env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        self.assertIn("OKF knowledge bundle detected", result.stdout)
```

Set `sys.dont_write_bytecode = True`. Add assertions for all six binaries, `.gitattributes`, and executable launcher mode on Unix.

- [ ] **Step 3: Verify packaging tests fail**

Run: `python3 tests/test_okf_plugin.py`

Expected: FAIL because the hook, versions, and script removal are not updated.

- [ ] **Step 4: Switch the hook**

Set the SessionStart command to `"${CLAUDE_PLUGIN_ROOT}/scripts/okf.cmd" remind`. Preserve the `startup` matcher and five-second timeout.

- [ ] **Step 5: Move skill and agent commands**

Replace Perl invocations in `okf/skills/okf/SKILL.md` and `okf/agents/context-writer.md` with `${CLAUDE_PLUGIN_ROOT}/scripts/okf.cmd`. Preserve the lock, atomic-write, and no-hand-edit guidance.

- [ ] **Step 6: Update documentation**

Update `okf/README.md` and root `CLAUDE.md` with the Go CLI, polyglot launcher, six binaries, build command, platform support, and lock compatibility. Remove Perl-only claims.

- [ ] **Step 7: Bump versions**

Set both OKF manifests to `0.3.0` without reformatting unrelated peer changes.

- [ ] **Step 8: Remove legacy scripts**

Delete `okf-update.pl` and `okf-reminder.sh` only after Tasks 1–5 pass.

- [ ] **Step 9: Verify packaging tests pass**

Run: `python3 tests/test_okf_plugin.py`

Expected: PASS.

- [ ] **Step 10: Run full validation**

Run:

```sh
python3 context-status/bin/test_context_statusline.py
python3 context-status/scripts/test_ensure_statusline.py
python3 branch-indicator/scripts/test_branch_indicator.py
python3 tests/test_rule_plugins.py
python3 tests/test_finish_check.py
python3 tests/test_okf_plugin.py
python3 jev-guard/scripts/test_content_guard.py
python3 -m py_compile jev-guard/scripts/content_guard.py
cd okf && GOTOOLCHAIN=go1.24.0 go test ./... && GOTOOLCHAIN=go1.24.0 go vet ./...
cd .. && claude plugin validate .
git diff --check
```

Expected: every command passes. Remove any test-created `__pycache__` directories before committing.
