# communication-style

SessionStart hook that injects the Agent Communication & Style Standard. It combines ASD-STE100 structural limits, brevity, and plain-language-first explanations.

The structural rules began as a port of `ocode` (`internal/agent/prompt.go`). Local rules now add plain-language-first explanations.

The text lives in `standard.md`. Edit it there; the hook is `cat standard.md`, so a missing file makes it exit non-zero. Works unchanged in ocode, which runs the same SessionStart hook with empty stdin.

Explanations stop at outcome, cause, and impact unless detail is asked for or needed to act, verify, diagnose, or run safely.

```
/plugin install communication-style@claude-si
```

Test: `python3 tests/test_rule_plugins.py`

Precedence: the standard itself says explicit user instructions and project docs override it. Your `~/.claude/CLAUDE.md` rules (caveman mode, Flow Context, ULTRATHINK) therefore win on conflict.

To prefer this standard over caveman, remove the "Communication: Caveman Mode" section from `~/.claude/CLAUDE.md`. To turn this plugin off, run `/plugin disable communication-style@claude-si`.

Injected on `startup`, `clear` and `compact`. Skipped on `resume`, where the earlier context is restored.
