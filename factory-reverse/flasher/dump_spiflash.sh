#!/usr/bin/env bash
# ==============================================================================
# Allwinner V553 SPI Flash Auto Backup Tool
# Allwinner V553 SPI NOR Flash firmware auto-dump and verify tool
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUNXI_FEL="${SCRIPT_DIR}/sunxi-tools/sunxi-fel"
FLASH_SIZE="0x1000000" # 16 MBytes (XM25QH128C)
BACKUP_FILE="${SCRIPT_DIR}/backup_factory_16MB.bin"

echo "========================================================="
echo " Allwinner V553 Flash Dumper"
echo "========================================================="

echo "[*] Waiting for FEL device (1f3a:efe8)..."
while ! lsusb | grep -qi "1f3a:efe8"; do
    sleep 0.5
done

echo "[+] FEL device detected!"
"${SUNXI_FEL}" -v ver
echo ""

echo "[*] Querying SPI Flash info..."
"${SUNXI_FEL}" spiflash-info || true
echo ""

echo "[*] Dumping 16MB SPI Flash to ${BACKUP_FILE}..."
echo "[*] Dumping the full 16MB Flash image to ${BACKUP_FILE}, do not disconnect..."
"${SUNXI_FEL}" -p spiflash-read 0 "${FLASH_SIZE}" "${BACKUP_FILE}"

echo ""
echo "[+] Dump complete!"
ls -lh "${BACKUP_FILE}"
echo ""
echo "[*] Generating SHA256 checksum"
sha256sum "${BACKUP_FILE}" | tee "${BACKUP_FILE}.sha256"
echo ""
echo "[+] Factory backup secured!"
