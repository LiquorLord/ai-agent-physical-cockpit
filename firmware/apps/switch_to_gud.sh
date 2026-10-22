#!/bin/sh
# Switch V553 to GUD USB Display (1200x600) + USB HID Keyboard Composite Gadget
# Switch V553 into a GUD native USB display (1200x600) + USB HID keyboard composite device (pure, no network)

LOG="/tmp/gud_switch.log"
exec > "$LOG" 2>&1

echo "========================================================"
echo "[*] Starting Pure GUD Display + HID Keyboard Switch at $(date)..."
echo "========================================================"

# Stop any running instances
killall -9 gud_v553 touch_cockpit 2>/dev/null
sleep 0.2

CONFIGFS_DIR=/sys/kernel/config
GADGET_DIR=$CONFIGFS_DIR/usb_gadget/g1
UDC_DEV=$(ls /sys/class/udc 2>/dev/null | head -n 1)
[ -z "$UDC_DEV" ] && UDC_DEV="4100000.usb"

echo "[*] Step 1: Unbinding UDC..."
echo "" > $GADGET_DIR/UDC 2>/dev/null
sleep 0.5

# Clean old functions from configuration
rm -f $GADGET_DIR/configs/c.1/ncm.* 2>/dev/null
rm -f $GADGET_DIR/configs/c.1/ffs.* 2>/dev/null
rm -f $GADGET_DIR/configs/c.1/hid.* 2>/dev/null

echo "[*] Step 2: Configuring GUD (OpenMoko) VID/PID & Composite Descriptors..."
mkdir -p $GADGET_DIR
mkdir -p $GADGET_DIR/configs/c.1
echo 0x1d50 > $GADGET_DIR/idVendor
echo 0x614d > $GADGET_DIR/idProduct
echo 0x0200 > $GADGET_DIR/bcdUSB

# Multi-Interface Device Class for composite GUD Display + HID Keyboard
echo 0xef > $GADGET_DIR/bDeviceClass
echo 0x02 > $GADGET_DIR/bDeviceSubClass
echo 0x01 > $GADGET_DIR/bDeviceProtocol

mkdir -p $GADGET_DIR/strings/0x409
echo "0123456789ABCDEF" > $GADGET_DIR/strings/0x409/serialnumber
echo "Ternary Systems" > $GADGET_DIR/strings/0x409/manufacturer
echo "V553 Smart Cockpit (GUD + HID)" > $GADGET_DIR/strings/0x409/product

mkdir -p $GADGET_DIR/configs/c.1/strings/0x409
echo "GUD Display + HID Keyboard" > $GADGET_DIR/configs/c.1/strings/0x409/configuration
echo 500 > $GADGET_DIR/configs/c.1/MaxPower

echo "[*] Step 3: Setting up Functions (GUD Display + HID Keyboard)..."
# Function 1: FunctionFS for GUD Display
mkdir -p $GADGET_DIR/functions/ffs.gud
ln -sf $GADGET_DIR/functions/ffs.gud $GADGET_DIR/configs/c.1/ 2>/dev/null

# Function 2: Standard USB HID Keyboard
mkdir -p $GADGET_DIR/functions/hid.usb0
echo 1 > $GADGET_DIR/functions/hid.usb0/subclass      # Boot interface
echo 1 > $GADGET_DIR/functions/hid.usb0/protocol      # Keyboard
echo 8 > $GADGET_DIR/functions/hid.usb0/report_length
# Standard 8-byte Keyboard HID Report Descriptor (63 bytes)
printf '\x05\x01\x09\x06\xa1\x01\x05\x07\x19\xe0\x29\xe7\x15\x00\x25\x01\x75\x01\x95\x08\x81\x02\x95\x01\x75\x08\x81\x01\x95\x05\x75\x01\x05\x08\x19\x01\x29\x05\x91\x02\x95\x01\x75\x03\x91\x01\x95\x06\x75\x08\x15\x00\x25\x65\x05\x07\x19\x00\x29\x65\x81\x00\xc0' > "$GADGET_DIR/functions/hid.usb0/report_desc"
ln -sf "$GADGET_DIR/functions/hid.usb0" "$GADGET_DIR/configs/c.1/" 2>/dev/null

echo "[*] Step 4: Mounting FunctionFS..."
mkdir -p /dev/ffs-gud
umount /dev/ffs-gud 2>/dev/null || true
if ! mount -t functionfs gud /dev/ffs-gud; then
    echo "[-] Failed to mount FunctionFS!"
    exit 1
fi

echo "[*] Step 5: Launching gud_v553 daemon..."
/usr/bin/gud_v553 > /tmp/gud_v553.log 2>&1 &
GUD_PID=$!
echo "[+] gud_v553 launched with PID $GUD_PID"

# Wait for descriptors to be written to ep0
sleep 0.8

echo "[*] Step 6: Binding to UDC ($UDC_DEV)..."
BOUND=0
for i in 1 2 3 4 5; do
    if echo "$UDC_DEV" > $GADGET_DIR/UDC 2>/dev/null; then
        echo "[+] Successfully bound composite gadget to $UDC_DEV!"
        BOUND=1
        break
    fi
    echo "[*] Retrying UDC bind ($i/5)..."
    sleep 0.5
done

if [ $BOUND -eq 0 ]; then
    echo "[-] UDC bind failed!"
    exit 1
fi

echo "[*] Step 7: Launching touch_cockpit Action Bar..."
/usr/bin/touch_cockpit > /tmp/touch_cockpit.log 2>&1 &

echo "[+] Pure GUD USB Display (1200x600) + USB HID Keyboard is active and operational!"
echo "========================================================"
