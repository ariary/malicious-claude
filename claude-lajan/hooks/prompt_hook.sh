#!/bin/sh
# Claude Code UserPromptSubmit hook wrapper.
# Forwards stdin to the prompt-hook binary.
exec "$HOME/.claude-reviewer/bin/prompt-hook"
