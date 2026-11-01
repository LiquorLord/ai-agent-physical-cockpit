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

echo "[*] Querying SPI Flash info and detecting capacity..."
INFO_OUTPUT=$("${SUNXI_FEL}" spiflash-info 2>&1 || true)
echo "${INFO_OUTPUT}"
echo ""

# Dynamically parse capacity from JEDEC query (e.g., "size: 16777216 bytes")
DETECTED_SIZE=$(echo "${INFO_OUTPUT}" | grep -oiE "size: *[0-9]+" | awk '{print $2}' || true)
if [ -n "${DETECTED_SIZE}" ] && [ "${DETECTED_SIZE}" -gt 0 ]; then
    FLASH_SIZE=$(printf "0x%x" "${DETECTED_SIZE}")
    echo "[+] Dynamically detected Flash capacity: ${FLASH_SIZE} (${DETECTED_SIZE} bytes)"
else
    FLASH_SIZE="0x1000000"
    echo "[!] Could not parse exact size from info, defaulting to 16MB (${FLASH_SIZE})"
fi

BACKUP_FILE="${SCRIPT_DIR}/backup_factory_${FLASH_SIZE}.bin"
echo "[*] Dumping Flash image to ${BACKUP_FILE}, do not disconnect..."
"${SUNXI_FEL}" -p spiflash-read 0 "${FLASH_SIZE}" "${BACKUP_FILE}"

echo ""
echo "[+] Dump complete!"
ls -lh "${BACKUP_FILE}"
echo ""
echo "[*] Generating SHA256 checksum"
sha256sum "${BACKUP_FILE}" | tee "${BACKUP_FILE}.sha256"
echo ""
echo "[+] Factory backup secured!"
