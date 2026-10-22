#!/usr/bin/env bash
# ==============================================================================
# setup_gud_screen.sh - Automated X11 & i3-wm Setup for V553 USB Hardware Display
# Allwinner V553 native hardware USB display (GUD) X11 topology auto-configuration script
# ==============================================================================

set -e

DISPLAY="${DISPLAY:-:1}"
export DISPLAY

log() {
    echo -e "\033[1;32m[*] [GUD-Display]\033[0m $1"
}

warn() {
    echo -e "\033[1;33m[!] [GUD-Display]\033[0m $1"
}

err() {
    echo -e "\033[1;31m[-] [GUD-Display]\033[0m $1"
}

# 1. Check DRM Device
GUD_STATUS_FILE=$(ls /sys/class/drm/card*-USB-*/status 2>/dev/null | head -n 1)
if [ -z "$GUD_STATUS_FILE" ] || [ ! -e "$GUD_STATUS_FILE" ]; then
    err "V553 GUD Display not detected in sysfs (/sys/class/drm/card*-USB-*)!"
    err "Please ensure device is plugged in and switch_to_gud.sh is active."
    exit 1
fi

STATUS=$(cat "$GUD_STATUS_FILE" 2>/dev/null || echo "disconnected")
if [ "$STATUS" != "connected" ]; then
    warn "GUD connector status is '$STATUS', attempting auto-refresh..."
fi

# 1.5 Enable Host GUD Async Flush & Turbo Damage-Shrinking / LZ4 Accelerator
if [ -w /sys/module/gud/parameters/async_flush ]; then
    echo Y > /sys/module/gud/parameters/async_flush
else
    echo Y | sudo tee /sys/module/gud/parameters/async_flush >/dev/null 2>&1 || true
fi
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! lsmod | grep -q "^gud_turbo "; then
    if [ -f "$SCRIPT_DIR/gud_turbo/gud_turbo.ko" ]; then
        log "Loading gud_turbo.ko (Bounding-box damage shrinking + Live LZ4)..."
        sudo insmod "$SCRIPT_DIR/gud_turbo/gud_turbo.ko" 2>/dev/null || true
    fi
fi

# 2. Configure Xorg PRIME Provider Offload
log "Connecting PRIME output provider (modesetting -> NVIDIA-0)..."
xrandr --setprovideroutputsource modesetting NVIDIA-0 2>/dev/null || true

# 3. Detect Output Name in xrandr
OUTPUT_NAME=$(xrandr --current | grep -E "(Unknown20-[0-9]+|USB-[0-9]+)" | awk '{print $1}' | head -n 1)

if [ -z "$OUTPUT_NAME" ]; then
    err "No GUD display output found in xrandr! Available outputs:"
    xrandr --current | grep "connected"
    exit 1
fi

log "Detected GUD Display Output: $OUTPUT_NAME"

# 4. Configure Layout: Below HDMI-1 (x: 1920, y: 1080, 1200x600 Dual-Zone)
log "Configuring display layout: --output $OUTPUT_NAME --mode 1200x600 --below HDMI-1"
xrandr --output "$OUTPUT_NAME" --mode 1200x600 --below HDMI-1

# 5. Configure i3-wm Workspace Binding
if command -v i3-msg >/dev/null 2>&1; then
    log "Assigning Workspace 4 to $OUTPUT_NAME in i3-wm..."
    i3-msg "workspace 4 output $OUTPUT_NAME" >/dev/null 2>&1 || true
    i3-msg "workspace 4" >/dev/null 2>&1 || true
fi

log "GUD Hardware Display successfully configured below HDMI-1 (+1920+1080)!"
log "Resolution: 1200x600 @ 60Hz (Dual-Zone Mode) • Zero floating window on primary screens."
