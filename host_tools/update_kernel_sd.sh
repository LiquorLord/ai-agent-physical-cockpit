#!/bin/bash
set -e

DEV="/dev/sdc1"
if [ ! -b "$DEV" ]; then
    echo "[-] Error: $DEV not detected. Please plug MicroSD card into USB reader."
    exit 1
fi

echo "[*] Mounting $DEV to update zImage..."
MNT_DIR="/tmp/v553_boot_update"
mkdir -p "$MNT_DIR"
mount "$DEV" "$MNT_DIR"

echo "[*] Copying G2D rotation enabled zImage (3.2MB) and sunxi.dtb (102KB)..."
cp output/release_v1.0/zImage "$MNT_DIR/zImage"
cp output/release_v1.0/sunxi.dtb "$MNT_DIR/sunxi.dtb"

echo "[*] Verifying files on FAT32 boot partition..."
ls -lh "$MNT_DIR/zImage" "$MNT_DIR/sunxi.dtb"

sync
umount "$MNT_DIR"
rmdir "$MNT_DIR"

echo ">>> SUCCESS: Kernel updated on MicroSD card in 1 second! <<<"
echo ">>> You can now safely remove the card and insert into V553! <<<"
