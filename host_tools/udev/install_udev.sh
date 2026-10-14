#!/bin/bash
# ============================================================================
# AGY Physical Cockpit - Host Udev Installer (Bilingual)
# 安装 Allwinner V553 座舱 USB 设备权限规则与系统集成
# ============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RULE_SRC="$SCRIPT_DIR/99-agy-cockpit.rules"
RULE_DEST="/etc/udev/rules.d/99-agy-cockpit.rules"

echo "[*] Installing AGY Cockpit Udev rules to $RULE_DEST..."
sudo cp "$RULE_SRC" "$RULE_DEST"
sudo chmod 644 "$RULE_DEST"

echo "[*] Reloading host udev daemon..."
sudo udevadm control --reload-rules
sudo udevadm trigger

echo "[OK] Udev rules installed successfully! Non-root ADB and FEL access active."
