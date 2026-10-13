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

# Setup 3-in-1 Composite USB Gadget (ADB + CDC-NCM Ethernet + USB HID Keyboard)
GADGET_DIR="/sys/kernel/config/usb_gadget/g1"
CONFIG_DIR="$GADGET_DIR/configs/c.1"

[ -d /sys/kernel/config/usb_gadget ] || mount -t configfs none /sys/kernel/config
mkdir -p "$GADGET_DIR"

# 1. Base USB IDs & IAD Composite Device Class
echo 0x18d1 > "$GADGET_DIR/idVendor"
echo 0xd002 > "$GADGET_DIR/idProduct"
echo 0x0200 > "$GADGET_DIR/bcdUSB"
echo 0x0100 > "$GADGET_DIR/bcdDevice"
echo 0xef   > "$GADGET_DIR/bDeviceClass"
echo 0x02   > "$GADGET_DIR/bDeviceSubClass"
echo 0x01   > "$GADGET_DIR/bDeviceProtocol"

mkdir -p "$GADGET_DIR/strings/0x409"
echo "Allwinner Technology Inc." > "$GADGET_DIR/strings/0x409/manufacturer"
echo "AGY Physical Cockpit"      > "$GADGET_DIR/strings/0x409/product"
echo "20080411"                  > "$GADGET_DIR/strings/0x409/serialnumber"

# 2. Configuration Descriptor
mkdir -p "$CONFIG_DIR"
echo 0x80 > "$CONFIG_DIR/bmAttributes"
echo 500  > "$CONFIG_DIR/MaxPower"
mkdir -p "$CONFIG_DIR/strings/0x409"
echo "ADB + CDC-NCM Ethernet + HID Keyboard" > "$CONFIG_DIR/strings/0x409/configuration"

# 3. Create Function: ffs.adb
mkdir -p "$GADGET_DIR/functions/ffs.adb"
[ -d /dev/usb-ffs/adb ] || {
    mkdir -p /dev/usb-ffs/adb
    mount -o uid=2000,gid=2000 -t functionfs adb /dev/usb-ffs/adb/
}
ln -sf "$GADGET_DIR/functions/ffs.adb" "$CONFIG_DIR/ffs.adb"

# 4. Create Function: ncm.usb0 (CDC-NCM Virtual Ethernet)
mkdir -p "$GADGET_DIR/functions/ncm.usb0"
echo "42:69:70:68:79:01" > "$GADGET_DIR/functions/ncm.usb0/host_addr"
echo "42:69:70:68:79:02" > "$GADGET_DIR/functions/ncm.usb0/dev_addr"
ln -sf "$GADGET_DIR/functions/ncm.usb0" "$CONFIG_DIR/ncm.usb0"

# 5. Create Function: hid.usb0 (USB HID Keyboard /dev/hidg0)
mkdir -p "$GADGET_DIR/functions/hid.usb0"
echo 1 > "$GADGET_DIR/functions/hid.usb0/subclass"
echo 1 > "$GADGET_DIR/functions/hid.usb0/protocol"
echo 8 > "$GADGET_DIR/functions/hid.usb0/report_length"
printf '\x05\x01\x09\x06\xa1\x01\x05\x07\x19\xe0\x29\xe7\x15\x00\x25\x01\x75\x01\x95\x08\x81\x02\x95\x01\x75\x08\x81\x03\x95\x05\x75\x01\x05\x08\x19\x01\x29\x05\x91\x02\x95\x01\x75\x03\x91\x03\x95\x06\x75\x08\x15\x00\x25\x65\x05\x07\x19\x00\x29\x65\x81\x00\xc0' > "$GADGET_DIR/functions/hid.usb0/report_desc"
ln -sf "$GADGET_DIR/functions/hid.usb0" "$CONFIG_DIR/hid.usb0"

# 6. Start ADB daemon to initialize FunctionFS endpoints
/bin/adbd -D &
sleep 0.5

# 7. Bind UDC controller with all 3 functions active
echo 4100000.udc-controller > "$GADGET_DIR/UDC" 2>/dev/null
echo "[OK] 3-in-1 Composite USB Gadget active (ADB + NCM + HID)"

sleep 0.5
ifconfig usb0 192.168.10.1 netmask 255.255.255.0 up 2>/dev/null

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

# 5. Enable Telnet Daemon for network debug
telnetd -l /bin/sh 2>/dev/null &

# 6. Launch Multimodal Interaction Daemons (Touch Action Bar + Visual Gesture AI)
if [ -x /mnt/sdcard/touch_cockpit ]; then
    /mnt/sdcard/touch_cockpit >/tmp/touch.log 2>&1 &
    echo "[OK] Touch Cockpit controller active"
fi

if [ -x /mnt/sdcard/gesture_detector ]; then
    /mnt/sdcard/gesture_detector >/tmp/gesture.log 2>&1 &
    echo "[OK] Visual Gesture Detector active"
fi

# 7. FOREVER SUPERVISOR LOOP: Block rcS from EVER reaching S70app / cdr!
# 阻断原厂 rcS 脚本继续执行，彻底隔离原厂 cdr 行车记录仪应用
echo "[OK] Cockpit supervisor loop active (factory cdr fully blocked)..."
while true; do
    if [ -x /mnt/sdcard/fb_receiver ]; then
        /mnt/sdcard/fb_receiver
    fi
    sleep 1
done
