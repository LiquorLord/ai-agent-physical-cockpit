#!/usr/bin/env bash
# ==============================================================================
# Flash latest Linux 6.12 Mainline Kernel & Userspace Stack to MicroSD Card
# One-shot deploy the latest mainline Linux 6.12 kernel, device tree and full cockpit stack to the MicroSD card
# ==============================================================================
set -e

WORKSPACE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV="$1"
OUTPUT_DIR="$2"

if [ -z "$OUTPUT_DIR" ]; then
    if [ -d "$WORKSPACE_DIR/board/mainline/output" ]; then
        OUTPUT_DIR="$WORKSPACE_DIR/board/mainline/output"
    elif [ -d "$WORKSPACE_DIR/output/release_v1.0" ]; then
        OUTPUT_DIR="$WORKSPACE_DIR/output/release_v1.0"
    else
        OUTPUT_DIR="$WORKSPACE_DIR/board/mainline/output"
    fi
fi

# 1. Auto-detect SD card boot partition or use argument
if [ -z "$DEV" ]; then
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
    echo "[-] Usage: $0 [/dev/sdX1] [output_dir]"
    echo "[-] Please insert the MicroSD card into a USB reader connected to the computer and retry."
    echo "=========================================================================="
    exit 1
fi

if [ ! -f "$OUTPUT_DIR/zImage" ] || [ ! -f "$OUTPUT_DIR/sunxi.dtb" ]; then
    echo "[-] Error: zImage or sunxi.dtb not found in '$OUTPUT_DIR'."
    echo "[-] Please build the mainline kernel or provide a valid output directory containing zImage and sunxi.dtb."
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
        [ -f "$OUTPUT_DIR/touch_cockpit" ] && sudo cp "$OUTPUT_DIR/touch_cockpit" "$MNT_ROOT/usr/bin/touch_cockpit"
        [ -f "$OUTPUT_DIR/gud_v553" ] && sudo cp "$OUTPUT_DIR/gud_v553" "$MNT_ROOT/usr/bin/gud_v553"
        [ -f "$OUTPUT_DIR/switch_to_gud.sh" ] && sudo cp "$OUTPUT_DIR/switch_to_gud.sh" "$MNT_ROOT/opt/switch_to_gud.sh"
        [ -f "$OUTPUT_DIR/cockpit_agents.json" ] && sudo cp "$OUTPUT_DIR/cockpit_agents.json" "$MNT_ROOT/mnt/sdcard/cockpit_agents.json"
        sync
        sudo umount "$MNT_ROOT"
    fi
    rmdir "$MNT_ROOT" 2>/dev/null || true
fi

echo "=========================================================================="
echo ">>> SUCCESS: MicroSD Card Linux 6.12 kernel and stack updated!        <<<"
echo ">>> Safely remove card and insert into Allwinner V553 dashcam!        <<<"
echo "=========================================================================="
