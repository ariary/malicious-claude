#!/data/data/com.termux/files/usr/bin/bash
# connect_mac.sh — Connect to Mac SSH session from Termux via ngrok TCP
# Requires: termux-api (pkg install termux-api), jq (pkg install jq)
# Install: curl -fsSL <url>/connect_mac.sh -o ~/.shortcuts/connect_mac.sh && chmod +x ~/.shortcuts/connect_mac.sh

CONFIG_FILE="$HOME/.mac_connect_config"
KEY_FILE="$HOME/.ssh/id_mac_access"

# ── Helper: dialog input ─────────────────────────────────────────────────────
ask() {
    local title="$1" hint="${2:-}"
    termux-dialog text -t "$title" -i "$hint" | jq -r '.text // empty'
}

# ── First-time setup ─────────────────────────────────────────────────────────
if [ ! -f "$CONFIG_FILE" ]; then
    MAC_USER=$(ask "Mac username" "e.g. john")
    if [ -z "$MAC_USER" ]; then termux-toast "Cancelled."; exit 1; fi
    echo "MAC_USER=$MAC_USER" > "$CONFIG_FILE"
fi

# shellcheck disable=SC1090
source "$CONFIG_FILE"

# ── Generate SSH key if missing ──────────────────────────────────────────────
if [ ! -f "$KEY_FILE" ]; then
    mkdir -p ~/.ssh && chmod 700 ~/.ssh
    ssh-keygen -t ed25519 -f "$KEY_FILE" -N "" -q
    termux-dialog confirm \
        -t "Add key to Mac" \
        -i "Run on Mac:\n\necho '$(cat "${KEY_FILE}.pub")' >> ~/.ssh/authorized_keys\n\nTap OK when done."
fi

# ── Ask for ngrok host and port ──────────────────────────────────────────────
NGROK_HOST=$(ask "ngrok host" "e.g. 5.tcp.eu.ngrok.io")
if [ -z "$NGROK_HOST" ]; then termux-toast "Cancelled."; exit 1; fi

NGROK_PORT=$(ask "ngrok port" "e.g. 12345")
if [ -z "$NGROK_PORT" ]; then termux-toast "Cancelled."; exit 1; fi

termux-toast "Connecting to $MAC_USER@$NGROK_HOST:$NGROK_PORT ..."
echo "Connecting to $MAC_USER@$NGROK_HOST port $NGROK_PORT ..."

ssh \
    -i "$KEY_FILE" \
    -p "$NGROK_PORT" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ForwardAgent=yes \
    -t \
    "${MAC_USER}@${NGROK_HOST}"
