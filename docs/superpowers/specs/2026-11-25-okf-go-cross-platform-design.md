# OKF Go Cross-Platform Design

**Date:** 2026-11-25

**Status:** Approved in chat; self-reviewed; pending written-spec approval
**Owners:** claude-si maintainers

## Goal

Replace the OKF Perl updater with a standalone Go CLI that works natively on macOS, Linux, and Windows for both Claude Code and ocode, without requiring Perl, Python, Node.js, Go, or Git Bash at runtime.

## Scope

### Included

- Bundle detection and the SessionStart reminder.
- `index.md` regeneration.
- `log.md` insertion plus index regeneration under one lock.
- A thin launcher shared by Claude Code and ocode.
- Prebuilt binaries for amd64 and arm64 on macOS, Linux, and Windows.
- Golden parity tests against the current Perl implementation.
- Documentation, manifest, marketplace, and CI updates.

### Excluded

- Changes to the OKF v0.1 schema or workflow.
- Claude Code mods or any Claude Code version floor.
- Runtime installation or compilation of Go source.
- A PowerShell-only or POSIX-shell-only implementation.
- Changes to unrelated plugins or pre-existing uncommitted work.

## Constraints

- Native Windows must work without Git Bash.
- Both hosts must invoke the same reminder command.
- The launcher must resolve paths from its own location, not the current directory.
- Unsupported platforms and architectures must fail with a clear non-zero error.
- OKF logic must have one implementation; platform code may differ only for locking and file replacement.
- The existing `.okf.lock` protocol must remain compatible with ocode.
- The polyglot launcher must keep LF endings in every checkout.
- Golden fixtures must be committed so Windows CI never needs Perl.
- Perl remains available until golden-output parity is proven.
- The checkout has no OKF bundle, so this task does not mutate `docs/index.md` or `docs/log.md` as bundle data.

## Current Behavior

- `okf/skills/okf/scripts/okf-update.pl` uses Perl core modules and `flock(2)`.
- `okf/scripts/okf-reminder.sh` uses POSIX `sh` and `awk`.
- `okf/hooks/hooks.json` invokes the reminder through `sh`.
- The updater appends one log entry, regenerates the index, and writes both atomically.
- The README reports under one second for a 1,200-file bundle.
- Native Windows does not guarantee Perl, `sh`, `awk`, or `flock`.

## Decision

Use one dependency-free Go CLI and one polyglot launcher.

The launcher is a `.cmd` file that is also a valid POSIX shell script. Its first line is exactly `: << 'CMDBLOCK'`, with no shebang or BOM. Unix shells execute the POSIX branch through their `ENOEXEC` fallback; Windows `cmd.exe` and PowerShell execute the batch branch. It selects a prebuilt binary from the plugin directory and forwards arguments unchanged.

This avoids a Claude Code mod, avoids duplicate reminder logic, and gives ocode the same native Windows path. The unusual launcher syntax is acceptable only with native CI coverage on all three operating systems.

## Architecture

### Go module

- Create `okf/go.mod` with module path `github.com/u007/claude-plugins/okf`.
- Target Go 1.24 or newer.
- Use only the Go standard library.
- Keep `cmd/okf` thin; place behavior in focused internal packages.

### CLI contract

The launcher exposes the existing updater command shapes plus a reminder subcommand:

- `okf.cmd remind`
- `okf.cmd <docs-dir> index`
- `okf.cmd <docs-dir> log <Action> <doc-path> <summary>`

`Action` is one of `Creation`, `Update`, `Deprecation`, or `Deletion`.

The CLI writes normal success output to stdout and errors to stderr. Every failure returns non-zero. No fallback or silent retry is permitted.

### Launcher contract

- File: `okf/scripts/okf.cmd`.
- First line: exactly `: << 'CMDBLOCK'`, with no shebang, BOM, or CR bytes.
- Unix branch: executable mode `0755`, LF line endings, invoked through `bash -c`.
- Windows branch: valid `cmd.exe` batch syntax inside the heredoc block.
- Batch branch: `@echo off`, binary invocation with `%*`, then `exit /b %ERRORLEVEL%` before `CMDBLOCK`.
- Resolve the plugin root from the launcher location.
- Map macOS to `darwin`, Linux to `linux`, and Windows to `windows`.
- Map `x86_64` and `AMD64` to `amd64`.
- Map `aarch64` and `ARM64` to `arm64`.
- Select `okf/bin/<os>/<arch>/okf`, adding `.exe` on Windows.
- Preserve the child exit code, stdout, stderr, and argument boundaries.
- Fail clearly when the selected binary is missing.

The hook uses command-string form rather than exec form because ocode routes Windows command strings through `cmd.exe`, which can execute `.cmd` files directly.

The hook command is `"${CLAUDE_PLUGIN_ROOT}/scripts/okf.cmd" remind`. Preserve the current `startup` matcher and five-second timeout.

This contract was verified against ocode's `plugins.hookCmd`: command hooks without `args` use `cmd /C` on Windows and `bash -c` elsewhere. The lock contract was verified against `knowledge.WithBundleLock`, `knowledge.lockFileName`, and `filelock.tryLockFile`: the lock is `.okf.lock`, and Windows `LockFileEx` uses the exclusive and fail-immediately flags over one byte at offset zero.

### Binary layout

```text
okf/bin/darwin/amd64/okf
okf/bin/darwin/arm64/okf
okf/bin/linux/amd64/okf
okf/bin/linux/arm64/okf
okf/bin/windows/amd64/okf.exe
okf/bin/windows/arm64/okf.exe
```

Binaries are committed so plugin installation needs no build step or network access.

### Checkout attributes

- Create a root `.gitattributes`.
- Pin `okf/scripts/okf.cmd` to `text eol=lf`.
- Pin `okf/scripts/build.sh` to `text eol=lf`.
- Mark `okf/testdata/perl-golden/**` as `-text -whitespace` to preserve intentional CRLF and blank-at-EOF fixture bytes.
- Mark `okf/bin/**` as `binary`.
- Preserve executable mode for the launcher in the Git index.

## Behavior Parity

### Bundle detection and reminder

- Read `CLAUDE_PROJECT_DIR` when set; otherwise use the current working directory.
- Inspect `<project>/docs/index.md`.
- Recognize `okf_version` only inside leading YAML frontmatter.
- Accept LF and CRLF frontmatter delimiters.
- Print the current reminder text exactly when a bundle is present.
- Exit silently with status 0 when no bundle exists.
- Report unreadable files as errors rather than treating them as absent.

### Index regeneration

- Require a non-empty `okf_version` in the existing index frontmatter.
- Walk the bundle recursively.
- Skip hidden files and directories.
- Skip `index.md`, `log.md`, `PLAN-*.md`, and `*.OCODE.md`.
- Apply `.okfignore` patterns to basenames and bundle-relative slash paths.
- Treat a trailing `/` pattern as a directory exclusion.
- Use whole-string glob matching where `*` and `?` never cross `/`.
- Match the current Perl glob translation for malformed classes and escapes; an unmatched `[` is literal.
- Parse only the top-level scalar fields needed for indexing.
- Support plain, single-quoted, double-quoted, literal, and folded scalars.
- Treat flow sequence and mapping values as empty; block collections fold literally, matching the Perl script.
- Group conforming documents by top-level directory.
- Place root documents under `Concepts`.
- Sort sections by name.
- Sort entries by case-insensitive title, then path for deterministic ties.
- Use the path as the title when title is empty.
- Append ` (deprecated)` when status is `deprecated`.
- List non-conforming files under `Unclassified`, sorted by path.
- Emit LF line endings and the exact current Markdown shape.

### Log insertion

- Trim leading `/` from the document path.
- Reject absolute paths, `..` path segments, `index.md`, and `log.md`.
- Require the document to exist unless the action is `Deletion`.
- Require a single-line summary.
- Use the local date in `YYYY-MM-DD` form.
- Insert after today's heading, preserving newest-first ordering.
- Create the date heading near the top when absent.
- Create `log.md` with `# Directory Update Log` when absent.
- Regenerate the index in the same locked transaction.

## Locking and Atomicity

- Open `<docs-dir>/.okf.lock` with read-write create semantics.
- Acquire an exclusive advisory lock before log validation or writes.
- Retry every 100 ms for at most 60 seconds.
- Retry only lock-contention errors; fail immediately on other operating-system errors.
- Unix: use `syscall.Flock` with `LOCK_EX|LOCK_NB`.
- Windows: call `LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` over one byte at offset zero.
- Unlock and close on every return path.
- Write `index.md` and `log.md` through same-directory temporary files.
- Create temporary files with `0600` semantics, matching Perl and ocode.
- Sync and close temporary files before replacement.
- Unix: replace with `os.Rename`.
- Windows: replace with `MoveFileEx` using `MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH`.
- Remove temporary files after any failed write or replacement.
- Each failed replacement leaves that target file intact.
- Log and index updates are serialized, not rolled back as a multi-file transaction.

## File Layout

```text
okf/
  go.mod
  cmd/okf/main.go
  internal/okf/cli.go
  internal/okf/bundle.go
  internal/okf/frontmatter.go
  internal/okf/index.go
  internal/okf/log.go
  internal/okf/remind.go
  internal/filelock/lock.go
  internal/filelock/atomic.go
  internal/filelock/lock_unix.go
  internal/filelock/lock_windows.go
  internal/filelock/replace_unix.go
  internal/filelock/replace_windows.go
  scripts/okf.cmd
  scripts/build.sh
  bin/<os>/<arch>/okf[.exe]
.gitattributes
```

Tests live beside the package they exercise.

Commit golden fixtures under `okf/testdata/perl-golden/`. Generate them once on a host with Perl, then use them as read-only expectations on every platform.

## Build and Distribution

- Add `okf/scripts/build.sh` for maintainers.
- Build with `CGO_ENABLED=0`.
- Use `-trimpath`, `-buildvcs=false`, and `-ldflags="-s -w -buildid="`.
- Build all six targets from a clean checkout.
- Verify that a second build produces identical hashes.
- CI must fail if committed binaries differ from a fresh build.
- Keep the launcher free of OKF business logic.
- Create CI from scratch; this repository currently has no workflow directory.
- Pin one exact Go toolchain version in CI and document it for maintainers.

## Testing Strategy

### Golden parity

Before deleting Perl, create and commit fixture bundles that cover:

- Root and nested conforming documents.
- Missing, empty, quoted, and multiline titles and descriptions.
- Literal and folded scalars.
- Deprecated documents.
- Non-conforming documents.
- Hidden files and directories.
- Default exclusions and `.okfignore` file and directory patterns.
- LF and CRLF input.
- Tied titles and case-insensitive ordering.
- Existing and missing logs.
- Existing and missing date headings.
- Invalid actions, traversal paths, reserved paths, and multiline summaries.

Capture Perl stdout, stderr, exit codes, and resulting file bytes in `okf/testdata/perl-golden/`. Replace the local date in log fixtures with a `{{LOCAL_DATE}}` token so expectations remain valid over time. Tests substitute the current local date before comparison. The Go CLI must match successful output bytes exactly. Error text must remain clear but need not be byte-identical.

### Go tests

- Unit-test frontmatter parsing and scalar decoding.
- Unit-test ignore matching and document classification.
- Unit-test index rendering and log insertion.
- Integration-test the complete CLI against temporary bundles.
- Test reminder output with and without a bundle.
- Test lock contention and timeout behavior.
- Test failed writes leave no temporary files and preserve prior targets.
- Test traversal and reserved-path rejection.

### Launcher and host tests

- Execute the launcher directly on macOS and Linux.
- Execute it through `bash -c` to model ocode.
- Execute it through `cmd.exe` and PowerShell on Windows.
- Test plugin paths containing spaces.
- Test summaries containing spaces, quotes, ampersands, and Unicode.
- Verify child exit codes and stderr propagation.
- Verify unsupported-platform errors.

### CI

- Run `go test ./...` and `go vet ./...` on macOS, Linux, and Windows.
- Execute native binaries on amd64 and arm64 runners.
- Cross-build all six targets on every run.
- Run `claude plugin validate .`.
- Rebuild binaries and fail on hash drift.
- Run the launcher through both Windows shells.

## Rollout

1. Add characterization fixtures and capture Perl golden outputs.
2. Add the Go module, CLI, locking, and atomic replacement code.
3. Prove byte parity on the host platform.
4. Add `.gitattributes` before adding the launcher or binaries.
5. Add the polyglot launcher and maintainer build script.
6. Add native CI for all supported platforms and architectures.
7. Switch the hook, skill, writer agent, and README to the launcher.
8. Bump the OKF plugin and marketplace entry to `0.3.0`.
9. Delete `okf-update.pl` and `okf-reminder.sh` only after all checks pass.
10. Run the full repository test suite and plugin validation.

## Documentation Updates

- Update `okf/README.md` with runtime, platform, binary, and build details.
- Update `okf/skills/okf/SKILL.md` command examples.
- Update `okf/agents/context-writer.md`.
- Update root `CLAUDE.md` with the Go helper and launcher architecture.
- Keep the marketplace and plugin manifests aligned.
- Preserve existing uncommitted edits in `okf/README.md` and `okf/.claude-plugin/plugin.json`.

## Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Polyglot launcher parses differently across shells | Native CI through Bash, `cmd.exe`, and PowerShell invocation |
| A shebang pollutes Windows stderr | Use the no-shebang heredoc header and assert the exact first line |
| Windows rewrites launcher line endings | `.gitattributes` pins `okf/scripts/okf.cmd` to LF and marks binaries binary |
| Windows lock semantics differ from Unix | Mirror ocode's one-byte `LockFileEx` range and test contention on Windows |
| Windows replacement fails when the target exists | Use `MoveFileEx` with replace and write-through flags |
| Argument quoting changes summaries | Forward `"$@"` on Unix and `%*` on Windows; test special characters |
| Unicode sorting differs from Perl | Add Unicode fixtures and match the intended case-insensitive ordering |
| Committed binaries become stale | CI rebuilds all targets and fails on hash drift |
| Binary size increases repository size | Strip symbols and build IDs; accept the offline-install trade-off |
| Older Claude Code cannot use mods | No mods are used; the command hook remains compatible |

## Success Criteria

- `okf.cmd remind` works on macOS, Linux, and Windows in both hosts.
- `okf.cmd <docs-dir> index` matches Perl output bytes on golden fixtures.
- `okf.cmd <docs-dir> log ...` matches Perl output bytes on golden fixtures.
- Concurrent writers serialize through the shared `.okf.lock`.
- Failed writes preserve prior files and leave no temporary files.
- No runtime dependency beyond the operating system is required.
- All six binaries rebuild reproducibly.
- The full repository test suite and plugin validation pass.
- Perl and the POSIX reminder script are removed only after parity is proven.
