# Claude Code: automatic capture + distill

Make joyvend retention **automatic** — no reliance on the agent remembering to call
`retain`. Two hooks:

- **`UserPromptSubmit` → `joyvend-capture.sh`** logs every user turn as a raw, deduped
  `capture` memory (the safety net).
- **`Stop` → `joyvend-distill-check.sh`** nudges Claude every *N* turns to promote the
  durable captures into curated `mental_model`s (the judgment stays Claude's).

## Prerequisites

- A running joyvend (`joyvend serve` or the GUI), with `joyvend` on your `PATH`.
- `jq` installed.

## Install (per project)

```sh
mkdir -p .claude
cp integrations/claude-code/joyvend-capture.sh        .claude/
cp integrations/claude-code/joyvend-distill-check.sh  .claude/
chmod +x .claude/joyvend-*.sh
```

Then merge `integrations/claude-code/settings.example.json` into your `.claude/settings.json`
(or `~/.claude/settings.json` for all projects). Restart Claude Code so it picks up the hooks.

## Configure (optional env vars)

| Var | Default | Meaning |
|---|---|---|
| `JOYVEND_BANK` | project dir name | which memory bank to capture into |
| `JOYVEND_BASE` | `http://127.0.0.1:8765` | joyvend server URL (used in the distill nudge) |
| `JOYVEND_DISTILL_EVERY` | `10` | nudge Claude to distill every N turns |

## Verify

1. Send any prompt in Claude Code, then run `joyvend memories --bank <bank>` — your prompt
   was captured **without Claude doing anything**.
2. Stop joyvend, send a prompt — the turn proceeds normally (capture is non-fatal).
3. Set `JOYVEND_DISTILL_EVERY=2`; after 2 prompts Claude receives the distill checkpoint and
   promotes captures via `retain {type:"mental_model", supersedes:[...]}`.

## Notes

- Captured rows are **hidden from `recall`/`reflect` by default** (they're a substrate); they
  surface only after distillation, or via `recall {"include_captures": true}`.
- The capture hook is silent and never writes to Claude's context; the distill hook only speaks
  every N turns, and guards against Stop-hook loops via `stop_hook_active`.
