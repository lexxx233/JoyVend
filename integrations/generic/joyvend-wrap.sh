#!/usr/bin/env bash
# joyvend generic capture wrapper — for any agent driven from a shell loop (custom
# REPLs, CI agents, chat clients with a shell tool) that lacks a hook system.
#
# Usage:  joyvend-wrap.sh <your-agent-command> [args...]
# It captures each stdin line as a raw turn, then pipes the line to your agent.
#
# Env: JOYVEND_BANK (default: "default").
set -uo pipefail
bank="${JOYVEND_BANK:-default}"

while IFS= read -r line; do
  joyvend capture --bank "$bank" --role user -- "$line" >/dev/null 2>&1 || true
  printf '%s\n' "$line"
done | "$@"
