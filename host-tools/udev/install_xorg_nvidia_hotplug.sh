#!/bin/bash
# ==============================================================================
# AI-Agent Physical Cockpit — Optional Xorg + NVIDIA PRIME Hotplug Installer
# Allwinner V553 smart cockpit — Xorg + NVIDIA dual-GPU hotplug auto-activation install script (optional)
#
# Note:
# - On Wayland (GNOME / KDE / Sway) or single-GPU Intel/AMD desktops, the V553
#   GUD USB Display works 100% plug-and-play without this script.
# - On legacy Xorg + NVIDIA proprietary driver desktops, Xorg registers hotplugged
#   USB DRM cards as unassociated Sink Outputs. This script installs a systemd+udev
#   hotplug service that automatically runs `xrandr --setprovideroutputsource`
#   and activates the 1200x600 extended display whenever the USB cable is plugged in.
# ==============================================================================
set -e

echo "[*] Installing /usr/local/bin/gud-hotplug.sh..."
sudo tee /usr/local/bin/gud-hotplug.sh > /dev/null << 'EOF'
#!/bin/bash
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
exec >/tmp/gud-hotplug.log 2>&1
echo "[*] gud-hotplug triggered at $(date)"

for attempt in 1 2 3 4 5 6; do
    sleep 1
    XPID=$(pgrep -f "/usr/lib/xorg/Xorg" | head -n1)
    [ -z "$XPID" ] && continue
    XDISP=$(tr '\0' '\n' < /proc/$XPID/cmdline | grep -E '^:[0-9]+$' | head -n1)
    XAUTH=$(tr '\0' '\n' < /proc/$XPID/cmdline | grep -A1 '^-auth$' | tail -n1)
    [ -z "$XDISP" ] && XDISP=":0"
    export DISPLAY="$XDISP"
    [ -n "$XAUTH" ] && export XAUTHORITY="$XAUTH"

    # If card1 is present on USB gud driver but Xorg ignored card1, hold card1 open and rebind gud so kernel allocates card2
    if ! xrandr --listproviders 2>/dev/null | grep -q "name:modesetting"; then
        if [ -e /dev/dri/card1 ] && [ "$attempt" -eq 2 ]; then
            GUD_INTF=$(basename "$(readlink -f /sys/class/drm/card1/device)" 2>/dev/null)
            if [ -n "$GUD_INTF" ] && [ -e "/sys/bus/usb/drivers/gud/$GUD_INTF" ]; then
                echo "[*] Xorg ignored card1; bouncing $GUD_INTF to allocate card2..."
                (sleep 3 < /dev/dri/card1 &)
                sleep 0.2
                echo "$GUD_INTF" > /sys/bus/usb/drivers/gud/unbind 2>/dev/null || true
                sleep 0.4
                echo "$GUD_INTF" > /sys/bus/usb/drivers/gud/bind 2>/dev/null || true
                sleep 1.0
            fi
        fi
    fi

    if xrandr --listproviders 2>/dev/null | grep -q "name:modesetting"; then
        xrandr --setprovideroutputsource modesetting NVIDIA-0 || \
        xrandr --setprovideroutputsource 1 0 || true
        OUT=$(xrandr | awk '/^(Unknown|USB).* connected/{print $1; exit}')
        if [ -n "$OUT" ]; then
            echo "[+] Enabling $OUT (1200x600 below HDMI-1)"
            XUSER=$(stat -c '%U' "/proc/$XPID" 2>/dev/null || echo "")
            XUID=$(stat -c '%u' "/proc/$XPID" 2>/dev/null || echo "1000")
            if [ -n "$XUSER" ]; then
                runuser -u "$XUSER" -- env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$XUID/bus" DISPLAY="$DISPLAY" XAUTHORITY="$XAUTHORITY" xfconf-query -c displays -p "/Default/$OUT/Scale/X" -s 1.0 2>/dev/null || true
                runuser -u "$XUSER" -- env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$XUID/bus" DISPLAY="$DISPLAY" XAUTHORITY="$XAUTHORITY" xfconf-query -c displays -p "/Default/$OUT/Scale/Y" -s 1.0 2>/dev/null || true
                runuser -u "$XUSER" -- env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$XUID/bus" DISPLAY="$DISPLAY" XAUTHORITY="$XAUTHORITY" xfconf-query -c displays -p "/Fallback/$OUT/Scale/X" -s 1.0 2>/dev/null || true
                runuser -u "$XUSER" -- env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$XUID/bus" DISPLAY="$DISPLAY" XAUTHORITY="$XAUTHORITY" xfconf-query -c displays -p "/Fallback/$OUT/Scale/Y" -s 1.0 2>/dev/null || true
            fi
            xrandr --output "$OUT" --mode 1200x600 --below HDMI-1 || \
            xrandr --output "$OUT" --mode 1200x600 --auto || true
            break
        fi
    fi
done
EOF
sudo chmod +x /usr/local/bin/gud-hotplug.sh

echo "[*] Installing /etc/systemd/system/gud-hotplug.service..."
sudo tee /etc/systemd/system/gud-hotplug.service > /dev/null << 'EOF'
[Unit]
Description=GUD USB Display PRIME Hotplug Auto-Setup

[Service]
Type=oneshot
ExecStart=/usr/local/bin/gud-hotplug.sh
EOF

echo "[*] Installing /etc/udev/rules.d/99-gud-display.rules..."
sudo tee /etc/udev/rules.d/99-gud-display.rules > /dev/null << 'EOF'
SUBSYSTEM=="drm", ACTION=="add", KERNEL=="card[0-9]*", TAG+="systemd", ENV{SYSTEMD_WANTS}="gud-hotplug.service"
EOF

sudo systemctl daemon-reload
sudo udevadm control --reload-rules
echo "[+] Xorg + NVIDIA GUD hotplug service installed successfully!"
