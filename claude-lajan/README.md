# claude-lajan

Self-improving feedback loop for Claude Code sessions running in auto/bypass-permissions mode.

After each session, four adversarial AI agents debate what worked and what didn't. Their consensus is injected back into future sessions as reminders — so Claude Code progressively self-corrects without manual review.

---

## Install

```sh
export ANTHROPIC_API_KEY=sk-ant-...   # add to ~/.zshrc to persist
cd claude-lajan
make install
```

That's it. Every Claude Code session is now reviewed automatically.

---

## How it works

1. **Session ends** → stop hook queues the transcript, spawns analysis in background
2. **4 agents debate** (≤5 rounds): Advocate finds strengths, Critic finds waste, Challenger stress-tests both, Consensus decides
3. **Learnings are injected** into future sessions via 3 hooks:
   - `UserPromptSubmit` — top learnings prepended to your first prompt each session
   - `PreToolUse` — improvement reminders shown before Bash/Edit/Write calls
   - Project memory + `~/.claude/CLAUDE.md` — persisted for long-term context
4. **Over time**, the area of improvement shrinks

Background log: `tail -f ~/.claude-lajan/lajan.log`

---

## Commands

| Command | Description |
|---------|-------------|
| `lajan summarize` | Show all learnings currently injected into sessions |
| `lajan digest` | Print the full rolling digest |
| `lajan list` | Show sessions waiting in the queue |
| `lajan run` | Process all queued sessions manually |
| `lajan run --last` | Process only the most recent session |
| `lajan run --dry-run` | Debate without writing or injecting anything |
| `lajan reset` | Interactive reset — choose what to remove |
| `lajan reset --global` | Remove learnings from `~/.claude/CLAUDE.md` |
| `lajan reset --project` | Remove learnings from current project's memory |
| `lajan reset --digest` | Clear the rolling digest |
| `lajan reset --hooks` | Uninstall all lajan hooks from settings |
| `lajan reset --all --yes` | Remove everything, no prompt |

---

## Uninstall

```sh
make uninstall
```

Removes all hooks from `~/.claude/settings.json` and deletes all binaries. Your CLAUDE.md and project memory are untouched unless you run `lajan reset --all` first.

---

## Details

- [How the debate engine works](docs/debate.md)
- [Injection methods explained](docs/injection.md)
- [Tuning and configuration](docs/config.md)
