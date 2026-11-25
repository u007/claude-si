# okf

Claude Code plugin for working with an OKF (Open Knowledge Format) v0.1 knowledge bundle: a `docs/` directory of markdown files with YAML frontmatter, a marked `docs/index.md` and an append-only `docs/log.md`.

- **Skill `okf`:** the workflow. Look up the bundle before any task, write docs only for durable knowledge, deduplicate first, prune the sections a change makes false, keep `index.md` and `log.md` in sync, deprecate a whole doc instead of deleting it.
- **SessionStart hook:** runs `scripts/okf.cmd remind`. When the project has `docs/index.md` with `okf_version`, it tells Claude to dispatch the `context` agent for the lookup before any task. Silent when there is no bundle. The hook reads no stdin, so it works where the payload is empty.

- **Agents `context` and `context-writer`:** Haiku subagents (`okf:context`, `okf:context-writer`) shipped with the plugin. `context` is read-only (no write tools) and returns a lookup digest before a task; `context-writer` updates docs (pruning stale sections), log and index after a task with durable knowledge. The main agent decides what is durable, briefs the writer, and checks `git diff docs/` afterwards. If the agents are unavailable, the skill is followed directly.

- **Binary and launcher:** a Go CLI (`okf`, source in `cmd/okf` and `internal/`) built into `bin/<os>/<arch>/okf` (`okf.exe` on Windows) for darwin, linux and windows on amd64 and arm64. `scripts/okf.cmd` is a polyglot launcher: `sh` runs it on macOS and Linux, `cmd` on Windows. It picks the binary for the platform, and the hook, the skill and the agents all call it. Subcommands: `remind` (hook), `<docs-dir> index`, and `<docs-dir> log <Action> <doc-path> <summary>`. Writes hold an exclusive lock on `docs/.okf.lock` and replace files atomically (temp file, then rename).
- **Original Perl script `skills/okf/scripts/okf-update.pl`:** no longer called. It stays in the repo as the reference that the golden fixtures in `testdata/perl-golden` were captured from; no test runs it.

The skill decides what to write; Claude writes the docs, and the launcher keeps the log and index in sync. The hook only makes the lookup reminder reliable.

In ocode, the okf:context and okf:context-writer agents are eligible for the configured context/small model, so lookups and doc writes run on a cheap model instead of the main one. In Claude Code they run on Haiku via the model: haiku frontmatter.

## Install

```sh
claude plugin marketplace add u007/claude-plugins
claude plugin install okf@claude-si
```

Then remove any standalone copy at `~/.claude/skills/okf` so the skill is not listed twice.

## Build

```sh
sh okf/scripts/build.sh
```

Cross-compiles all six targets into `okf/bin/` with `CGO_ENABLED=0`. The script requires Go 1.24.0 exactly and refuses any other version. The built `bin/` must be committed for the plugin to work after install, because installs do not run the build.

## Tests

```sh
sh okf/scripts/build.sh     # launcher tests run bin/ and need it built first
cd okf && go test ./...
```

`scripts/launcher_test.go` runs `okf.cmd` against the host binary in `bin/`. `golden_test.go` checks index and log output against the fixtures in `testdata/perl-golden`.

Test without installing:

```sh
claude --plugin-dir ./okf
```

Not yet verified end to end in the desktop app.
