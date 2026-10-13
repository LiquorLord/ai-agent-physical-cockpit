#!/bin/sh
# ==============================================================================
# AGY Physical Cockpit: USB Composite Gadget ConfigFS Driver
# 全志 V553 物理级专属座舱 USB 免驱复合外设驱动配置 (NCM以太网 + HID键盘)
# ==============================================================================

GADGET_DIR="/sys/kernel/config/usb_gadget/g1"
CONFIG_DIR="$GADGET_DIR/configs/c.1"

# Dynamically locate UDC controller
UDC_CONTROLLER=$(ls /sys/class/udc 2>/dev/null | head -n 1)
[ -z "$UDC_CONTROLLER" ] && UDC_CONTROLLER="4100000.udc-controller"

echo "[*] Initializing AGY USB Composite Gadget on $UDC_CONTROLLER..."

# 1. Unbind UDC if currently bound
if [ -f "$GADGET_DIR/UDC" ]; then
    echo "" > "$GADGET_DIR/UDC" 2>/dev/null
    sleep 0.2
fi

# 2. Ensure ConfigFS mount and base gadget directory
[ -d /sys/kernel/config/usb_gadget ] || mount -t configfs none /sys/kernel/config
mkdir -p "$GADGET_DIR"

# 3. Setup USB IDs and Device Class
# Standard Linux Foundation Composite Gadget (0x1d6b / 0x0104)
echo 0x1d6b > "$GADGET_DIR/idVendor"
echo 0x0104 > "$GADGET_DIR/idProduct"
echo 0x0200 > "$GADGET_DIR/bcdUSB"
echo 0x0100 > "$GADGET_DIR/bcdDevice"
echo 0xef   > "$GADGET_DIR/bDeviceClass"     # Miscellaneous Device Class
echo 0x02   > "$GADGET_DIR/bDeviceSubClass"  # Common Class
echo 0x01   > "$GADGET_DIR/bDeviceProtocol"  # Interface Association Descriptor (IAD)

# 4. Device Strings
mkdir -p "$GADGET_DIR/strings/0x409"
echo "AGY Open Source Project"  > "$GADGET_DIR/strings/0x409/manufacturer"
echo "AGY Physical Cockpit"     > "$GADGET_DIR/strings/0x409/product"
echo "AGY20261013"              > "$GADGET_DIR/strings/0x409/serialnumber"

# 5. Configuration Descriptor
mkdir -p "$CONFIG_DIR"
echo 0x80 > "$CONFIG_DIR/bmAttributes" # Bus powered
echo 500  > "$CONFIG_DIR/MaxPower"     # 500 mA
mkdir -p "$CONFIG_DIR/strings/0x409"
echo "CDC-NCM Ethernet + HID Keyboard" > "$CONFIG_DIR/strings/0x409/configuration"

# 6. Clean existing function links from c.1
rm -f "$CONFIG_DIR"/* 2>/dev/null

# 7. Function: CDC-NCM Virtual Ethernet
mkdir -p "$GADGET_DIR/functions/ncm.usb0"
echo "42:69:70:68:79:01" > "$GADGET_DIR/functions/ncm.usb0/host_addr"
echo "42:69:70:68:79:02" > "$GADGET_DIR/functions/ncm.usb0/dev_addr"
ln -sf "$GADGET_DIR/functions/ncm.usb0" "$CONFIG_DIR/ncm.usb0"

# 8. Function: USB HID Keyboard (/dev/hidg0)
mkdir -p "$GADGET_DIR/functions/hid.usb0"
echo 1 > "$GADGET_DIR/functions/hid.usb0/subclass"  # Boot Interface
echo 1 > "$GADGET_DIR/functions/hid.usb0/protocol"  # Keyboard
echo 8 > "$GADGET_DIR/functions/hid.usb0/report_length"

# Standard 63-byte USB HID Keyboard Report Descriptor
printf '\x05\x01\x09\x06\xa1\x01\x05\x07\x19\xe0\x29\xe7\x15\x00\x25\x01\x75\x01\x95\x08\x81\x02\x95\x01\x75\x08\x81\x03\x95\x05\x75\x01\x05\x08\x19\x01\x29\x05\x91\x02\x95\x01\x75\x03\x91\x03\x95\x06\x75\x08\x15\x00\x25\x65\x05\x07\x19\x00\x29\x65\x81\x00\xc0' > "$GADGET_DIR/functions/hid.usb0/report_desc"
ln -sf "$GADGET_DIR/functions/hid.usb0" "$CONFIG_DIR/hid.usb0"

# 9. Bind UDC controller
echo "$UDC_CONTROLLER" > "$GADGET_DIR/UDC"
if [ $? -eq 0 ]; then
    echo "[OK] USB Composite Gadget bound to $UDC_CONTROLLER"
else
    echo "[WARN] Direct UDC bind failed, attempting retry..."
    sleep 0.5
    echo "$UDC_CONTROLLER" > "$GADGET_DIR/UDC"
fi

# 10. Configure Network Interface on device
sleep 0.5
ifconfig usb0 192.168.10.1 netmask 255.255.255.0 up 2>/dev/null
echo "[OK] Device IP configured: 192.168.10.1 on usb0"

echo "[OK] USB Composite Gadget initialized successfully!"
