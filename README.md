# claude-si

Personal Claude Code plugin marketplace (`claude-si`).

| Plugin | Path | What it does |
| --- | --- | --- |
| `context-status` | `context-status/` | Status line: model, dir, git branch, context usage. |
| `jev-guard` | `jev-guard/` | PostToolUse guardrail: scans inbound web/MCP/network results with TypeSafe Jev. |
| `okf` | `okf/` | OKF knowledge bundle workflow skill plus SessionStart lookup reminder. |
| `branch-indicator` | `branch-indicator/` | Announces the git worktree (or branch) the session works in, at start and on change. |
| `communication-style` | `communication-style/` | SessionStart: injects the ASD-STE100 communication standard (ported from ocode). |
| `working-principles` | `working-principles/` | SessionStart: fix suggestion protocol, docs-as-truth, surgical changes, planning rules. |
| `code-quality` | `code-quality/` | SessionStart: no fallbacks, error logging, no upsert, TODO tracking, test safety. |
| `git-safety` | `git-safety/` | SessionStart: no bare `git reset --soft` in shared repos. |
| `frontend-standards` | `frontend-standards/` | SessionStart: use the present UI library, semantic HTML, UX care. |
| `lean-changes` | `lean-changes/` | SessionStart: smallest complete change, reuse existing helpers and libraries first, no unrequested abstractions. |
| `review-changes` | `review-changes/` | `/review-changes` command: deep review through the `code-reviewer` subagent, then issue-by-issue resolution. |

## Installation

### 1. Add the marketplace

The marketplace is named `claude-si`. The GitHub repository is still named `claude-plugins`, so the URLs below use that name.

From GitHub, in a terminal:

```sh
claude plugin marketplace add https://github.com/u007/claude-plugins
# or the short form:
claude plugin marketplace add u007/claude-plugins
```

Or inside a Claude Code session:

```
/plugin marketplace add u007/claude-plugins
```

Check it was added:

```sh
claude plugin marketplace list
```

Or from a local clone:

```sh
git clone https://github.com/u007/claude-plugins ~/www/claude-si
```

```
/plugin marketplace add ~/www/claude-si
```

### 2. Install jev-guard

```
/plugin install jev-guard@claude-si
```

Then finish the setup in [jev-guard/README.md](jev-guard/README.md#setup): export `TYPESAFE_API_KEY` in your shell profile and start Claude Code from a terminal that loads it.

### 3. Install okf

```
/plugin install okf@claude-si
```

See [okf/README.md](okf/README.md).

### 4. Install context-status

```
/plugin install context-status@claude-si
```

Installing the plugin switches the status line on. A SessionStart hook writes `statusLine` into `~/.claude/settings.json` when it is unset, and repoints a stale context-status path after a plugin update. It leaves any other `statusLine` untouched. Restart Claude Code after the first install.

To use a different status line, set `statusLine` yourself. The hook will not overwrite a command that does not point at context-status.

`context-status/bin/prev_command.txt` chains a previous status line. The shipped value points at a cognee statusline that most machines do not have. Blank the file, or replace it with your own command, unless you use that one. A broken chained command only drops its own segment.

Not yet verified end to end on a clean machine.

### 5. Install branch-indicator

```
/plugin install branch-indicator@claude-si
```

See [branch-indicator/README.md](branch-indicator/README.md).

## Install a single plugin

Each plugin is independent. Install only the ones you want.

### Via the marketplace

Add the marketplace (step 1 above), then install one plugin by name:

```sh
claude plugin install jev-guard@claude-si
# or: okf@claude-si, context-status@claude-si, branch-indicator@claude-si
```

### Manually from a local clone

No marketplace needed. Clone the repo, then load one plugin directory for a session:

```sh
git clone https://github.com/u007/claude-plugins ~/www/claude-si
claude --plugin-dir ~/www/claude-si/jev-guard
```

`--plugin-dir` applies to that run only. To make it permanent, use the marketplace route with the local clone path (`/plugin marketplace add ~/www/claude-si`, then `/plugin install <name>@claude-si`).

Plugin-specific setup still applies: `TYPESAFE_API_KEY` for jev-guard, the `statusLine` entry for context-status.

## Claude desktop app

Plugins work in the **Code** tab of the desktop app, in local and SSH sessions. Cloud sessions have no plugin browser. The regular Claude chat tab does not load Claude Code plugins.

Quick install:

1. Open the **Code** tab and start a local session.
2. Add the marketplace by typing `/plugin marketplace add u007/claude-plugins` in the prompt.
3. Install the plugins you want, for example `/plugin install okf@claude-si`. Or open the plugin browser and pick them from `claude-si`.
4. Start a new session so the plugins load.

Prefer a terminal? Run steps 1 and 2 of [Installation](#installation) there. The desktop app's local sessions read the same settings files as the terminal, so the plugins show up in the Code tab with nothing more to do.

Manage, disable or uninstall later from **+ > Plugins > Manage plugins**.

`context-status` needs the extra `statusLine` entry from [Installation](#4-install-context-status) in `~/.claude/settings.json`. Whether the desktop Code tab renders a custom status line is not verified.

## Upgrading from `personal-plugins`

The marketplace was renamed from `personal-plugins` to `claude-si`. Installs recorded as `<plugin>@personal-plugins` no longer resolve. Add the marketplace again and reinstall each plugin as `<plugin>@claude-si`. Then remove the stale `@personal-plugins` entries from `enabledPlugins` and `extraKnownMarketplaces` in `~/.claude/settings.json`.

## Updating

Refresh the marketplace, then update the plugin:

```sh
claude plugin marketplace update claude-si
claude plugin update jev-guard@claude-si
```

Inside a session: `/plugin marketplace update claude-si`, then `/plugin update jev-guard@claude-si`. Restart Claude Code afterwards so the new version loads.

Updates are picked up only when the plugin `version` changes, so bump `version` in both the plugin's `plugin.json` and its `marketplace.json` entry when releasing.

If you installed from a local clone, pull first:

```sh
git -C ~/www/claude-si pull
```

`context-status` runs from the clone path in `~/.claude/settings.json`, so `git pull` alone updates it. No plugin update is needed for the status line script.

## Tests

```sh
python3 context-status/bin/test_context_statusline.py
python3 branch-indicator/scripts/test_branch_indicator.py
```
