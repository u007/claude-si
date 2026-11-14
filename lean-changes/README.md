# lean-changes

Injects lean-change rules at session start: smallest complete change, reuse the codebase and its existing libraries before the standard library, no unrequested abstractions.

SessionStart hook (`startup|clear|compact`, not `resume`) prints `rules.md` as session context. Edit `rules.md` to change the rules. A missing file makes the hook exit non-zero.

The rules are adapted from [Ponytail](https://github.com/DietrichGebert/ponytail) v5.3.3 (MIT, Copyright Dietrich Gebert). Differences from the original:

- A library the project already uses for a job ranks above the standard library, for consistency.
- No persona, no levels, and no required closing line on every reply.
- Deleting code is limited to what the change itself makes unnecessary, to agree with `working-principles`.
- A loaded process skill (test-driven development, brainstorming, planning) wins where it asks for more.

Install:

```
/plugin install lean-changes@claude-si
```
