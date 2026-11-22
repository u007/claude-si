---
name: finish-check
description: Run the read-only finish check before the final response on long or multi-file tasks. Use when the task changed several files, started background work or servers, or when a build is about to be reported complete.
---

# Finish Check

Run the bundled script against the current project directory:

```sh
sh "${CLAUDE_PLUGIN_ROOT}/scripts/finish-check.sh"
```

Report its four sections to the user. Exit 0 means clean and every check ran. Exit 1 means review the listed items, or a scan could not run to completion (the output says why). Exit 2 means the path argument was invalid.

The script only reads. It never changes, deletes, or kills anything. The marker scan covers source and config files; prose (`*.md`, `*.txt`) is skipped because its ordinary wording trips the patterns, and tracked deferrals in `TODO.md` are excluded — that file is the record of incomplete work required by the code-quality rules.

The process scan may not see background work started in another directory. Check yourself that no agents, builds or servers you started are still running, and that no README claims were left unverified.
