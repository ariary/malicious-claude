# remote-control-alternative

A Claude Code skill that opens a remote SSH session from your Mac to an Android device (Termux) using a personal SSH server + ngrok TCP tunnel. Bypasses macOS Remote Login restrictions.

---

## How it works

```
Mac (sshd on port 2222) ──ngrok TCP──► internet ◄── Termux (ssh client)
```

1. Starts a personal OpenSSH server on port 2222 (no admin rights needed)
2. Exposes it via ngrok TCP (plain TCP tunnel, no client-side proxy required)
3. You enter the ngrok host/port in Termux to connect

---

## Setup

### Mac — one-time

**1. Install dependencies**
```bash
brew install openssh ngrok/ngrok/ngrok
```

**2. Authenticate ngrok** (free account required for TCP tunnels)
```bash
ngrok config add-authtoken <your-token>
```
Get your token at https://dashboard.ngrok.com/get-started/your-authtoken

**3. Install the skill**
```bash
mkdir -p ~/.claude/skills/remote-control-alternative
cp SKILL.md mac_share.sh ~/.claude/skills/remote-control-alternative/
chmod +x ~/.claude/skills/remote-control-alternative/mac_share.sh
```

**4. Add an alias** (optional but convenient)
```bash
echo "alias remote-mobile='~/.claude/skills/remote-control-alternative/mac_share.sh'" >> ~/.aliases
```

### Android (Termux) — one-time

**1. Install Termux** from [F-Droid](https://f-droid.org/packages/com.termux/) (not Play Store)

**2. Install dependencies**
```bash
pkg install openssh termux-api jq
```
Also install the **Termux:API** app from F-Droid.

**3. Install the connect script**
```bash
curl -fsSL <url>/connect_mac.sh -o ~/.shortcuts/connect_mac.sh
chmod +x ~/.shortcuts/connect_mac.sh
```
Or copy `connect_mac.sh` manually.

**4. First run** — the script will:
- Ask for your Mac username (saved to `~/.mac_connect_config`)
- Generate an SSH key at `~/.ssh/id_mac_access`
- Show you the public key to add on your Mac

**5. Add phone key to Mac**
```bash
echo 'ssh-ed25519 AAAA...' >> ~/.ssh/authorized_keys
```

---

## Daily usage

**On Mac** — run the skill via Claude Code:
```
/remote-control-alternative
```
Or directly:
```bash
remote-mobile
```
It will print:
```
  Host : 5.tcp.eu.ngrok.io
  Port : 12345
```

**On Termux** — run:
```bash
~/.shortcuts/connect_mac.sh
```
Enter the host and port when prompted → connected.

---

## Bonus — run Claude Code from your phone

Once connected via SSH, you have a full Mac terminal on your phone. You can launch Claude Code:

```bash
claude -r
```

This is the alternative to Claude Code's built-in `/remote-control` feature (currently limited/not available to everyone) — you get the same result: a fully interactive Claude Code session accessible from your phone, without needing the official feature to be enabled on your account.

---

## Files

| File | Purpose |
|---|---|
| `SKILL.md` | Claude Code skill entry point |
| `mac_share.sh` | Starts sshd + ngrok TCP on Mac |
| `connect_mac.sh` | Termux connect script (runs on Android) |
