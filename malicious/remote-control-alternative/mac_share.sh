#!/bin/bash
# mac_share.sh — Personal SSH server + ngrok TCP tunnel
# Requires: ngrok, openssh (brew install ngrok openssh)

set -uo pipefail

PORT=2222
SSHD_CONF="$HOME/.ssh/sshd_config_personal"
HOST_KEY="$HOME/.ssh/ssh_host_ed25519_key"
PID_FILE="$HOME/.ssh/sshd_personal.pid"

NGROK_PID=""

cleanup() {
    echo ""
    echo "Shutting down..."
    [ -n "$NGROK_PID" ] && kill "$NGROK_PID" 2>/dev/null || true
    if [ -f "$PID_FILE" ]; then
        kill "$(cat "$PID_FILE")" 2>/dev/null || true
        rm -f "$PID_FILE"
    fi
    rm -f "$SSHD_CONF"
    echo "Done."
}
trap cleanup EXIT INT TERM

# ── Find sshd (prefer Homebrew) ──────────────────────────────────────────────
if   [ -x "/opt/homebrew/sbin/sshd" ]; then SSHD="/opt/homebrew/sbin/sshd"
elif [ -x "/usr/local/sbin/sshd" ];    then SSHD="/usr/local/sbin/sshd"
else echo "❌ sshd not found. Run: brew install openssh"; exit 1; fi

command -v ngrok &>/dev/null || { echo "❌ ngrok not found. Run: brew install ngrok/ngrok/ngrok"; exit 1; }

# ── Generate host key if needed ──────────────────────────────────────────────
if [ ! -f "$HOST_KEY" ]; then
    echo "Generating SSH host key..."
    ssh-keygen -t ed25519 -f "$HOST_KEY" -N "" -q
fi

mkdir -p ~/.ssh
touch "$HOME/.ssh/authorized_keys"
chmod 700 ~/.ssh
chmod 600 "$HOME/.ssh/authorized_keys"

# ── Kill any stale process on the port ──────────────────────────────────────
lsof -ti tcp:$PORT 2>/dev/null | xargs kill -9 2>/dev/null || true
sleep 0.5

# ── Write sshd config ────────────────────────────────────────────────────────
cat > "$SSHD_CONF" <<EOF
Port $PORT
ListenAddress 127.0.0.1
HostKey $HOST_KEY
AuthorizedKeysFile $HOME/.ssh/authorized_keys
PasswordAuthentication no
PubkeyAuthentication yes
ChallengeResponseAuthentication no
UsePAM no
StrictModes no
PidFile $PID_FILE
LogLevel ERROR
EOF

# ── Start SSH server ─────────────────────────────────────────────────────────
"$SSHD" -f "$SSHD_CONF"
sleep 1

if [ ! -f "$PID_FILE" ]; then
    echo "❌ SSH server failed to start."
    exit 1
fi
echo "✅ SSH server running (PID: $(cat "$PID_FILE"))"

# ── Start ngrok TCP tunnel ───────────────────────────────────────────────────
echo "Opening ngrok TCP tunnel..."
ngrok tcp "$PORT" > /tmp/ngrok_tcp.log 2>&1 &
NGROK_PID=$!

NGROK_ADDR=""
for i in $(seq 1 15); do
    sleep 2
    NGROK_ADDR=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | python3 -c "
import sys,json
try:
    for t in json.load(sys.stdin)['tunnels']:
        url=t['public_url']
        if url.startswith('tcp://'): print(url.replace('tcp://',''),end=''); break
except: pass
" 2>/dev/null) || true
    [ -n "$NGROK_ADDR" ] && break
    echo "  Waiting for tunnel... ($((i*2))s)"
done

if [ -z "$NGROK_ADDR" ]; then
    echo "❌ Tunnel failed to start."
    cat /tmp/ngrok_tcp.log
    exit 1
fi

NGROK_HOST="${NGROK_ADDR%:*}"
NGROK_PORT="${NGROK_ADDR##*:}"

echo ""
echo "================================================"
echo "  MOBILE CONNECTION READY"
echo "================================================"
echo "  Mac user : $(whoami)"
echo "  Host     : $NGROK_HOST"
echo "  Port     : $NGROK_PORT"
echo "================================================"
echo "  Press Ctrl+C to stop"
echo ""

wait "$NGROK_PID"
