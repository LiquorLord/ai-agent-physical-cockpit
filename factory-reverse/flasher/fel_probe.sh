#!/usr/bin/env bash
# ==============================================================================
# Allwinner V553 Device FEL Probe & Backup Helper
# Allwinner V553 device FEL probing and backup helper script
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUNXI_FEL="${SCRIPT_DIR}/sunxi-tools/sunxi-fel"

if [[ ! -x "${SUNXI_FEL}" ]]; then
    echo "[!] Error: sunxi-fel binary not found at ${SUNXI_FEL}"
    echo "[!] Error: no executable sunxi-fel found at ${SUNXI_FEL}"
    exit 1
fi

echo "========================================================"
echo " Allwinner FEL Device Probing"
echo "========================================================"

# Check for USB ID 1f3a:efe8
if lsusb | grep -qi "1f3a:efe8"; then
    echo "[+] Found Allwinner FEL device via lsusb (1f3a:efe8)!"
    echo "[+] Detected Allwinner FEL-mode device (1f3a:efe8) via lsusb!"
else
    echo "[-] Device not found in FEL mode yet (1f3a:efe8 missing)."
    echo "[-] No FEL-mode device detected yet."
    echo "    Hint: hold the physical button next to Type-C, plug in the USB data cable, keep holding for 2-3 seconds then release."
fi

echo ""
echo "--- Testing sunxi-fel connection ---"
if "${SUNXI_FEL}" ver 2>/dev/null; then
    echo ""
    echo "[+] FEL communication SUCCESS!"
    echo ""
    echo "--- Checking SPI Flash ---"
    "${SUNXI_FEL}" spiflash-info || echo "[-] No standard SPI Flash detected or needs custom DRAM initialization."
else
    echo "[-] sunxi-fel ver failed or device not ready."
fi
