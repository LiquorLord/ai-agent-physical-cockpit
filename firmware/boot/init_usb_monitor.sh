#!/bin/sh
# ==============================================================================
# AGY Cockpit Dual-Boot Handover Engine / 双启动移交引擎
# ==============================================================================
exec > /mnt/sdcard/boot_debug.log 2>&1
set -x

echo "=========================================================="
echo " Starting AGY Physical Cockpit System (Standalone Buildroot)"
echo " Timestamp: $(date 2>/dev/null || echo boot)"
echo "=========================================================="

# 1. Start watchdog feeder in background to prevent hardware reset
(
    while true; do
        echo 1 > /dev/watchdog 2>/dev/null
        sleep 3
    done
) >/dev/null 2>&1 &
echo "[OK] Watchdog feeder active"

# 2. Display bootsplash immediately
if [ -x /mnt/sdcard/show_splash ] && [ -f /mnt/sdcard/splash.raw ]; then
    /mnt/sdcard/show_splash /mnt/sdcard/splash.raw 2>/dev/null || cat /mnt/sdcard/splash.raw > /dev/fb0 2>/dev/null
    echo "[OK] Bootsplash rendered"
fi

# 3. Handover rootfs to SD card
echo "[*] Handing over execution to Buildroot standalone userspace..."
cd /mnt/sdcard || exit 1
mkdir -p oldroot

mount --make-rprivate / 2>/dev/null || true
if pivot_root . oldroot 2>/dev/null; then
    echo "[OK] pivot_root successful!"
    mount -t proc none /proc 2>/dev/null || true
    mount -t sysfs none /sys 2>/dev/null || true
    mount -t devtmpfs none /dev 2>/dev/null || true
    mount -t tmpfs none /tmp 2>/dev/null || true
    mount -t tmpfs none /run 2>/dev/null || true
    echo "[OK] Executing /sbin/init via pivot_root..."
    exec chroot . /sbin/init <dev/console >dev/console 2>&1
else
    echo "[-] pivot_root failed, falling back to direct chroot..."
    mount -t proc none /mnt/sdcard/proc 2>/dev/null || true
    mount -t sysfs none /mnt/sdcard/sys 2>/dev/null || true
    mount -t devtmpfs none /mnt/sdcard/dev 2>/dev/null || true
    mount -t tmpfs none /mnt/sdcard/tmp 2>/dev/null || true
    mount -t tmpfs none /mnt/sdcard/run 2>/dev/null || true
    exec chroot /mnt/sdcard /sbin/init <dev/console >dev/console 2>&1
fi
