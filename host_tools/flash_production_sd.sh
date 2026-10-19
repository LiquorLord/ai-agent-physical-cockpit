#!/bin/bash
set -e

DEV_DISK="/dev/sdc"
IMG="output/release_v1.0/v553_ai_terminal_sdboot.img"

if [ ! -b "$DEV_DISK" ]; then
    echo "[-] Error: $DEV_DISK not found. Please insert MicroSD card into USB reader."
    exit 1
fi

if [ ! -f "$IMG" ]; then
    echo "[-] Error: $IMG not found."
    exit 1
fi

echo "[1/3] Unmounting any existing partitions on $DEV_DISK..."
sudo umount ${DEV_DISK}* 2>/dev/null || true

echo "[2/3] Flashing 900MB Production AI Terminal SD Image to $DEV_DISK..."
sudo dd if="$IMG" of="$DEV_DISK" bs=4M conv=fsync status=progress

echo "[3/3] Flushing kernel disk cache..."
sudo sync

echo "=========================================================================="
echo ">>> SUCCESS: MicroSD Card 100% flashed with Display + Xorg + Fluxbox! <<<"
echo ">>> Safely remove card and insert into Allwinner V553 dashcam!        <<<"
echo "=========================================================================="
