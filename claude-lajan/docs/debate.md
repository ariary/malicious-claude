# Debate Engine

## Agent Roles

| Agent | Model | Role |
|-------|-------|------|
| **Advocate** | claude-sonnet-4-5 | Identifies what worked well — efficient tool use, good decisions, patterns worth repeating |
| **Critic** | claude-sonnet-4-5 | Identifies waste — redundant steps, missed shortcuts, premature completion claims |
| **Challenger** | claude-sonnet-4-5 | Stress-tests both sides — rejects vague findings, calls out weak reasoning |
| **Consensus** | claude-opus-4-6 | Synthesises final verdict, tags each finding by type and scope |

## Round Structure

Each round:
1. Advocate + Critic run **in parallel** (halves latency)
2. Challenger reads both outputs and challenges weak findings
3. Consensus agent reads all three, outputs structured JSON:

```json
{
  "status": "CONSENSUS_REACHED" | "CONTINUE",
  "rationale": "brief reason",
  "findings": [
    { "type": "strength" | "improvement", "scope": "project" | "global", "text": "..." }
  ]
}
```

The loop exits on `CONSENSUS_REACHED` or after 5 rounds (safety cap). If the consensus agent returns malformed JSON, it falls back to `CONTINUE` — the debate never crashes.

## Finding Scopes

- **`project`** — specific to the current codebase (e.g. "this repo uses X pattern, stop reinventing it")
- **`global`** — applies to all Claude Code sessions (e.g. "always verify before claiming a task is done")

Scope determines where findings are injected. See [injection.md](injection.md).

## Session Input

The debate receives a structured summary of the JSONL transcript: duration, turn counts, tool calls, files touched, bash commands, and a truncated (≤8000 chars) conversation transcript. Thinking blocks are stripped.
