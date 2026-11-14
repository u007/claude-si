# review-changes

Deep review of the active change set via the code-reviewer subagent, then issue-by-issue resolution with the user. Covers correctness, contracts, security, production gotchas, tests, DRY and lean code.

One command, `commands/review-changes.md`. Run it as `/review-changes [scope or context]`. With no argument it reviews staged plus unstaged changes, or the commits ahead of the default branch when the tree is clean.

The main agent gathers context and dispatches the `code-reviewer` subagent with the Part B briefing (it reviews small low-risk diffs inline, and splits very large ones across parallel reviewers). It then prints the report and walks each Critical and Important finding with you: fix, fix differently, ignore with a reason, or skip.

This plugin replaces the user-level `~/.claude/commands/review-changes.md`. Remove that file after you install the plugin, or the command exists twice.

In ocode the command loads as a slash command. An ocode build older than the `$ARGUMENTS` support drops the argument; pass the scope in a follow-up message there.

Install:

```
/plugin install review-changes@claude-si
```
