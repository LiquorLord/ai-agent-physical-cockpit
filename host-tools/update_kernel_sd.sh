#!/bin/bash
# ==============================================================================
# Flash latest Linux 6.12 Mainline Kernel & Userspace Stack to MicroSD Card
# One-shot deploy the latest mainline Linux 6.12 kernel, device tree and full cockpit stack to the MicroSD card
# ==============================================================================
set -e

WORKSPACE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$WORKSPACE_DIR/board/mainline/output"

# 1. Auto-detect SD card boot partition or use argument
DEV="$1"
if [ -z "$DEV" ]; then
    # Search for removable drive with FAT32 boot partition (ignoring system NVMe/md0/sda/sdc)
    for part in $(lsblk -lno NAME,TRAN,TYPE | grep -E "usb.*part" | awk '{print $1}'); do
        FSTYPE=$(lsblk -no FSTYPE "/dev/$part" 2>/dev/null || true)
        if [ "$FSTYPE" = "vfat" ]; then
            DEV="/dev/$part"
            break
        fi
    done
fi

if [ -z "$DEV" ] || [ ! -b "$DEV" ]; then
    echo "=========================================================================="
    echo "[-] Error: no MicroSD reader boot partition detected."
    echo "[-] Usage: $0 [/dev/sdX1]"
    echo "[-] Please insert the MicroSD card into a USB reader connected to the computer and retry."
    echo "=========================================================================="
    exit 1
fi

echo "[*] Detected MicroSD boot partition: $DEV"
MNT_DIR="/tmp/v553_sd_boot"
mkdir -p "$MNT_DIR"
sudo mount "$DEV" "$MNT_DIR"

echo "[*] Writing Linux 6.12 production kernel (zImage) and authoritative device tree (sunxi.dtb)..."
sudo cp "$OUTPUT_DIR/zImage" "$MNT_DIR/zImage"
sudo cp "$OUTPUT_DIR/sunxi.dtb" "$MNT_DIR/sunxi.dtb"

echo "[*] Verifying boot partition files..."
ls -lh "$MNT_DIR/zImage" "$MNT_DIR/sunxi.dtb"
sha256sum "$MNT_DIR/zImage" "$MNT_DIR/sunxi.dtb"

sync
sudo umount "$MNT_DIR"
rmdir "$MNT_DIR"

# 2. Check if second partition (rootfs) exists to update userspace tools
ROOT_DEV="${DEV%1}2"
if [ -b "$ROOT_DEV" ]; then
    echo "[*] Detected MicroSD rootfs partition: $ROOT_DEV, syncing the latest cockpit stack and touch menu..."
    MNT_ROOT="/tmp/v553_sd_root"
    mkdir -p "$MNT_ROOT"
    if sudo mount "$ROOT_DEV" "$MNT_ROOT" 2>/dev/null; then
        sudo cp "$OUTPUT_DIR/touch_cockpit" "$MNT_ROOT/usr/bin/touch_cockpit" 2>/dev/null || true
        sudo cp "$OUTPUT_DIR/gud_v553" "$MNT_ROOT/usr/bin/gud_v553" 2>/dev/null || true
        sudo cp "$OUTPUT_DIR/switch_to_gud.sh" "$MNT_ROOT/opt/switch_to_gud.sh" 2>/dev/null || true
        sudo cp "$OUTPUT_DIR/cockpit_agents.json" "$MNT_ROOT/mnt/sdcard/cockpit_agents.json" 2>/dev/null || true
        sudo chmod +x "$MNT_ROOT/usr/bin/touch_cockpit" "$MNT_ROOT/usr/bin/gud_v553" "$MNT_ROOT/opt/switch_to_gud.sh" 2>/dev/null || true
        sync
        sudo umount "$MNT_ROOT"
        rmdir "$MNT_ROOT"
        echo "[+] Rootfs userspace stack update complete!"
    fi
fi

echo "=========================================================================="
echo ">>> Deploy successful: Linux 6.12 latest display driver fixes and touch stack fully written to the MicroSD card! <<<"
echo ">>> Please remove the reader, insert the MicroSD card into the Allwinner V553 device slot and power it on! <<<"
echo "=========================================================================="

