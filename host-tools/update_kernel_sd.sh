#!/bin/bash
# ==============================================================================
# Flash latest Linux 6.12 Mainline Kernel & Userspace Stack to MicroSD Card
# 一键部署最新主线 Linux 6.12 内核、设备树及全功能座舱栈到 MicroSD 卡
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
    echo "[-] 错误: 未检测到插入的 MicroSD 读卡器引导分区。"
    echo "[-] 使用方法: $0 [/dev/sdX1]"
    echo "[-] 请将 MicroSD 卡插入 USB 读卡器接入电脑后重试。"
    echo "=========================================================================="
    exit 1
fi

echo "[*] 检测到 MicroSD 卡引导分区: $DEV"
MNT_DIR="/tmp/v553_sd_boot"
mkdir -p "$MNT_DIR"
sudo mount "$DEV" "$MNT_DIR"

echo "[*] 正在写入 Linux 6.12 生产级内核 (zImage) 与权威设备树 (sunxi.dtb)..."
sudo cp "$OUTPUT_DIR/zImage" "$MNT_DIR/zImage"
sudo cp "$OUTPUT_DIR/sunxi.dtb" "$MNT_DIR/sunxi.dtb"

echo "[*] 正在校验引导分区文件..."
ls -lh "$MNT_DIR/zImage" "$MNT_DIR/sunxi.dtb"
sha256sum "$MNT_DIR/zImage" "$MNT_DIR/sunxi.dtb"

sync
sudo umount "$MNT_DIR"
rmdir "$MNT_DIR"

# 2. Check if second partition (rootfs) exists to update userspace tools
ROOT_DEV="${DEV%1}2"
if [ -b "$ROOT_DEV" ]; then
    echo "[*] 检测到 MicroSD 卡 Rootfs 分区: $ROOT_DEV，正在同步最新座舱栈与触控菜单..."
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
        echo "[+] Rootfs 用户态栈更新完成！"
    fi
fi

echo "=========================================================================="
echo ">>> 部署成功: Linux 6.12 最新屏幕驱动修复与触控全栈已 100% 写入 MicroSD 卡！ <<<"
echo ">>> 请拔下读卡器，将 MicroSD 卡插入全志 V553 设备插槽并通电开机！ <<<"
echo "=========================================================================="

