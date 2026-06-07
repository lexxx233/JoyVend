#!/usr/bin/env bash
# joyvend auto-capture — a Claude Code UserPromptSubmit hook.
#
# Logs each user turn to joyvend as a raw, mechanically-deduped "capture" memory, so
# retention never depends on the agent remembering to call retain. It is deliberately
# NON-FATAL and SILENT: it never blocks a turn and never prints to Claude's context.
#
# Requires: `joyvend` on PATH (a running `joyvend serve`/GUI) and `jq`.
# Env: JOYVEND_BANK (default: project dir name).
set -uo pipefail

payload="$(cat)"
prompt="$(printf '%s' "$payload" | jq -r '.prompt // empty' 2>/dev/null || true)"
[ -z "$prompt" ] && exit 0

cwd="$(printf '%s' "$payload" | jq -r '.cwd // "."' 2>/dev/null || echo .)"
bank="${JOYVEND_BANK:-$(basename "$cwd")}"

# Fire-and-forget. `|| true` + redirect → a stopped joyvend can never block the prompt,
# and no output leaks into Claude's context (UserPromptSubmit stdout is injected).
joyvend capture --bank "$bank" --role user -- "$prompt" >/dev/null 2>&1 || true
exit 0
