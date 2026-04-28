#!/bin/bash
# Auto-rename Claude Code session with a meaningful title based on the first user prompt.
# Uses Haiku to generate a concise 3-6 word title.

INPUT=$(cat /dev/stdin)
PROMPT=$(echo "$INPUT" | jq -r '.prompt')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // .sessionId // empty')

# Use a flag file to only rename on the first prompt
FLAG="/tmp/claude-renamed-${SESSION_ID:-$$}"

if [ -f "$FLAG" ]; then
  echo '{}'
  exit 0
fi

touch "$FLAG"

# Ask Haiku to generate a short, meaningful title
TITLE=$(curl -s https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d "$(jq -n --arg p "$PROMPT" '{
    "model": "claude-haiku-4-5-20251001",
    "max_tokens": 20,
    "messages": [{"role": "user", "content": ("Give me a concise 3-6 word title for this task. Reply with ONLY the title, nothing else.\n\nTask: " + $p)}]
  }')" | jq -r '.content[0].text')

# Fallback to truncated prompt if API call fails
if [ -z "$TITLE" ] || [ "$TITLE" = "null" ]; then
  TITLE=$(echo "$PROMPT" | head -c 40 | sed 's/[^a-zA-Z0-9 -]//g' | xargs)
fi

jq -n --arg t "$TITLE" '{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "sessionTitle": $t
  }
}'
