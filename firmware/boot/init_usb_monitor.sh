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

# Start ADB baseline with proper initialization delay
/usr/bin/setusbconfig adb
/bin/adbd -D &
sleep 0.5
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

# 6. FOREVER SUPERVISOR LOOP: Block rcS from EVER reaching S70app / cdr!
# 阻断原厂 rcS 脚本继续执行，彻底隔离原厂 cdr 行车记录仪应用
echo "[OK] Cockpit supervisor loop active (factory cdr fully blocked)..."
while true; do
    if [ -x /mnt/sdcard/fb_receiver ]; then
        /mnt/sdcard/fb_receiver
    fi
    sleep 1
done
