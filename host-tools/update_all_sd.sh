#!/usr/bin/env bash
# ==============================================================================
# Update SyterKit Boot0 & Kernel/DTB on existing MicroSD Card
# ==============================================================================
set -e

DEV_DISK="$1"
RELEASE_DIR="${2:-output/release_v1.0}"

if [ -z "$DEV_DISK" ] || [ ! -b "$DEV_DISK" ]; then
    echo "Usage: $0 /dev/sdX [release_dir]"
    echo "Example: $0 /dev/sdb output/release_v1.0"
    echo ""
    echo "Removable devices currently available:"
    lsblk -d -o NAME,SIZE,MODEL,TRAN,HOTPLUG 2>/dev/null || true
    exit 1
fi

BOOT0_BIN="$RELEASE_DIR/syter_boot_card.bin"
ZIMAGE_BIN="$RELEASE_DIR/zImage"
DTB_BIN="$RELEASE_DIR/sunxi.dtb"

for f in "$BOOT0_BIN" "$ZIMAGE_BIN" "$DTB_BIN"; do
    if [ ! -f "$f" ]; then
        echo "[-] Error: Required file '$f' not found."
        echo "[-] Please ensure prebuilt binaries or build outputs exist in '$RELEASE_DIR'."
        exit 1
    fi
done

# Deduce partition 1
if [[ "$DEV_DISK" =~ [0-9]$ ]]; then
    DEV_BOOT="${DEV_DISK}p1"
else
    DEV_BOOT="${DEV_DISK}1"
fi

if [ ! -b "$DEV_BOOT" ]; then
    echo "[-] Error: Boot partition '$DEV_BOOT' not found."
    exit 1
fi

echo "Target Disk  : $DEV_DISK"
echo "Target Boot  : $DEV_BOOT"
echo "Release Files: $RELEASE_DIR"
echo ""

read -r -p "Confirm update to $DEV_BOOT? (y/N): " CONFIRM
if [[ "$CONFIRM" != [yY] && "$CONFIRM" != [yY][eE][sS] ]]; then
    echo "Aborted."
    exit 0
fi

echo "[1/4] Flashing patched SyterKit Boot0 to Sector 16 (offset 8KB)..."
sudo dd if="$BOOT0_BIN" of="$DEV_DISK" bs=512 seek=16 conv=notrunc status=none

echo "[2/4] Mounting $DEV_BOOT..."
MNT="/tmp/v553_boot_update"
mkdir -p "$MNT"
sudo mount "$DEV_BOOT" "$MNT"

echo "[3/4] Updating kernel, DTB, and config.txt..."
sudo cp -f "$ZIMAGE_BIN" "$MNT/zImage"
sudo cp -f "$DTB_BIN" "$MNT/sunxi.dtb"

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

echo "=========================================================================="
echo ">>> SUCCESS: SyterKit Boot0 & Boot partition 100% updated!            <<<"
echo ">>> Safely remove MicroSD card and insert into Allwinner V553 device! <<<"
echo "=========================================================================="
