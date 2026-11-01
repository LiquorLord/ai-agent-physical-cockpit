#!/bin/bash
export DISPLAY="${DISPLAY:-:1}"
# Find Unknown20 or USB connector
OUT=$(xrandr --current | awk '/^(Unknown20-[0-9]+|USB-[0-9]+) connected/{print $1; exit}')
if [ -z "$OUT" ]; then
    echo "No GUD display found"
    exit 1
fi

CUR_MODE=$(xrandr --current | awk -v out="$OUT" '$1 == out {found=1; next} found && /^[ ]+[0-9]+x[0-9]+/ {if (/\*/) {print $1; exit}}')
echo "Current mode: $CUR_MODE"

if [ "$CUR_MODE" = "1600x600" ]; then
    echo "Switching to 1200x600 (Dual-Zone below HDMI-1)..."
    xrandr --output "$OUT" --mode 1200x600 --below HDMI-1
else
    echo "Switching to 1600x600 (Fullscreen below HDMI-1)..."
    xrandr --output "$OUT" --mode 1600x600 --below HDMI-1
fi
