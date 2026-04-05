---
name: remote-control-alternative
description: Start a remote SSH session accessible from a mobile device via ngrok TCP tunnel
---

## Remote Mobile Access

Dependency check: !`command -v ngrok &>/dev/null && echo "✅ ngrok ready" || echo "❌ ngrok not found — run: brew install ngrok/ngrok/ngrok"`

## Your task

1. Locate the script next to this skill file. The skill is installed at `~/.claude/skills/remote-control-alternative/` — run `mac_share.sh` from that directory using the Bash tool **in the background** (append `&`), redirecting output to `/tmp/remote_mobile.log`.

2. Poll `/tmp/remote_mobile.log` every 2 seconds (up to 30s) until you see a line containing `MOBILE CONNECTION READY`.

3. Once ready, parse and display to the user:
   - **Host** (line containing `Host :`)
   - **Port** (line containing `Port :`)
   - Remind the user to run `mac` in Termux and enter those two values.

4. Tail `/tmp/remote_mobile.log` for any errors and surface them if the tunnel fails to start.
