# code-quality

Injects code quality rules at session start: no fallbacks, error logging, no upsert, TODO tracking, test safety.

SessionStart hook (`startup|clear|compact`, not `resume`) prints `rules.md` as session context. Edit `rules.md` to change the rules. A missing file makes the hook exit non-zero.

Install:

```
/plugin install code-quality --marketplace u007/claude-plugins
```

## Finish check

Run `/code-quality:finish-check` to scan the current project before the final response. It is a plugin skill, not a hook.

The script at `scripts/finish-check.sh` is POSIX `sh` and read-only. It reports four sections:

1. Placeholder markers in source and config files — `TODO`, `TBD`, `FIXME`, `XXX`, `@@...@@`, `{{...}}`, "coming soon". Prose (`*.md`, `*.txt`) is not scanned because its ordinary wording trips the patterns; `TODO.md` is also skipped because that file is the tracked record of incomplete work required by the rules. The match is literal, so a file that documents these markers (including this plugin's own script and tests) is reported.
2. World-readable files that look like secrets or databases. Templates such as `.env.example`, `.env.sample` and `.env.template` are excluded.
3. Processes whose working directory is inside the project.
4. Listening TCP ports from those processes.

When part of the tree cannot be read, or the process scan is unavailable, the script prints a `WARNING` and exits 1 instead of claiming clean.

Exit status: 0 = clean and every check ran, 1 = review the listed items or could not verify, 2 = invalid path argument. The script never changes, deletes, or kills anything.

You can also run it directly:

```sh
sh "${CLAUDE_PLUGIN_ROOT}/scripts/finish-check.sh" path/to/project
```

The finish check is adapted from [Fable-Cheap](https://github.com/aicodeking/Fable-Cheap) by AICodeKing (MIT, Copyright 2026 AICodeKing). Changes: generic wording, and the `TODO.md` skip. The script itself cannot see background work started in another directory, so verify that yourself.
