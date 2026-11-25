# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A Claude Code plugin marketplace (`claude-si`) holding eleven plugins (`context-status`, `jev-guard`, `okf`, `branch-indicator`, `communication-style`, `working-principles`, `code-quality`, `git-safety`, `frontend-standards`, `lean-changes`, `review-changes`). The marketplace manifest is `.claude-plugin/marketplace.json`; each plugin has its own `.claude-plugin/plugin.json`. Plugin `source` paths in the marketplace are relative (`./context-status`, `./jev-guard`).

## Commands

```sh
# Status line tests (stdlib unittest, no install step)
python3 context-status/bin/test_context_statusline.py
python3 context-status/scripts/test_ensure_statusline.py

# branch-indicator tests
python3 branch-indicator/scripts/test_branch_indicator.py

# okf: build the Go binary for every os/arch into okf/bin/ (needs go1.24.0, refuses other versions)
sh okf/scripts/build.sh

# okf Go tests (scripts/launcher_test.go runs bin/, so build first)
cd okf && go test ./...

# Single test: ClassName.method_name
python3 context-status/bin/test_context_statusline.py ModelSegment.test_shows_model_display_name

# Validate marketplace + every plugin manifest (exit 1 on errors; add --strict for CI)
claude plugin validate .

# rule-pack plugins (communication-style, working-principles, code-quality, git-safety, frontend-standards, lean-changes)
python3 tests/test_rule_plugins.py

# finish-check script (read-only, no network)
python3 tests/test_finish_check.py

# jev-guard tests (judge is stubbed, no network)
python3 jev-guard/scripts/test_content_guard.py
python3 -m py_compile jev-guard/scripts/content_guard.py
```

Delete any `__pycache__` the test run leaves behind. It is gitignored, but do not commit it.

## Architecture

### context-status: statusLine renderer

- Registered in the user's `~/.claude/settings.json` as `statusLine.command`, pointing at `bin/context-statusline.sh`. Plugins cannot declare `statusLine`, so a SessionStart hook (`hooks/hooks.json` -> `scripts/ensure_statusline.py`) writes it: sets it when absent, repoints a stale context-status path, leaves any other `statusLine` alone. Invalid settings JSON raises.
- Claude Code pipes a JSON payload on stdin. `context_statusline_render.py` reads only `model`, `workspace.current_dir` and `context_window`, and builds one line: model · dir · branch · context bar · chained output.
- Every segment is best-effort. Missing data drops the segment rather than rendering a placeholder. Keep it that way when adding segments.
- The branch is read from the repo's `HEAD` file, not by spawning `git`. This is deliberate: the status line runs on every refresh and must not spawn processes or block on a hung `git`. `_git_dir` handles normal repos, linked worktrees, submodules and bare repos.
- `bin/prev_command.txt` is a shell command whose output is appended to the line (`shell=True`, with a 5s timeout). It preserves a previously configured status line. Blanking the file disables chaining. Its path is resolved via `__file__`, so it works through the `~/.claude/plugins/context-status` symlink.

### jev-guard: PostToolUse inbound-content guardrail

- `hooks/hooks.json` registers one PostToolUse hook, matching `WebFetch|WebSearch|Bash|mcp__.*`, that runs `scripts/content_guard.py`.
- `source_for()` decides what gets scanned. Bash is scanned only when the command invokes a network binary (`curl`, `wget`, `nc`, ...) against a non-loopback host. Local Bash and Read output is never scanned.
- The result text is split into 6000-char chunks (first 32 scanned; a larger result is truncated to the scanned prefix), and the chunks are judged in parallel against the TypeSafe Jev API.
- Three outcomes:
  - **Redirect:** WebFetch's own 3xx `REDIRECT DETECTED` notice is skipped (the judge flags its "use WebFetch again" wording as tool coercion); the model is told it is unverified.
  - **Truncated:** more than 32 chunks; only the scanned prefix is delivered, with a note.
  - **Flagged** (also a clean verdict below 0.6 confidence, with a distinct notice): `decision: block` plus `updatedToolOutput` with the same response shape and a withheld notice.
  - **Unvetted** (no `TYPESAFE_API_KEY`, or judge error or timeout): the result is delivered, and the model is told it is unvetted. Failures are never silent.
  - **Clean:** no output.
- `TYPESAFE_API_KEY` is read from the environment only. `.env.example` holds a placeholder. Never add a real key to the repo.

### branch-indicator: worktree/branch announcer

- `hooks/hooks.json` runs `scripts/branch_indicator.py` on SessionStart and on PostToolUse for `Bash|EnterWorktree|ExitWorktree`. It reads `cwd` from the hook input, resolves the git dir and reads `HEAD` directly (no `git` spawn). A linked worktree is a git dir under `.git/worktrees/<name>`.
- Output is `systemMessage` (shown to the user) plus `additionalContext` (for Claude). PostToolUse is silent unless the location changed since the last announcement, tracked per session in a temp file.
- The git-dir/HEAD parsing duplicates `context-status`; plugins install independently so they cannot share code. Keep the two in step.

### okf: knowledge bundle CLI

- Go CLI in `okf/cmd/okf` and `okf/internal/okf`, built by `okf/scripts/build.sh` into `okf/bin/<os>/<arch>/okf[.exe]` (six targets, `CGO_ENABLED=0`). `bin/` must be committed before release, because installs do not build.
- `okf/scripts/okf.cmd` is a polyglot launcher (`sh` on Unix, `cmd` on Windows) that picks the binary. `hooks/hooks.json` (SessionStart, `startup` only) runs `"${CLAUDE_PLUGIN_ROOT}/scripts/okf.cmd" remind`. The skill and `context-writer` call `docs log` and `docs index` through the launcher at `<skill-base-dir>/../../scripts/okf.cmd`.
- `remind` reads no stdin. A missing or unmarked `docs/index.md` is silent. An unreadable `index.md` or `docs` exits 1 with a message, where the old shell hook exited 0 silently. CRLF files are accepted, which the old awk rejected. Like the awk, `okf_version:` must start at column 0. Keep `ReminderText` in step with `scripts/okf-reminder.sh`: `remind_test.go` reads that file, so do not delete it.
- Writes take an exclusive lock on `docs/.okf.lock` (flock on Unix, LockFileEx on Windows), then replace `log.md` and `index.md` atomically.
- `skills/okf/scripts/okf-update.pl` is no longer called. Only the golden fixtures in `okf/testdata/perl-golden` trace back to it (captured at commit `f74ffe0`; regenerate only with Perl on that commit). No test runs the script.

### Rule-pack plugins: communication-style, working-principles, code-quality, git-safety, frontend-standards, lean-changes

- Each holds one markdown file (`rules.md`; `standard.md` for communication-style, ASD-STE100 limits plus software overrides and plain-language-first rules, derived from ocode `internal/agent/prompt.go`) and a SessionStart hook (matcher `startup|clear|compact`, so not `resume`) whose command is `cat "${CLAUDE_PLUGIN_ROOT}/<file>"`. Stdout becomes session context. A missing file makes the hook exit non-zero (Claude Code shows a non-blocking error; ocode ignores it). No hook script, so nothing is duplicated across plugins. `code-quality` also ships a read-only script invoked by its `/code-quality:finish-check` skill, not by the hook.
- No Python hook script on purpose: ocode runs Claude Code SessionStart hooks with empty stdin and drops hook failures silently (ocode repo, `internal/plugins/sessionstart.go`: `cmd.Stdin = nil`, error returns `""`), so a script that reads the hook payload crashes there unnoticed. Keep these hooks stdin-independent.
- Split from the user's former global `~/.claude/CLAUDE.md`. Users opt out of a theme by not installing its plugin.
- Brevity and ASD-STE100 style live in `communication-style`; the other packs carry no style rules.
- `lean-changes` is adapted from Ponytail (MIT; attribution in its `rules.md` and README). Its delete rule is limited to what the change itself makes unnecessary, so it agrees with the Surgical Changes rule in `working-principles`. Keep the two in step.

### code-quality: finish-check skill

- `skills/finish-check/SKILL.md` runs `scripts/finish-check.sh` on the current directory. It is manual; no hook runs it, so hook failures cannot go unnoticed. Claude Code exposes it as `/code-quality:finish-check`.
- Use a skill, not a command: ocode expands `${CLAUDE_PLUGIN_ROOT}` in skills (`internal/skill/loader.go`) but not in command markdown (`internal/commands/loader.go`). Current Claude Code expands it in both.
- The script is POSIX `sh` and read-only. It reports placeholder markers in source/config files (`*.md`/`*.txt` and `TODO.md` are skipped so prose does not trip the patterns), world-readable secret/DB files (templates such as `.env.example` are excluded), and processes plus listening ports owned by the project. Exit 0 means clean and every check ran; 1 means review or could not verify; 2 usage error.
- Adapted from AICodeKing's Fable-Cheap (MIT; attribution in the script header and `code-quality/README.md`). Keep the `TODO.md` skip in step with the rule that tracked deferrals go in `TODO.md`.

### review-changes: review command

- One file, `commands/review-changes.md`: Part A is the main agent's orchestration and interactive resolution, Part B is the briefing passed to the `code-reviewer` subagent. No hooks and no scripts.
- It is a command, not a skill, so ocode loads it too (ocode loads plugin `commands/*.md`). ocode substitutes `$ARGUMENTS` like its own `{{args}}`; an older ocode build drops the argument.
- Its lean checks (Phase 3, DRY section, Evidence Standard) state the review side of the `lean-changes` rules. When one changes, check the other.

## Conventions specific to this repo

- The `plugin.json` and the marketplace entry must agree on name and version. `claude plugin tag` checks this.
- Plugin folder names are part of the install identity (`jev-guard@claude-si`). Do not rename them.
- Keep the marketplace name `claude-si` (renamed from `personal-plugins`). It is referenced from user settings. Installs recorded as `@personal-plugins` do not migrate on their own, so the root README keeps an upgrade note. Do not rename the marketplace again without updating it.
- Do not commit personal paths or usernames. Use `$HOME` or `~` in shipped config and fixtures.
