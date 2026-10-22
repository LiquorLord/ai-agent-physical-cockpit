#!/bin/bash
set -e

DEV_DISK="/dev/sdc"
DEV_BOOT="/dev/sdc1"

if [ ! -b "$DEV_DISK" ]; then
    echo "[-] Error: $DEV_DISK not found. Please insert MicroSD card into USB reader."
    exit 1
fi

echo "[1/4] Flashing patched SyterKit Boot0 to Sector 16 (offset 8KB)..."
sudo dd if=output/release_v1.0/syter_boot_card.bin of="$DEV_DISK" bs=512 seek=16 conv=notrunc status=none

echo "[2/4] Mounting $DEV_BOOT..."
MNT="/tmp/v553_boot_update"
mkdir -p "$MNT"
sudo mount "$DEV_BOOT" "$MNT"

echo "[3/4] Updating kernel, DTB, and config.txt..."
sudo cp -f output/release_v1.0/zImage "$MNT/zImage"
sudo cp -f output/release_v1.0/sunxi.dtb "$MNT/sunxi.dtb"

sudo bash -c "cat << 'CFG_EOF' > '$MNT/config.txt'
[configs]
bootdelay = 1
bootargs = earlyprintk=sunxi-uart,0x02500000 console=ttyS0,115200 root=/dev/mmcblk0p2 rootwait rw devtmpfs.mount=1 loglevel=8 panic=10 init=/sbin/init
CFG_EOF"

echo "[4/4] Verifying updated files on $DEV_BOOT:"
ls -lh "$MNT/"
cat "$MNT/config.txt"

sudo sync
sudo umount "$MNT"
sudo rmdir "$MNT"

echo ">>> SUCCESS: SyterKit Boot0 & Boot partition 100% updated in 2 seconds! <<<"
echo ">>> Safely remove MicroSD card and insert into V553 dashcam! <<<"
