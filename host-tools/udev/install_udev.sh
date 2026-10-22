#!/bin/bash
# ============================================================================
# AI-Agent Physical Cockpit - Host Udev Installer
# Install Allwinner V553 cockpit USB device permission rules and system integration
# ============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RULE_SRC="$SCRIPT_DIR/99-ai-agent-cockpit.rules"
RULE_DEST="/etc/udev/rules.d/99-ai-agent-cockpit.rules"

echo "[*] Installing AI-Agent Cockpit Udev rules to $RULE_DEST..."
sudo cp "$RULE_SRC" "$RULE_DEST"
sudo chmod 644 "$RULE_DEST"

echo "[*] Reloading host udev daemon..."
sudo udevadm control --reload-rules
sudo udevadm trigger

echo "[OK] Udev rules installed successfully! Non-root ADB and FEL access active."
