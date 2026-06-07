#!/usr/bin/env bash
# joyvend distill checkpoint — a Claude Code Stop hook.
#
# Every N turns, nudges Claude to promote its raw captures into curated mental_models.
# The TRIGGER is automatic; the JUDGMENT (what to promote, what to drop) stays Claude's —
# joyvend does no reasoning. Emits JSON with hookSpecificOutput.additionalContext, which
# Claude sees at turn end and can act on in the same session.
#
# Requires: `jq`. Env: JOYVEND_BANK, JOYVEND_BASE (default http://127.0.0.1:8765),
# JOYVEND_DISTILL_EVERY (default 10).
set -uo pipefail

payload="$(cat)"

# Avoid loops: if we're already continuing from a previous Stop hook, do nothing.
active="$(printf '%s' "$payload" | jq -r '.stop_hook_active // false' 2>/dev/null || echo false)"
[ "$active" = "true" ] && exit 0

cwd="$(printf '%s' "$payload" | jq -r '.cwd // "."' 2>/dev/null || echo .)"
bank="${JOYVEND_BANK:-$(basename "$cwd")}"
base="${JOYVEND_BASE:-http://127.0.0.1:8765}"
every="${JOYVEND_DISTILL_EVERY:-10}"

dir="${CLAUDE_PROJECT_DIR:-.}/.claude"
ctr="$dir/.joyvend-capture-count"
mkdir -p "$dir" 2>/dev/null || true
n=$(( $(cat "$ctr" 2>/dev/null || echo 0) + 1 ))
if [ "$n" -lt "$every" ]; then printf '%s' "$n" > "$ctr" 2>/dev/null || true; exit 0; fi
printf '0' > "$ctr" 2>/dev/null || true

cat <<JSON
{"hookSpecificOutput":{"hookEventName":"Stop","additionalContext":"joyvend distill checkpoint: read your recent raw captures with GET ${base}/v1/banks/${bank}/memories?type=experience&tag=capture&limit=50 , then promote the durable ones into observation/mental_model via retain (set supersedes:[<the raw capture ids you fold in>] so the raw rows are deleted), and drop the noise. Keep it brief."}}
JSON
exit 0
