# okf

Claude Code plugin for working with an OKF (Open Knowledge Format) v0.1 knowledge bundle: a `docs/` directory of markdown files with YAML frontmatter, a marked `docs/index.md` and an append-only `docs/log.md`.

- **Skill `okf`:** the workflow. Look up the bundle before any task, write docs only for durable knowledge, deduplicate first, prune the sections a change makes false, keep `index.md` and `log.md` in sync, deprecate a whole doc instead of deleting it.
- **SessionStart hook:** when the project has `docs/index.md` with `okf_version`, tells Claude to dispatch the `context` agent for the lookup before any task. Silent when there is no bundle. POSIX `sh` only, so it works in the desktop app, which does not load shell profiles.

- **Agents `context` and `context-writer`:** Haiku subagents (`okf:context`, `okf:context-writer`) shipped with the plugin. `context` is read-only (no write tools) and returns a lookup digest before a task; `context-writer` updates docs (pruning stale sections), log and index after a task with durable knowledge. The main agent decides what is durable, briefs the writer, and checks `git diff docs/` afterwards. If the agents are unavailable, the skill is followed directly.

- **Script `skills/okf/scripts/okf-update.pl`:** appends the log entry and regenerates `index.md` under the `docs/.okf.lock` flock, in one call (well under a second on a 1,200-file bundle). Its index has the same lines as ocode's `knowledge.GenerateIndex`; only entries with tied titles may order differently. Perl core only.

The skill decides what to write; Claude writes the docs, and the script keeps the log and index in sync. The hook only makes the lookup reminder reliable.

## Install

```sh
claude plugin marketplace add u007/claude-plugins
claude plugin install okf@claude-si
```

Then remove any standalone copy at `~/.claude/skills/okf` so the skill is not listed twice.

Test without installing:

```sh
claude --plugin-dir ./okf
```

Not yet verified end to end in the desktop app.
