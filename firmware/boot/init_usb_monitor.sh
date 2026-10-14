#!/bin/sh
# ============================================================================
# AGY Physical Cockpit Boot Engine / 全志 V553 物理级专属座舱启动引擎
# Dual-Boot Mode: Launched exclusively when MicroSD card is inserted
# ============================================================================

# 0. ALL OUTPUT REDIRECTED TO MICROSD CARD FOR 100% PERSISTENT AUDITING
exec > /mnt/sdcard/boot_debug.log 2>&1
set -x

echo "=========================================================="
echo " Starting AGY Physical Cockpit System (1600x600 Ultra-Wide)"
echo " Timestamp: $(date 2>/dev/null || echo 'boot')"
echo "=========================================================="

# 1. Hardware Watchdog Feeder / 硬件看门狗喂狗守护（静默运行）
(
    while true; do
        echo 1 > /dev/watchdog 2>/dev/null
        sleep 3
    done
) >/dev/null 2>&1 &
echo "[OK] Watchdog feeder active"

# 2. System Subsystem Initialization / 启动基础设备管理守护
echo "[*] Initializing system udev and device nodes..."
if [ -x /etc/init.d/S10udev ]; then
    /etc/init.d/S10udev start
fi
if [ -x /etc/init.d/S11dev ]; then
    /etc/init.d/S11dev start
fi

# 3. Spawn Interactive Root Shell on Serial Console / 串口终端始终保持交互式 Shell
/bin/sh </dev/console >/dev/console 2>&1 &
echo "[OK] Interactive root shell spawned on /dev/console"

# 4. Configure Display Engine & Activate Custom Bootsplash / 激活 1600x600 专属开机画面
echo 0 > /sys/devices/virtual/disp/disp/attr/disp 2>/dev/null
if [ -x /mnt/sdcard/show_splash ]; then
    /mnt/sdcard/show_splash /mnt/sdcard/splash.raw
    echo "[OK] 1600x600 Custom Bootsplash activated"
elif [ -f /mnt/sdcard/splash.raw ]; then
    cat /mnt/sdcard/splash.raw > /dev/fb0
fi

# 5. USB 2.0 High-Speed Gadget Activation (Factory-Proven setusbconfig adb)
echo "[*] Initializing USB OTG Controller..."
cat /sys/devices/platform/soc/usbc0/usb_device >/dev/null 2>&1

# Reset any existing gadget UDC binding
echo "" > /sys/kernel/config/usb_gadget/g1/UDC 2>/dev/null

# Call factory setusbconfig
if [ -x /usr/bin/setusbconfig ]; then
    echo "[*] Calling /usr/bin/setusbconfig adb..."
    /usr/bin/setusbconfig adb
else
    echo "[!] /usr/bin/setusbconfig not found, using manual configfs adb..."
    mkdir -p /sys/kernel/config/usb_gadget/g1
    echo 0x18d1 > /sys/kernel/config/usb_gadget/g1/idVendor
    echo 0xd002 > /sys/kernel/config/usb_gadget/g1/idProduct
    mkdir -p /sys/kernel/config/usb_gadget/g1/strings/0x409
    echo "Allwinner Technology Inc." > /sys/kernel/config/usb_gadget/g1/strings/0x409/manufacturer
    echo "AGY Physical Cockpit" > /sys/kernel/config/usb_gadget/g1/strings/0x409/product
    mkdir -p /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409
    mkdir -p /sys/kernel/config/usb_gadget/g1/functions/ffs.adb
    mkdir -p /dev/usb-ffs/adb
    mount -t functionfs adb /dev/usb-ffs/adb/ 2>/dev/null
    ln -sf /sys/kernel/config/usb_gadget/g1/functions/ffs.adb /sys/kernel/config/usb_gadget/g1/configs/c.1/ffs.adb
fi

# Kill any stale adbd and restart
killall -9 adbd 2>/dev/null
/bin/adbd -D &

# Detect available UDC name
UDC_CONTROLLER=$(ls /sys/class/udc 2>/dev/null | head -n 1)
[ -z "$UDC_CONTROLLER" ] && UDC_CONTROLLER="4100000.udc-controller"
echo "[*] Detected UDC Controller: $UDC_CONTROLLER"

# Robust UDC bind retry loop (enables internal 1.5k D+ pullup)
echo "[*] Binding UDC Controller to activate USB D+ pullup..."
for i in 1 2 3 4 5 6 7 8 9 10; do
    sleep 0.3
    echo "$UDC_CONTROLLER" > /sys/kernel/config/usb_gadget/g1/UDC 2>&1
    if [ $? -eq 0 ]; then
        echo "[OK] USB Gadget successfully bound to $UDC_CONTROLLER (attempt $i)"
        break
    else
        echo "[-] UDC bind attempt $i failed, retrying..."
    fi
done

echo "[*] Current UDC Binding Status:"
cat /sys/kernel/config/usb_gadget/g1/UDC 2>&1
ls -la /sys/class/udc 2>&1

# Fix adb_shell for seamless non-interactive execution
cat << "EOF_SHELL" > /tmp/adb_shell
#!/bin/sh
export ENV="/etc/adb_profile"
if [ "$1" = "-c" ]; then
    shift
    exec /bin/sh -c "$*"
elif [ -n "$1" ]; then
    exec /bin/sh -c "$*"
else
    exec /bin/sh
fi
EOF_SHELL
chmod +x /tmp/adb_shell
mount --bind /tmp/adb_shell /bin/adb_shell 2>/dev/null

# 6. Enable Telnet Daemon for network debug
telnetd -l /bin/sh 2>/dev/null &

# 7. Create Event FIFO for Host Communication
mkfifo /tmp/cockpit_event.fifo 2>/dev/null
chmod 666 /tmp/cockpit_event.fifo 2>/dev/null

# 8. Launch Multimodal Interaction Daemons (Touch Action Bar + Visual Gesture AI)
if [ -x /mnt/sdcard/touch_cockpit ]; then
    /mnt/sdcard/touch_cockpit >/tmp/touch.log 2>&1 &
    echo "[OK] Touch Cockpit controller active"
fi

if [ -x /mnt/sdcard/gesture_detector ]; then
    /mnt/sdcard/gesture_detector >/tmp/gesture.log 2>&1 &
    echo "[OK] Visual Gesture Detector active"
fi

# 9. Dump full dmesg snapshot to SD card
dmesg > /mnt/sdcard/dmesg_boot.log
sync

echo "[OK] System boot sequence completed. Entering supervisor loop."

# 10. FOREVER SUPERVISOR LOOP: Block rcS from EVER reaching S70app / cdr!
# 阻断原厂 rcS 脚本继续执行，彻底隔离原厂 cdr 行车记录仪应用
LOOP_COUNT=0
while true; do
    if [ -x /mnt/sdcard/fb_receiver ]; then
        /mnt/sdcard/fb_receiver
    fi
    LOOP_COUNT=$((LOOP_COUNT + 1))
    if [ $((LOOP_COUNT % 10)) -eq 0 ]; then
        sync
    fi
    sleep 1
done
