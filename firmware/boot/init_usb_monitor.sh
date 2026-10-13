#!/bin/sh
# ============================================================================
# Allwinner V553 Custom USB Monitor Boot Engine / 全志 V553 USB 显示器启动引擎
# Dual-Boot Mode: Launched exclusively when MicroSD card is inserted
# ============================================================================

echo "=========================================================="
echo " Starting Custom System: USB Monitor Pipeline (1600x600)  "
echo "=========================================================="

# 1. Hardware Watchdog Feeder / 硬件看门狗喂狗守护（防止系统复位）
(
    while true; do
        echo 1 > /dev/watchdog 2>/dev/null
        sleep 3
    done
) &
WDT_PID=$!
echo "[OK] Watchdog feeder active (PID: $WDT_PID)"

# 2. Configure Display Engine / 配置显示引擎，释放摄像头图层
echo 0 > /sys/devices/virtual/disp/disp/attr/disp 2>/dev/null

# 3. Blit Boot Splash Screen / 绘制开机专属仪表盘画面 (1600x600 RGB565)
if [ -f /mnt/sdcard/splash.raw ]; then
    cat /mnt/sdcard/splash.raw > /dev/fb0
    echo "[OK] 1600x600 Splash Screen loaded onto /dev/fb0"
fi

# 4. Activate USB High-Speed OTG Composite Device (CDC-NCM + HID + ADB)
cat /sys/devices/platform/soc/usbc0/usb_device >/dev/null 2>&1
/bin/adbd -D &
if [ -x /mnt/sdcard/setup_gadget.sh ]; then
    /mnt/sdcard/setup_gadget.sh
else
    /usr/bin/setusbconfig adb
    echo 4100000.udc-controller > /sys/kernel/config/usb_gadget/g1/UDC 2>/dev/null
    echo "[OK] USB OTG ADB active & bound to UDC controller"
fi

# 5. Fix adb_shell for seamless non-interactive execution
cat << EOF_SHELL > /tmp/adb_shell
#!/bin/sh
export ENV="/etc/adb_profile"
if [ "$1" = "-c" ]; then
    shift
    exec /bin/sh -c "$*"
else
    exec /bin/sh
fi
EOF_SHELL
chmod +x /tmp/adb_shell
mount --bind /tmp/adb_shell /bin/adb_shell 2>/dev/null

# 6. Start Native TCP Display Stream Server / 启动原生高性能显示接收器
if [ -x /mnt/sdcard/fb_receiver ]; then
    echo "[OK] Starting /mnt/sdcard/fb_receiver on :5900..."
    exec /mnt/sdcard/fb_receiver
fi
