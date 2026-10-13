#!/bin/sh
# ============================================================================
# AGY Physical Cockpit Boot Engine / 全志 V553 物理级专属座舱启动引擎
# Dual-Boot Mode: Launched exclusively when MicroSD card is inserted
# ============================================================================

echo "=========================================================="
echo " Starting AGY Physical Cockpit System (1600x600 Ultra-Wide)"
echo "=========================================================="

# 1. Hardware Watchdog Feeder / 硬件看门狗喂狗守护（静默运行）
(
    while true; do
        echo 1 > /dev/watchdog 2>/dev/null
        sleep 3
    done
) >/dev/null 2>&1 &
echo "[OK] Watchdog feeder active"

# 2. Spawn Interactive Root Shell on Serial Console / 串口终端始终保持交互式 Shell
/bin/sh </dev/console >/dev/console 2>&1 &
echo "[OK] Interactive root shell spawned on /dev/console"

# 3. Configure Display Engine & Activate Custom Bootsplash / 激活 1600x600 专属开机画面
echo 0 > /sys/devices/virtual/disp/disp/attr/disp 2>/dev/null
if [ -x /mnt/sdcard/show_splash ]; then
    /mnt/sdcard/show_splash /mnt/sdcard/splash.raw
    echo "[OK] 1600x600 Custom Bootsplash activated"
elif [ -f /mnt/sdcard/splash.raw ]; then
    cat /mnt/sdcard/splash.raw > /dev/fb0
fi

# 4. Activate USB High-Speed OTG Device / 激活 USB 高速从机模式
cat /sys/devices/platform/soc/usbc0/usb_device >/dev/null 2>&1

# Always ensure base ADB connectivity first
/usr/bin/setusbconfig adb
echo 4100000.udc-controller > /sys/kernel/config/usb_gadget/g1/UDC 2>/dev/null
echo "[OK] Baseline USB ADB active"

# Fix adb_shell for seamless non-interactive execution
cat << EOF_SHELL > /tmp/adb_shell
#!/bin/sh
export ENV="/etc/adb_profile"
if [ "$1" = "-c" ]; then
    shift
    exec /bin/sh -c "\$*"
else
    exec /bin/sh
fi
EOF_SHELL
chmod +x /tmp/adb_shell
mount --bind /tmp/adb_shell /bin/adb_shell 2>/dev/null

# 5. Enable Telnet Daemon for network debug
telnetd -l /bin/sh 2>/dev/null &

# 6. Try upgrading to full USB Composite Gadget (CDC-NCM + HID) in background
if [ -x /mnt/sdcard/setup_gadget.sh ]; then
    (
        sleep 2
        /mnt/sdcard/setup_gadget.sh > /tmp/gadget.log 2>&1
    ) &
fi

# 7. Start Native TCP Display Stream Server / 启动原生高性能显示接收器
if [ -x /mnt/sdcard/fb_receiver ]; then
    echo "[OK] Starting /mnt/sdcard/fb_receiver on :5900..."
    /mnt/sdcard/fb_receiver &
fi

echo "[OK] Cockpit boot initialization complete!"
