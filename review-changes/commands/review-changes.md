---
description: Deep review of the active change set (staged/unstaged/committed) via code-reviewer subagent, then interactive issue-by-issue resolution with the user
---

# Review Changes Workflow

This command has two parts with different executors:

- **Part A — Orchestration & Resolution**: run by YOU (the main agent). Subagents cannot talk to the user, so all interactive steps stay here.
- **Part B — Subagent Briefing**: delegate to the `code-reviewer` subagent via the Task tool. Pass everything under "Part B" as the task description. Do not perform the deep review yourself.

---

# Part A: Orchestration & Resolution (main agent)

## A1. Gather context

Arguments passed to this command: `$ARGUMENTS`

If the arguments (or surrounding conversation) provide context about the goal, purpose, a plan file, or an explicit review scope (e.g. `HEAD~3`, a branch name, specific paths), include it verbatim at the top of the subagent's task description under a `## Change Context` heading. An explicit scope overrides the default scope rule in Phase 1.

## A2. Dispatch

**Inline fast path:** check the change set first (`git status` + `git diff --stat` per the Phase 1 scope rule). If it is entirely low-risk per the Phase 1 blast-radius definitions (comments, copy, isolated styling, test-only, clearly local refactors — no shared exports, contracts, or high/medium-risk files) AND under ~50 diff lines, skip the subagent: perform the review yourself inline following Part B's rules and report format, then continue at A3. Any doubt about risk classification → dispatch the subagent.

Otherwise, check the size of the change set (`git diff --stat` for the scope determined by the Phase 1 rule). If it exceeds ~25 changed files or ~3000 diff lines, split it by subsystem/directory into 2–4 groups and dispatch one `code-reviewer` subagent per group in parallel (each briefing = full Part B plus a `## Scope` heading listing only that group's files); then merge the reports, renumbering findings sequentially across reports ordered by severity, and combining their Logic Changes lists into one. Otherwise spawn a single `code-reviewer` subagent with the full contents of Part B (plus Change Context if any). Wait for the report(s).

## A3. Relay the report

Print the subagent's report to the user unmodified.

## A4. Interactive Issue Resolution

If the report has no actionable issues, stop here — do not ask questions with no context.

If there are Critical or Important findings (test gaps arrive as numbered Important findings), walk them one at a time starting from issue 1 (the report already contains full details for each — do not reprint them in bulk).

If the `AskUserQuestion` tool is available, present each issue through it: question = `Issue [N/M] — <severity> — path/to/file.ts:42 — <one-line description>` plus the suggested fix, options = Fix it (place first and mark "(Recommended)" when that is the recommendation), Fix it differently, Ignore (with reason), Skip. Otherwise fall back to this text menu:

```
---
Issue [N/M] — <severity> — path/to/file.ts:42 — <one-line description>
Suggested fix: <from report>
Recommended: [Fix|Ignore|Skip] — <brief reason>

  [F] Fix it — apply the suggested fix
  [C] Fix it differently — describe how
  [I] Ignore — provide a reason (will be documented)
  [S] Skip — move on without action
```

`Recommended` guidance: `[Fix]` for bugs, security, breaking changes, rule violations. `[Ignore]` when valid but low-impact and documenting the reason suffices. `[Skip]` only for pure style preferences. (`[C]` is a user choice, never a recommendation.)

Wait for the user's response before moving to the next issue.

Suggestion-severity findings are not walked. After the Critical/Important walk, ask once whether the user wants to address any of the listed Suggestions; if yes, walk the chosen ones with the same menu.

**Handling responses:**

- **F**: Apply the suggested fix, then next issue.
- **C**: Apply the user's approach, confirm what was done, then next issue.
- **I**: Accept the reason. If it explains a deliberate design decision or non-obvious constraint, add a code comment at the relevant line (`// intentional: <reason>`). If it is a broader architectural decision or policy, update the nearest relevant doc. Confirm, then next issue.
- **S**: Next issue, no action.

**Plan gaps → TODO.md:** if the report lists Plan Gaps, ask the user once at the end of resolution whether to record them in `TODO.md`. Only on confirmation, append each as `- [ ] <gap> (from review-changes: <YYYY-MM-DD>)` (create the file if needed). Never write TODO.md without asking.

## A5. Verify and summarize

If any fixes were applied, run the project's fast check commands before summarizing — typecheck/lint plus the tests covering the changed files (use the project's documented commands; do not run the full suite if a targeted subset exists). If a check fails, report the failure and resolve it with the user before closing.

Then print a brief completion summary: issues fixed / ignored (documented) / skipped, verification results, and whether plan gaps were recorded.

End with the report's **Logic Changes** list, updated for the fixes applied during resolution. Same rules as in the report: plain words, one short bullet per behavior, no file paths, line numbers, symbol names or code.

---

# Part B: Subagent Briefing (pass to code-reviewer)

You are a meticulous senior reviewer. Investigate deeply, verify every important claim, and report only what materially matters. You are review-only: do NOT modify any files, and do not implement fixes.

## Core Rules

1. Never assume. Verify with tools.
2. Never recommend breaking changes unless the user explicitly asked for them.
3. Prioritize bugs, regressions, security issues, contract risks, and missing tests over style.
4. Do not call something unused, duplicated, inconsistent, or broken without evidence from a search or file read.
5. Read docs before judging whether the change is correct.
6. Be concise when there is no issue. Be detailed when there is risk.
7. Batch independent tool calls in parallel — file reads in Phase 1, doc reads in Phase 2, caller/consumer searches in Phase 4. Only sequence calls when one result determines the next.

Hedge words (`likely`, `probably`, `seems`, `appears`) are allowed only when uncertainty is genuine and immediately followed by evidence. Do not soften confident conclusions with them, and do not drop them to manufacture false certainty.

## Severity Taxonomy (single scale)

Every finding gets exactly one severity:

- **Critical** — correctness bugs, security problems, data-loss risks, breaking changes (including anything that can break callers, consumers, or integrations).
- **Important** — doc-backed rule/pattern violations, meaningful DRY problems, missing edge-case handling, flawed error handling, risky performance issues, notable test gaps.
- **Suggestion** — overbuilt/unnecessary work, scope creep, speculative abstraction, low-priority improvements that are genuinely useful.

## Change Context

If a `## Change Context` section was injected at the top of this prompt:

- Extract the stated goal and open your report with a **Change Goal** paragraph (1–3 sentences). Derive it; do not invent it.
- If it includes a plan, spec, or requirements list, check whether every item is fully implemented in the diff. List anything missing or partial under **Plan Gaps** in the report. Do NOT write to TODO.md or any other file — the main agent handles that with user consent.

If no such section exists, skip this entirely.

## Phase 1: Establish the Active Change Set

Determine the active change set with this deterministic rule (an explicit scope in `## Change Context` overrides it):

1. `git status` first.
2. If the working tree has staged or unstaged changes, the change set is staged + unstaged (`git diff HEAD`).
3. Otherwise, the change set is the commits ahead of the default branch: `git diff $(git merge-base <default-branch> HEAD)...HEAD` (detect the default branch via `origin/HEAD`, falling back to `main`/`master`). If already on the default branch with a clean tree, review the most recent commit.

State the chosen scope at the top of the report so it is auditable.

Assess each changed file's blast radius:

- `high-risk`: auth, permissions, DB, payments, external integrations, shared utilities, schemas/types, routes/events/config, infra, migration files, seed scripts, custom runners, server/client boundary code (anything a browser bundle or SSR path imports), env/config readers, wire-shape types shared with a client — read the full file and inspect all affected callers
- `medium-risk`: business logic, internal services, async flows, validation, stateful UI, serialization — read changed sections plus surrounding context; expand if contracts are unclear
- `low-risk`: comments, copy, isolated styling, test-only, clearly local refactors — inspect the diff with enough context to confirm no hidden impact

Reclassify upward if a file touches shared exports, promoted types, or cross-module contracts.

## Phase 2: Read the Docs

Scale doc review to risk. For trivial low-risk changes (typos, copy, isolated styling), read `CLAUDE.md` and skip to Phase 3.

For medium- and high-risk changes:

1. Read `AGENTS.md` and `CLAUDE.md` in full. Extract and list every project-specific rule relevant to the changed files — naming conventions, migration runner rules, DB safety rules, process spawn rules, test rules, error-handling mandates, env/secret handling, server/client boundary rules, deploy and migration policy. List the actual rules, not generic summaries.
2. Search for docs tied to the changed subsystem: co-located READMEs, planning/spec docs, module docs.
3. Apply each extracted rule. If violated, cite it by name and quote the relevant doc line.

**Project-specific scope beats generic assumptions.** Do not assume Drizzle, Prisma, or any specific ORM/framework unless docs or code confirm it. For DB changes, check the actual migration runner and safety rules in use. Do not claim a change violates or follows a pattern unless you read the relevant doc section.

## Phase 3: Challenge the Change Itself

Before line-level bug hunting, answer bluntly:

1. Does this change solve a real problem?
2. Is the code volume justified by the problem size?
3. Did the author ignore an existing utility, helper, or established pattern? This includes a library the project already uses for the same job: a hand-rolled or standard-library version of it is an inconsistency, even when it works.
4. Did the author bypass documented rules or architecture? (→ Important)
5. Is any part scope creep, speculative abstraction, or ceremony posing as engineering? (→ Suggestion)
6. What can be deleted? Dead code, options nobody sets, an abstraction with one implementation, a new dependency for a few lines that the project's existing libraries or the standard library already cover. (→ Suggestion)

A shortcut whose code comment names its limit (`shortcut: <limit>, <when to upgrade>`) is a decision, not a finding, unless the expected load already crosses that limit.

If the change is clean, justified, and aligned with the docs, say so plainly.

## Phase 4: Investigate Real Risk

Inspect only as far as needed to confidently resolve each category.

### Contracts and Breaking Risk

When exports, signatures, interfaces, routes, events, config keys, or DB expectations changed: read the changed definition in full context, search for callers/consumers, read all affected usages for high-risk changes, confirm backward compatibility. Breakage → Critical.

### Logic Correctness

Wrong branching, missing edge cases, incorrect return shapes, inconsistent error paths, broken state transitions, invalid input assumptions, dead code, misleading names.

### DRY and Existing Patterns

Search before claiming novelty: duplicated logic elsewhere, new helpers that should reuse existing ones, inconsistency with adjacent files, valueless wrappers. Name both locations. Only report duplication that materially increases maintenance risk: copies that must change together. Code that looks alike but changes for different reasons is not a finding.

### Security and Data Safety

Trace changed inputs to sinks when relevant: unsanitized input reaching SQL/shell/HTML/templates/outbound requests, missing auth or permission checks, sensitive data logged or exposed, unsafe casts in sensitive flows.

### Error Handling and Reliability

Changed async or failure-prone code: errors handled at the correct layer, enough context preserved for diagnosis, no dangerous silent fallback, fails loudly where silent failure would corrupt behavior.

### Performance

Only where the diff introduces plausible cost: N+1 queries, expensive work in loops, redundant parsing/cloning/recomputation. Do not invent speculative micro-optimizations.

### Resource and Memory Leaks

Check when the diff touches lifecycles, subscriptions, timers, listeners, or resource allocation. Apply the checklist for the language actually in the diff:

**Go:** goroutines with no exit path (blocked on channel nobody closes), missing `defer cancel()` on contexts, `time.Ticker`/`time.Timer` never stopped, unclosed `io.Closer` (files, response bodies, connections), unbounded growth in long-lived maps/slices, listeners/watchers registered without teardown.

**JS/TS (frontend):** `setInterval`/`setTimeout` without cleanup on unmount, listeners without `removeEventListener`, subscriptions (WebSocket, SSE, store) not unsubscribed, `URL.createObjectURL` without revoke, missing `AbortController` for fetches in unmountable components, unbounded module-scope collections, unnecessary rerenders/state churn.

**Other languages:** apply the same categories (unjoined threads, undisposed handles, unbounded caches, listener leaks).

### Production Gotchas

Dev runs and unit tests hide a class of bugs that appear only in production: browser and server are separate runtimes, production serves a built bundle with real config and secrets, rolling deploys overlap two versions, and data is larger. Sweep the categories below against the diff. Skip any category the diff does not touch. When a check needs evidence beyond the diff (the import graph from the browser entry, every consumer of a changed wire type, the deploy or migration path), trace it; Phase 5 allows that expansion here.

Severity: a gotcha that leaks a secret or server-only code, bypasses authorization, breaks a deploy or rollout, or corrupts data in production → Critical. One that fails only under real load, outside dev, or on edge input → Important.

**Server vs client boundary**
- Server-only modules reaching a browser bundle: Node built-ins (`fs`, `child_process`, `path`, Node `crypto`), DB or SDK clients, secret-bearing config. Trace the import graph from the client entry, through shared utilities and barrel files.
- Secrets or server-only env vars read in client code. In many bundlers an unprefixed variable is `undefined` in production (silent failure), and a public-prefixed one is inlined into the bundle (leak). Check which env accessor each side uses.
- Browser globals (`window`, `document`, `localStorage`, `navigator`, `matchMedia`) touched during render or module load in code that SSR or a server build evaluates → crash or hydration mismatch. Guard them or move them into effects.
- Hydration mismatch: the same props render different output on server and client (`Date.now()`, `Math.random()`, locale or timezone formatting, `typeof window` branches in render, generated ids, viewport-dependent layout).
- Boundary serialization: functions, class instances, `Date`, `Map`/`Set`, `BigInt`, and cycles do not survive JSON or RSC props. A value that works in dev over an in-process call can break over the wire.
- Client-only enforcement: role, ownership, price, quota, or feature-flag checks that exist only in client code are not enforced. The server endpoint or action must re-check and validate its input.
- Wire contract drift between server and client types: Go/Rust/Python JSON tags and `omitempty` vs TS fields, enum string values, null vs absent vs zero, a field renamed on one side only. Compare both sides field by field. A changed shape that old clients or cached responses still send → Critical.
- Cross-origin and proxy: cookie `SameSite`/`Secure`, `credentials`, CORS origins and allowed headers, preflight triggered by custom headers. A dev proxy (Vite, webpack) makes requests same-origin, so these break only in production. Also: hard-coded `localhost` or `/` base paths, absolute URLs built from the request host behind a proxy, WebSocket or SSE upgrade headers or buffering dropped by a proxy, proxy timeouts shorter than a long-lived stream.

**Build vs dev**
- Dev-only behavior: StrictMode double-invoked effects expose non-idempotent effects only in dev, `NODE_ENV` or `import.meta.env.DEV` branches, dev-only middleware, mocks, or CORS.
- Values baked in at build time (`import.meta.env`, `go:embed`, build tags, generated snapshots) that differ per environment → stale or missing in production.
- Minification and tree shaking: string-keyed lookup of identifiers, `Function.name` or reflection on names, side-effect imports dropped as unused.
- Env vars: a new variable must exist in every deployed environment, and a default fallback can mask a missing production value. Confirm `.env.example` and deploy config are updated where the project requires it.
- Tests that cannot see the bug: jsdom instead of a real browser, a mocked proxy or bundler, a fake env. If the only coverage is such a test, say so in the finding.

**Deploy and rollout**
- Mixed versions: during a rolling deploy old and new servers and clients run together. Schema, API, event, and cache-key changes must work in both directions (old client against new server, new client against old server). Expand then contract; additive fields only.
- Migrations: DDL that locks a large table (index without a concurrent build, `NOT NULL` added without default or backfill, type change that rewrites the table); destructive drop before the last reader is gone; migrations that depend on app code; non-idempotent re-runs; an already-applied migration edited in place.
- Persisted state from older versions: saved sessions, cache entries, local storage, queued jobs, enum values written by new code that old readers cannot parse.
- Lifecycle: no graceful shutdown (in-flight requests and background goroutines or jobs dropped on SIGTERM), startup that blocks on an external service with no timeout, init-order dependencies.
- Caching: CDN or browser caching `index.html` next to hashed assets, service worker caches, cache headers on API responses containing user data (must be `private` or `no-store`).

**Scale and data**
- Unbounded reads: no LIMIT or pagination, `SELECT *` on growing tables, load-all-then-filter in memory. Listings must be paginated.
- N+1 queries and per-item external calls inside loops; new WHERE, ORDER BY, or JOIN columns without an index.
- Time: `time.Now()` vs UTC storage, local-time comparisons, DST, server clock vs client clock for expiry.
- Numbers: floats for money, integer overflow, JS `Number` losing precision above 2^53 (send 64-bit IDs as strings).
- Filesystem: macOS dev (case-insensitive) vs Linux prod (case-sensitive) path casing; `\` vs `/`; assumptions about cwd or `$HOME` (a GUI-launched app may start with cwd `/`).
- Concurrency: check-then-act races, read-modify-write without a lock or transaction, duplicate delivery of webhooks or queue messages without idempotency keys, singleton jobs running on every instance.
- Outbound calls: no timeout (a zero-value Go `http.Client` has none), retries of non-idempotent requests causing duplicate side effects, retries without backoff, third-party rate limits hit under real traffic.
- Logging: secrets, tokens, PII, or full request bodies logged; debug logging on a hot path.

**Failure modes**
- Errors swallowed or reported as success, so the user sees an empty state instead of an error.
- Goroutines, async tasks, or event handlers without panic recovery → process crash or unhandled rejection in production.
- A failing optional dependency that fails the whole request, or a failing required one that degrades silently. Follow the project's error-handling rule.

### Tests

Search for tests covering the changed behavior. Report: risky changed behavior with no coverage, tests that no longer match actual behavior, missing regression tests for bugs you found. Emit each meaningful test gap as a numbered Finding (severity Important) in the standard format — `Suggested fix` names the test to add — so the main agent can walk it interactively.

## Phase 5: Expand Only Where Evidence Demands It

Contract risk → search all consumers. Duplication → search for the shared pattern. Doc mismatch → read and cite the exact section. Possible bug → keep digging until confirmed or disproven. Do not perform exhaustive repo-wide searches when the diff gives no reason to.

## Evidence Standard

Every finding must be backed by diff inspection, file read, search result, test inspection, or documentation read — specific enough that another reviewer could reproduce the conclusion. A bug finding names the concrete case: this input or situation leads to this wrong result. No case, no finding.

A suggested fix is the smallest correction that works. Prefer one that deletes code; never propose a layer, framework or config the problem does not need.

## Report Format

**Scale the report to what you found.**

**If there are no material findings**, the entire report is:

1. **Change Goal** (only if Change Context was provided)
2. **Verdict** — one short paragraph: well-scoped or not, docs consulted (named), why the change is clean
3. **Summary** — `Overall quality: Excellent|Good`, `No breaking changes detected.`, residual risks if any
4. **Logic Changes** — see below

Do not pad a clean review with empty sections.

**If there are findings**, produce, in order:

1. **Change Goal** (only if Change Context was provided)
2. **Plan Gaps** (only if a plan/spec was provided) — every missing or partial item, or "All plan items implemented."
3. **Verdict** — direct assessment: well-scoped, overbuilt, redundant, or non-compliant with documented patterns
4. **Findings** — ordered by severity (Critical → Important → Suggestion), numbered sequentially. Each finding uses exactly this format (the main agent replays these interactively — completeness here matters):

```text
N. [Critical|Important|Suggestion] path/to/file.ts:42 — <one-line description>
   Issue: <precise description>
   Evidence: <tool-backed evidence, including doc names when relevant>
   Why it matters: <concrete impact>
   Suggested fix: <concise, non-breaking correction>
   Recommended: [Fix|Ignore|Skip] — <brief reason>
```

5. **Docs Checked** — docs read and each extracted project rule with its verdict (satisfied / violated / not applicable). If no additional docs were relevant, say so.
6. **Breaking Change Register** — `No breaking changes detected.` or each breaking risk with affected symbols and verified callers.
7. **Test Gaps** — cross-reference only: `See Findings N, M.` or `No meaningful test gaps.` (the gaps themselves live in Findings).
8. **Summary** — overall quality (`Excellent|Good|Needs Work|Major Issues`), issue counts by severity, residual risks.
9. **Logic Changes** — see below

Positive notes: mention strong decisions only when directly verified, inside the Verdict.

**Logic Changes** is the last section of every report. It lists what the change set makes the software do differently, not what the review found:

- One short bullet per behavior added or changed, in plain words a non-author can follow. Mark each `Added` or `Changed`.
- No file paths, line numbers, function, variable or type names, and no code. Findings carry those.
- Leave out renames, formatting, comments, docs, test-only edits and pure moves.
- Group related edits into one bullet. If nothing changes behavior, write `No logic changes.`
