#!/bin/sh
# finish-check.sh - read-only finish gate.
#
# Usage: finish-check.sh [project-dir]   (default: current directory)
#
# Reports:
#   1. placeholder markers in source and config files (TODO, TBD, FIXME, XXX,
#      @@...@@, {{...}}, "coming soon"); prose files (.md, .txt) are not scanned
#      because their ordinary wording trips the marker patterns
#   2. world-readable files that look like secrets or databases (templates such
#      as .env.example are not reported)
#   3. processes whose working directory is inside the project, and their
#      listening TCP ports
#
# It only reads. It never deletes, changes permissions, or kills anything.
# Exit status: 0 = nothing found and every check ran, 1 = review or could not
# verify, 2 = usage error.
#
# Adapted from AICodeKing's Fable-Cheap skill (MIT, Copyright 2026 AICodeKing):
# https://github.com/aicodeking/Fable-Cheap
# Changes: generic wording, source/config-only marker scan, template exclusion,
# and TODO.md is skipped in the marker scan because that file is the tracked
# record of incomplete work.

set -u

PRUNE_DIRS='node_modules .git target dist build .next .venv venv __pycache__ vendor coverage'
MAX_LINES=40

dir=${1:-.}
if [ ! -d "$dir" ]; then
  printf 'finish-check: not a directory: %s\n' "$dir" >&2
  exit 2
fi
root=$(cd "$dir" && pwd -P) || exit 2
found=0

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/finish-check.XXXXXX") || {
  printf 'finish-check: cannot create a temporary directory\n' >&2
  exit 2
}
trap 'rm -rf "$tmpdir"' 0

prune_expr() {
  # Prints a find(1) prune expression for the noisy directories.
  first=1
  printf '('
  for d in $PRUNE_DIRS; do
    if [ $first -eq 1 ]; then first=0; else printf ' -o'; fi
    printf ' -name %s' "$d"
  done
  printf ' ) -prune -o'
}
PRUNE=$(prune_expr)

section() { printf '\n== %s\n' "$1"; }

# ---------------------------------------------------------------- 1. markers
section "Placeholder markers"
# shellcheck disable=SC2086
markers=$(find "$root" $PRUNE -type f -size -2000k \
    ! -name 'TODO.md' \
    \( -name '*.rs' -o -name '*.go' \
       -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' \
       -o -name '*.mjs' -o -name '*.py' -o -name '*.svelte' -o -name '*.vue' \
       -o -name '*.html' -o -name '*.css' -o -name '*.sh' -o -name '*.toml' \
       -o -name '*.json' -o -name '*.yml' -o -name '*.yaml' -o -name '*.c' \
       -o -name '*.h' -o -name '*.cpp' -o -name '*.swift' -o -name '*.kt' \) \
    -exec grep -HnIE '@@[A-Za-z0-9_]+@@|\{\{[A-Za-z0-9_ ]+\}\}|\bTODO\b|\bTBD\b|\bFIXME\b|\bXXX\b|[Cc]oming soon' {} + 2>"$tmpdir/markers.err")
# grep exits 1 when nothing matches; only stderr means the scan was incomplete.
if [ -s "$tmpdir/markers.err" ]; then
  found=1
  echo "WARNING: placeholder scan incomplete; these paths could not be read:"
  sed 's/^/  /' "$tmpdir/markers.err" | head -n "$MAX_LINES"
fi
if [ -n "$markers" ]; then
  found=1
  count=$(printf '%s\n' "$markers" | wc -l | tr -d ' ')
  printf '%s\n' "$markers" | awk -v r="$root/" 'index($0, r) == 1 { print substr($0, length(r) + 1); next } { print }' | head -n "$MAX_LINES"
  [ "$count" -gt "$MAX_LINES" ] && printf '... %s matches in total\n' "$count"
else
  echo "none"
fi

# ------------------------------------------------------ 2. exposed secrets
section "World-readable secret or database files"
# shellcheck disable=SC2086
secrets=$(find "$root" $PRUNE -type f -perm -004 \
    ! -name '.env.example' ! -name '.env.sample' ! -name '.env.template' \
    \( -name '*.db' -o -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.db-wal' \
       -o -name '*.pem' -o -name '*.key' -o -name '*.p12' -o -name '*.pfx' \
       -o -name 'id_rsa*' -o -name 'id_ed25519*' -o -name '.env' -o -name '.env.*' \
       \) -print 2>"$tmpdir/secrets.err")
if [ -s "$tmpdir/secrets.err" ]; then
  found=1
  echo "WARNING: secret-file scan incomplete; these paths could not be read:"
  sed 's/^/  /' "$tmpdir/secrets.err" | head -n "$MAX_LINES"
fi
if [ -n "$secrets" ]; then
  found=1
  printf '%s\n' "$secrets" | while IFS= read -r f; do
    # Only field 1 (the mode) is read, and the path is quoted; SC2012 does not apply.
    # shellcheck disable=SC2012
    ls -l "$f" 2>/dev/null | awk '{print $1}' | tr '\n' ' '
    printf '%s\n' "${f#"$root"/}"
  done | head -n "$MAX_LINES"
  echo "(check each: secrets belong in the OS keychain or 0600 files)"
else
  echo "none"
fi

# ------------------------------------------- 3. processes and listening ports
section "Processes running inside the project"
scan=1
pids=""
if [ -d /proc ] && [ -r /proc/self/cwd ]; then
  for p in /proc/[0-9]*; do
    cwd=$(readlink "$p/cwd" 2>/dev/null) || continue
    case "$cwd" in "$root"|"$root"/*) pids="$pids ${p#/proc/}" ;; esac
  done
elif command -v lsof >/dev/null 2>&1; then
  if lsof -nP -a -d cwd -Fpn >"$tmpdir/cwd.out" 2>"$tmpdir/cwd.err"; then
    pids=$(awk -v r="$root" '
      /^p/ { pid = substr($0, 2) }
      /^n/ { n = substr($0, 2); if (n == r || index(n, r "/") == 1) print pid }' <"$tmpdir/cwd.out")
  elif [ -s "$tmpdir/cwd.err" ]; then
    scan=0
    echo "(could not list process working directories)"
    sed 's/^/  /' "$tmpdir/cwd.err" | head -n "$MAX_LINES"
  fi
else
  scan=0
  echo "(neither /proc nor lsof available; skipped)"
fi
self=$$
procs=""
for pid in $pids; do
  [ "$pid" = "$self" ] && continue
  line=$(ps -o pid=,ppid=,command= -p "$pid" 2>/dev/null) || continue
  case "$line" in *finish-check.sh*) continue ;; esac
  [ -n "$line" ] && procs="$procs$line
"
done
if [ -n "$procs" ]; then
  found=1
  printf '%s' "$procs" | head -n "$MAX_LINES"
elif [ "$scan" -eq 0 ]; then
  echo "(not verified)"
else
  echo "none"
fi

section "Listening TCP ports from those processes"
if [ -n "$procs" ] && command -v lsof >/dev/null 2>&1; then
  plist=$(printf '%s' "$procs" | awk '{print $1}' | paste -sd, -)
  if lsof -nP -a -p "$plist" -iTCP -sTCP:LISTEN >"$tmpdir/ports.out" 2>"$tmpdir/ports.err"; then
    ports=$(awk 'NR > 1 {print $1, $2, $9}' <"$tmpdir/ports.out")
    if [ -n "$ports" ]; then
      printf '%s\n' "$ports"
      echo "(stop servers you started; anything on *:PORT is not bound to 127.0.0.1)"
    else
      echo "none"
    fi
  elif [ -s "$tmpdir/ports.err" ]; then
    found=1
    echo "WARNING: could not list listening ports:"
    sed 's/^/  /' "$tmpdir/ports.err" | head -n "$MAX_LINES"
  else
    echo "none"
  fi
else
  if [ "$scan" -eq 0 ]; then
    echo "(not verified: process scan skipped)"
  else
    echo "none"
  fi
fi

echo
if [ $found -eq 0 ]; then
  if [ "$scan" -eq 0 ]; then
    echo "finish-check: could not verify (process scan unavailable; nothing was changed)"
    exit 1
  fi
  echo "finish-check: clean"
else
  echo "finish-check: review the items above before the final response (nothing was changed)"
fi
exit $found
