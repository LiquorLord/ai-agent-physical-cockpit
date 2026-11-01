#!/bin/bash
# Universal GUD Mode Switcher for Sway / Wayland & X11
export SWAYSOCK=$(ls /run/user/$(id -u)/sway-ipc.*.sock 2>/dev/null | head -n 1)

# 1. Try Sway Wayland
if [ -n "$SWAYSOCK" ] && command -v swaymsg >/dev/null 2>&1; then
    # Detect GUD output in sway
    OUT=$(swaymsg -t get_outputs 2>/dev/null | jq -r '.[] | select(.model == "V553 Smart Cockpit (GUD + HID)" or (.name | test("^(Unknown|USB)"))) | .name' | head -n 1)
    if [ -n "$OUT" ]; then
        CUR_W=$(swaymsg -t get_outputs 2>/dev/null | jq -r ".[] | select(.name == \"$OUT\") | .current_mode.width")
        echo "[Sway] Detected GUD display $OUT, current width: $CUR_W"
        if [ "$CUR_W" = "1600" ]; then
            echo "[Sway] Switching to 1200x600 (Dual-Zone)..."
            swaymsg output "$OUT" mode 1200x600@60Hz
        else
            echo "[Sway] Switching to 1600x600 (Fullscreen)..."
            swaymsg output "$OUT" mode 1600x600@57.11Hz
        fi
        exit 0
    fi
fi

# 2. Fallback to X11 xrandr if running under Xwayland or Xorg
if command -v xrandr >/dev/null 2>&1 && [ -n "$DISPLAY" ]; then
    OUT=$(xrandr --current | awk '/^(Unknown|USB).* connected/{print $1; exit}')
    if [ -n "$OUT" ]; then
        CUR_MODE=$(xrandr --current | awk -v out="$OUT" '$1 == out {found=1; next} found && /^[ ]+[0-9]+x[0-9]+/ {if (/\*/) {print $1; exit}}')
        if [ "$CUR_MODE" = "1600x600" ]; then
            xrandr --output "$OUT" --mode 1200x600
        else
            xrandr --output "$OUT" --mode 1600x600
        fi
        exit 0
    fi
fi
