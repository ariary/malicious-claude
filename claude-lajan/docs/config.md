# Configuration

All tuning constants are in `internal/config/config.go`. Rebuild after changes (`make install`).

| Constant | Default | Description |
|----------|---------|-------------|
| `MaxDebateRounds` | `5` | Maximum debate rounds before forcing consensus |
| `MaxTokens` | `2048` | Max tokens per agent response |
| `DigestMaxItems` | `20` | Maximum findings kept in the rolling digest |
| `DigestInjectTop` | `5` | How many digest findings are injected per prompt/pretool |
| `ModelAdvocate` | `claude-sonnet-4-5` | Model for Advocate agent |
| `ModelCritic` | `claude-sonnet-4-5` | Model for Critic agent |
| `ModelChallenger` | `claude-sonnet-4-5` | Model for Challenger agent |
| `ModelConsensus` | `claude-opus-4-6` | Model for Consensus agent (heavier, fewer calls) |

## Runtime paths

| Path | Purpose |
|------|---------|
| `~/.claude-lajan/queue.txt` | Sessions pending review (one JSONL path per line) |
| `~/.claude-lajan/digest.md` | Rolling top learnings |
| `~/.claude-lajan/reports/YYYY-MM-DD/<id>.md` | Per-session debate report |
| `~/.claude-lajan/lajan.log` | Background process output |
| `~/.claude-lajan/bin/` | All installed binaries |
| `~/.claude/CLAUDE.md` | Global learnings section (managed block) |
| `~/.claude/projects/<path>/memory/feedback_learnings.md` | Project-specific learnings |

## PreToolUse hook — which tools trigger it

Defined in `cmd/hook/pretool/main.go`:

```go
var riskyTools = map[string]bool{
    "Bash":      true,
    "Edit":      true,
    "Write":     true,
    "MultiEdit": true,
}
```

To add or remove tools, edit this map and run `make install`.

## Disabling individual hooks

To disable a specific injection method without full uninstall:

```sh
# Disable only the PreToolUse hook
lajan reset --hooks --yes
# Then re-register only the ones you want:
# Edit ~/.claude/settings.json manually, or re-run make install
# and comment out the PreToolUse line in patchSettings()
```

Or use `lajan reset --hooks --yes` followed by manually editing `~/.claude/settings.json` to remove individual hook entries.
