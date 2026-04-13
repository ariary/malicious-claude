#!/bin/sh
# Claude Code Stop hook wrapper.
# Forwards stdin to the stop-hook binary.
exec "$HOME/.claude-reviewer/bin/stop-hook"
