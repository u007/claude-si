---
name: okf
description: Work with an OKF (Open Knowledge Format) v0.1 knowledge bundle, a docs/ directory of markdown files with YAML frontmatter plus a marked index.md and an append-only log.md. Defines the workflow - look up the bundle BEFORE starting work, update docs AFTER durable knowledge is produced, deduplicate before writing, and keep index.md and log.md in sync. Use at the start of any task in a repo whose docs/index.md contains okf_version, and whenever creating, auditing, or editing bundle docs.
when_to_use: At the start of ANY task (feature, fix, investigation, question about why or how a decision was made) in a repo where docs/index.md has an okf_version marker - run the lookup step first. Also when the user mentions OKF, a knowledge bundle, .okfignore, docs/.okf.lock, or asks to create, annotate, audit, deprecate or clean up curated docs.
---

# OKF knowledge bundle (v0.1)

A bundle is a `docs/` directory of curated markdown files with YAML frontmatter. It records decisions, gotchas, playbooks and concepts that are not obvious from code or git history.

This skill is the whole mechanism: nothing runs automatically. Follow the steps below yourself, in order.

## 0. Is there a bundle?
Check `docs/index.md` at the project root. It is a bundle only if its frontmatter has `okf_version: "0.1"`. If the file is missing, or exists without the marker (mkdocs, docusaurus, hand-written), there is no bundle: skip this skill, and never overwrite that `index.md`. Create a bundle only if the user asks (see Init).

## Delegate to the okf agents
This plugin ships two Haiku agents. The SessionStart hook tells you to use them when a bundle exists. Bundle reads and writes go through them, so doc dumps stay out of the main context.
- **Lookup:** `okf:context` (read-only: no write tools). Brief it with the task topic. It runs steps 1.1 to 1.4 and returns a short digest: relevant doc paths, the facts that matter, anything stale or contradicted by code, and "none matched" if so. Work from the digest. Read a cited doc yourself only when the digest is not enough.
- **Update:** `okf:context-writer`. You decide whether the task produced durable knowledge (step 2), because you have the conversation. If yes, give it a self-contained brief, since it cannot see the conversation: what was decided, found or changed, why, the files involved, and what you already verified. It follows steps 3 and 4 and reports the paths it created, updated or deprecated. It stops and reports if the brief is incomplete.
- **One writer:** run at most one `okf:context-writer` at a time, and do not edit the bundle yourself while it runs.
- **Check its work:** Haiku is a weaker model. After an update, run `git diff docs/` (and `git status docs/`) and confirm the doc says what you told it, the log entry exists, the index lists the doc, and nothing unrelated changed. Fix or revert problems before reporting done.
- **No dispatch available** (you are a subagent, or the agents are not registered): do the steps directly.

## 1. BEFORE work: look up
Do this before reading code or editing anything, for every task. Run it through `okf:context` as described above. Speed matters: the lookup runs before every task, so search, never browse.
1. Do not read `docs/index.md` in full; it can be hundreds of KB. Search it: each line is `- [title](path) - description`, so a case-insensitive grep of `docs/index.md` for each topic term, showing matching lines (Grep tool: `output_mode: content`, `-i`), returns the title and description matches directly.
2. Search doc bodies for the same terms, listing matching files only (`files_with_matches`) (`grep -ril "<term>" docs --include='*.md'`). Rank: a title match weighs 3, description 2, body 1; with several terms, require all of them. Respect `.okfignore` and skip `(deprecated)` entries unless checking history.
3. Read at most the top 5 docs, in full. Read another only when a doc you read points to it as essential.
4. Verify only the code claims your answer relies on, with one grep for the cited symbol or file. Do not check every anchor on a page. Citations rot (see Gotchas).
5. State briefly which docs you used, or that none matched. Then do the work.

## 2. AFTER work: decide whether to write
Update the bundle only for durable knowledge produced by this task: a decision and its reason, a gotcha or surprising failure, a playbook that worked, a schema or architecture change. Q&A, routine edits and chat-only discussion are no-ops. If in doubt, do nothing. Say in one line whether you updated the bundle and why or why not.

## 3. Writing a doc
(Done by `okf:context-writer`; see "Delegate to the okf agents".)
1. **Deduplicate first.** Re-run the search for the doc's topic. If a doc already covers it, update that doc instead of creating a near-duplicate. If two existing docs overlap, merge the content into the better one and deprecate the other with a reason pointing at the survivor.
2. Verify claims against the code or system before writing them down. Claims the brief marks as verified need no re-check; verify the rest. **Prune while editing:** delete or rewrite every part of an existing doc the change makes false. A live doc keeps no stale section. Deprecate the whole doc only when nothing in it still holds.
3. Write the doc (format below) at a bundle-relative path, one topic per doc. The path must stay inside `docs/` (no `..`, not absolute) and must not be `index.md` or `log.md`.
4. Set `timestamp` to now (RFC 3339) on create and on every content change.
5. Write the doc, then run the update script once (step 4). It takes the lock, appends the log entry and regenerates the index.

### Doc format
```yaml
---
type: Concept            # Concept, Gotcha, Playbook, Decision, ...
title: Short title
description: One line, at most 200 characters, specific enough to tell docs apart in a search.
resource: path/or/url    # what the doc is about
tags: [area, topic]
timestamp: 2026-10-30T09:00:00Z
status: active           # or deprecated
deprecated_reason: why   # only when deprecated
---
```
`type` is required. A doc without this frontmatter is non-conforming: it stays in the bundle, gets listed under Unclassified, and should be annotated. Keep unknown frontmatter keys when you rewrite a doc.

## 4. Log and index (keep both in sync after every change)
Never edit `docs/log.md` or `docs/index.md` by hand. After each doc change, run the bundled script (it sits in this skill's base directory, under `scripts/`):
```
perl <skill-base-dir>/scripts/okf-update.pl docs log <Creation|Update|Deprecation|Deletion> <bundle-relative-path> "<one-sentence summary>"
```
It takes the lock on `docs/.okf.lock`, adds the log entry under today's heading, regenerates `index.md` from every doc's frontmatter, and writes both atomically. Run one call per changed doc. `perl <skill-base-dir>/scripts/okf-update.pl docs index` only regenerates the index. If the script exits non-zero, report its error; do not fall back to hand edits. Keep the summary to one sentence: the log is history, the doc holds the detail.

The formats the script produces, for reference:

**`docs/log.md`** is append-only history, newest day first:
```
# Directory Update Log

## 2026-10-30

* **Update**: what changed and why ([docs/path.md](/path.md))
```
Action is one of `Creation`, `Update`, `Deprecation`, `Deletion`. Add the entry under today's `## YYYY-MM-DD` heading, creating that heading directly under the title if it does not exist. The link target is the bundle-relative path with one leading `/`.

**`docs/index.md`** is regenerated, not hand-patched. Rebuild it from every doc's frontmatter:
- Frontmatter holds only `okf_version: "0.1"` (preserve the existing value).
- One `# <dir>` section per top-level directory, sorted by name. Root-level docs go under `# Concepts`.
- Entries inside a section sorted by title, case-insensitive: `- [title](relative/path.md) - description`. Use the path as the title if the title is empty. Append ` (deprecated)` for deprecated docs.
- Non-conforming docs go last under `# Unclassified` as `- [filename](path)`, sorted by path.
- Write to a temp file and rename it over `index.md`, so a failed run leaves the old index intact.

## Deprecate and clean up
- Deprecate, never delete, as the normal path: set `status: deprecated` and `deprecated_reason`, then run the script with `Deprecation`.
- Cleanup deletes deprecated docs. List them with path and reason first and delete only after the user confirms. Then run the script with `Deletion` for each.

## Exclusions
Skipped as docs: `index.md`, `log.md`, dot-files and dot-directories, non-`.md` files, `PLAN-*.md` and `*.OCODE.md`. `docs/.okfignore` adds patterns, one per line (blank lines and `#` comments skipped). A pattern matches the basename and the bundle-relative path, and a trailing `/` matches directories. Put non-bundle markdown that lives under `docs/` (eval output, generated reports, drafts) in `.okfignore`, or it is counted as bundle docs and pollutes search and counts.

## Concurrency
Other writers (ocode, other sessions) may touch the bundle. They all take an exclusive `flock` on `docs/.okf.lock`. The update script holds that lock while it appends the log and regenerates the index, and it rebuilds the index from the files on disk, so a concurrent writer's doc is never lost from it. Run at most one `okf:context-writer` at a time.

## Init and audit
- **Init** (only when asked, and only when no marker exists): refuse if `docs/index.md` exists without the marker. Otherwise create `index.md` containing the marker and a `log.md` holding only `# Directory Update Log`, annotate existing docs with frontmatter without changing their bodies, then run the script in `index` mode. If the marker already exists, run an audit instead of failing.
- **Audit:** count conforming, non-conforming and deprecated docs. Stale docs (contradict current code) get fixed, not just reported: delegate the correction to `okf:context-writer`, which corrects or prunes the false parts and deprecates the doc when nothing in it still holds. Merge duplicates into the better doc and deprecate the other. Report orphans (not reachable from the index or never referenced) and broken anchors.

## Gotchas
- **Line-anchor drift:** citations like `file.go:123` silently rot when lines are added above them. After editing a cited file, re-derive EVERY anchor on the page, checking both ends of ranges: extract each `\w+\.\w+:\d+`, print the real source line, compare. Prefer citing a symbol name to a line number. Do not trust an automated "anchors corrected" report.
- Bulk non-bundle markdown under `docs/` inflates counts and degrades lookup. Fix it with `.okfignore`.
- A folder named `docs/` is not a bundle. Only the `okf_version` marker in `index.md` makes it one.
