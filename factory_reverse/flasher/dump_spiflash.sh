#!/usr/bin/env bash
# ==============================================================================
# Allwinner V553 SPI Flash Auto Backup Tool
# 全志 V553 SPI NOR Flash 固件自动导出与校验工具
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUNXI_FEL="${SCRIPT_DIR}/sunxi-tools/sunxi-fel"
FLASH_SIZE="0x1000000" # 16 MBytes (XM25QH128C)
BACKUP_FILE="${SCRIPT_DIR}/backup_factory_16MB.bin"

echo "========================================================="
echo " Allwinner V553 Flash Dumper / 全志 V553 固件全盘备份"
echo "========================================================="

echo "[*] Waiting for FEL device (1f3a:efe8)... / 等待 FEL 设备接入..."
while ! lsusb | grep -qi "1f3a:efe8"; do
    sleep 0.5
done

echo "[+] FEL device detected! / 检测到 FEL 设备！"
"${SUNXI_FEL}" -v ver
echo ""

echo "[*] Querying SPI Flash info... / 查询 SPI Flash 芯片信息..."
"${SUNXI_FEL}" spiflash-info || true
echo ""

echo "[*] Dumping 16MB SPI Flash to ${BACKUP_FILE}..."
echo "[*] 正在导出 16MB 完整 Flash 镜像到 ${BACKUP_FILE}，请勿断开连接..."
"${SUNXI_FEL}" -p spiflash-read 0 "${FLASH_SIZE}" "${BACKUP_FILE}"

echo ""
echo "[+] Dump complete! / 固件导出成功！"
ls -lh "${BACKUP_FILE}"
echo ""
echo "[*] Generating SHA256 checksum / 计算校验和:"
sha256sum "${BACKUP_FILE}" | tee "${BACKUP_FILE}.sha256"
echo ""
echo "[+] Factory backup secured! / 原厂固件备份完毕！"
