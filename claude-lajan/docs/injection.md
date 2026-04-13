# Injection Methods

Three hooks inject learnings into Claude Code sessions. Each serves a different moment and purpose.

---

## 1. UserPromptSubmit hook — session start injection

**When:** Fires when you submit any prompt.  
**What it injects:** Top 5 improvement + strength findings from the digest, wrapped in `<session-learnings>` tags, prepended to your prompt.  
**Effect:** Claude sees learnings as structured context at the start of each turn.  
**Best for:** High-level behavioural reminders that apply to the whole session.

```
<session-learnings>
Previous session analysis identified these key learnings:
- [improvement/global] Always verify file existence before claiming a task is done
- [strength/project] Using parallel agent calls reduced latency significantly
</session-learnings>

Your actual prompt here...
```

---

## 2. PreToolUse hook — pre-execution reminders

**When:** Fires before every Bash, Edit, Write, or MultiEdit call.  
**What it injects:** Improvement-type learnings written to stderr (shown in the Claude Code UI).  
**Effect:** Claude sees a reminder at the exact moment it's about to make a risky change.  
**Best for:** Catching repeated mistakes at the point of execution.

```
[claude-lajan] Reminders before Bash:
  • Don't retry a failing command without diagnosing why it failed first
  • Run go vet after edits, not just go build
```

This hook never blocks tool execution — it only adds context.

---

## 3. Persistent memory injection

**When:** Written after each `lajan run`, read by Claude Code at session start.  
**Destinations:**

| Scope | File | Loaded when |
|-------|------|-------------|
| `global` | `~/.claude/CLAUDE.md` | Every Claude Code session on this machine |
| `project` | `~/.claude/projects/<encoded-path>/memory/feedback_learnings.md` | Sessions in that project directory |

**Effect:** Long-term memory that persists across many sessions without repeating in prompts.  
**Best for:** Architectural preferences, project-specific patterns, accumulated wisdom.

The global section is managed between `<!-- claude-session-reviewer:start -->` and `<!-- claude-session-reviewer:end -->` markers — `lajan reset --global` removes only that section, leaving the rest of your CLAUDE.md intact.

---

## Digest

The digest (`~/.claude-lajan/digest.md`) is the rolling store of top learnings — max 20 items, deduplicated. New findings are prepended; old ones fall off when the cap is reached. Both hook 1 and hook 2 read from the digest.

Use `lajan digest` to inspect it, `lajan reset --digest` to clear it.
