#!/usr/bin/env bash
# ==============================================================================
# Allwinner V553 Device FEL Probe & Backup Helper
# 全志 V553 设备 FEL 探测与备份辅助脚本
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUNXI_FEL="${SCRIPT_DIR}/sunxi-tools/sunxi-fel"

if [[ ! -x "${SUNXI_FEL}" ]]; then
    echo "[!] Error: sunxi-fel binary not found at ${SUNXI_FEL}"
    echo "[!] 错误：未在 ${SUNXI_FEL} 找到可执行的 sunxi-fel"
    exit 1
fi

echo "========================================================"
echo " Allwinner FEL Device Probing / 全志 FEL 设备探测"
echo "========================================================"

# Check for USB ID 1f3a:efe8
if lsusb | grep -qi "1f3a:efe8"; then
    echo "[+] Found Allwinner FEL device via lsusb (1f3a:efe8)!"
    echo "[+] 通过 lsusb 检测到全志 FEL 模式设备 (1f3a:efe8)！"
else
    echo "[-] Device not found in FEL mode yet (1f3a:efe8 missing)."
    echo "[-] 尚未检测到 FEL 模式设备。"
    echo "    提示: 按住 Type-C 旁的物理按键不放，插上 USB 数据线，按住 2-3 秒后松开。"
fi

echo ""
echo "--- Testing sunxi-fel connection / 测试 sunxi-fel 连接 ---"
if "${SUNXI_FEL}" ver 2>/dev/null; then
    echo ""
    echo "[+] FEL communication SUCCESS! / FEL 通信成功！"
    echo ""
    echo "--- Checking SPI Flash / 检测 SPI Flash ---"
    "${SUNXI_FEL}" spiflash-info || echo "[-] No standard SPI Flash detected or needs custom DRAM initialization."
else
    echo "[-] sunxi-fel ver failed or device not ready."
fi
