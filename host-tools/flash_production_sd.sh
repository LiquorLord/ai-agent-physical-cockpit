#!/usr/bin/env bash
# ==============================================================================
# Flash Production SD Image for Allwinner V553 AI-Agent Cockpit
# Safe, interactive flasher supporting both uncompressed (.img) and gzip (.img.gz)
# ==============================================================================
set -e

DEV_DISK="$1"
IMG_INPUT="$2"

echo "=========================================================================="
echo "    AI-Agent Physical Cockpit - MicroSD Flashing Utility"
echo "=========================================================================="

# 1. Resolve Target Image
if [ -z "$IMG_INPUT" ]; then
    # Search common candidate paths
    for candidate in \
        "output/release_v1.0/v553_ai_terminal_sdboot.img" \
        "output/release_v1.0/v553_ai_terminal_sdboot.img.gz" \
        "v553_ai_terminal_sdboot.img" \
        "v553_ai_terminal_sdboot.img.gz"; do
        if [ -f "$candidate" ]; then
            IMG_INPUT="$candidate"
            break
        fi
    done
fi

if [ -z "$IMG_INPUT" ] || [ ! -f "$IMG_INPUT" ]; then
    echo "[-] Error: Production SD image not found."
    echo ""
    echo "Please download 'v553_ai_terminal_sdboot.img.gz' from the GitHub Releases page:"
    echo "  https://github.com/your-username/ai-agent-physical-cockpit/releases"
    echo ""
    echo "Usage:"
    echo "  $0 /dev/sdX [path_to_image]"
    echo ""
    echo "Example:"
    echo "  $0 /dev/sdb ./v553_ai_terminal_sdboot.img.gz"
    exit 1
fi

# 2. Resolve Target Device
if [ -z "$DEV_DISK" ]; then
    echo "[!] No target disk specified. Scanning removable USB block devices..."
    echo ""
    lsblk -d -o NAME,SIZE,MODEL,TRAN,HOTPLUG 2>/dev/null || true
    echo ""
    echo "Usage: $0 /dev/sdX [path_to_image]"
    echo "Example: $0 /dev/sdb $IMG_INPUT"
    exit 1
fi

if [ ! -b "$DEV_DISK" ]; then
    echo "[-] Error: Device '$DEV_DISK' is not a valid block device."
    exit 1
fi

# Safety check: prevent flashing system root or partition instead of whole disk
ROOT_DISK=$(findmnt -n -o SOURCE / | sed 's/p\?[0-9]*$//')
if [ "$DEV_DISK" = "$ROOT_DISK" ]; then
    echo "[-] CRITICAL REFUSAL: $DEV_DISK contains the host root filesystem (/)."
    echo "[-] Flashing aborted to protect host operating system."
    exit 1
fi

echo ""
echo "Target Disk  : $DEV_DISK"
lsblk "$DEV_DISK" -o NAME,SIZE,TYPE,MOUNTPOINT,MODEL
echo "Image File   : $IMG_INPUT ($(ls -lh "$IMG_INPUT" | awk '{print $5}'))"
echo ""
echo "=========================================================================="
echo " [!] WARNING: ALL DATA ON $DEV_DISK WILL BE PERMANENTLY ERASED!"
echo "=========================================================================="
read -r -p "Type 'YES' (all uppercase) to confirm and flash: " CONFIRM
if [ "$CONFIRM" != "YES" ]; then
    echo "Flashing canceled by user."
    exit 0
fi

echo ""
echo "[1/3] Unmounting existing partitions on $DEV_DISK..."
sudo umount ${DEV_DISK}* 2>/dev/null || true

echo "[2/3] Writing image to $DEV_DISK..."
if [[ "$IMG_INPUT" == *.gz ]]; then
    echo "Decompressing and writing gzip image via pipeline..."
    gzip -dc "$IMG_INPUT" | sudo dd of="$DEV_DISK" bs=4M status=progress conv=fsync
else
    sudo dd if="$IMG_INPUT" of="$DEV_DISK" bs=4M status=progress conv=fsync
fi

echo "[3/3] Flushing kernel cache..."
sudo sync

echo "=========================================================================="
echo ">>> SUCCESS: MicroSD Card 100% flashed!                               <<<"
echo ">>> Safely remove the card and insert into the Allwinner V553 device! <<<"
echo "=========================================================================="
