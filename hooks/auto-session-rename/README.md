# Auto Session Rename

Automatically rename your Claude Code session with a meaningful title based on your first message. Uses a `UserPromptSubmit` hook that calls Haiku to summarize your prompt into a concise 3-6 word title.

**Examples:**

| First prompt | Session title |
|---|---|
| "I want to refactor the auth middleware to use JWT tokens" | Refactor Auth To JWT |
| "Fix the bug where users can't log in on mobile" | Fix Mobile Login Bug |
| "Add a dark mode toggle to the settings page" | Add Dark Mode Toggle |

## Prerequisites

- `jq` installed (`brew install jq` / `apt install jq`)
- `curl` installed
- `ANTHROPIC_API_KEY` environment variable set (already the case if you use Claude Code)

## Setup

### 1. Copy the hook script

```bash
mkdir -p ~/.claude/hooks
cp auto-rename.sh ~/.claude/hooks/auto-rename.sh
chmod +x ~/.claude/hooks/auto-rename.sh
```

### 2. Register the hook

Add this to your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "type": "command",
        "command": "bash ~/.claude/hooks/auto-rename.sh"
      }
    ]
  }
}
```

> If you already have other `UserPromptSubmit` hooks, just append the entry to the existing array.

### 3. Done

Start a new Claude Code session and send a message. The session gets renamed automatically based on your first prompt. Subsequent prompts are ignored (no re-renaming).

## How it works

```
User sends first prompt
  → UserPromptSubmit hook fires
  → Script checks /tmp/claude-renamed-<session-id>
    → Flag exists? → return {} (no-op)
    → Flag missing? → continue
  → Creates flag file
  → Calls Haiku API: "summarize this into a 3-6 word title"
  → Returns { hookSpecificOutput: { sessionTitle: "<title>" } }
  → Session is renamed
```

## Performance

- **First prompt**: ~300-500ms added (one Haiku API call). Runs before Claude processes your message.
- **Subsequent prompts**: ~1ms (file existence check, no API call). Zero noticeable impact.
- **Cost**: ~$0.001 per session (one Haiku call with minimal tokens).

## Customization

- **Model**: Change `claude-haiku-4-5-20251001` to any model (e.g. `claude-sonnet-4-6` for slightly better titles, at higher cost/latency)
- **Title length**: Adjust `max_tokens` and the prompt wording (`3-6 word title`)
- **Scope**: Move the settings entry from `~/.claude/settings.json` (global) to `<project>/.claude/settings.json` (per-project) to only enable it in specific projects
