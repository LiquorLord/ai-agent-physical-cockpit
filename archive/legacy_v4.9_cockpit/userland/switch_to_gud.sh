#!/bin/sh
# Switch V553 to GUD USB Display + CDC-NCM Composite Gadget
# 切换 V553 为 GUD 原生 USB 显卡 + CDC-NCM 复合设备

LOG="/tmp/gud_switch.log"
exec > "$LOG" 2>&1

echo "========================================================"
echo "[*] Starting GUD Composite Switch at $(date)..."
echo "========================================================"

# Stop old receivers
killall -9 fb_receiver gud_v553 2>/dev/null
sleep 0.2

CONFIGFS_DIR=/sys/kernel/config
GADGET_DIR=$CONFIGFS_DIR/usb_gadget/g1
UDC_DEV=$(ls /sys/class/udc 2>/dev/null | head -n 1)
[ -z "$UDC_DEV" ] && UDC_DEV="4100000.udc-controller"

echo "[*] Step 1: Unbinding UDC..."
echo "" > $GADGET_DIR/UDC 2>/dev/null
sleep 0.5

echo "[*] Step 2: Configuring GUD (OpenMoko) VID/PID & Composite Descriptors..."
echo 0x1d50 > $GADGET_DIR/idVendor
echo 0x614d > $GADGET_DIR/idProduct
echo 0x0200 > $GADGET_DIR/bcdUSB

# Use Multi-Interface Association Descriptor (IAD) for composite GUD + NCM
echo 0xef > $GADGET_DIR/bDeviceClass
echo 0x02 > $GADGET_DIR/bDeviceSubClass
echo 0x01 > $GADGET_DIR/bDeviceProtocol

mkdir -p $GADGET_DIR/strings/0x409
echo "0123456789ABCDEF" > $GADGET_DIR/strings/0x409/serialnumber
echo "Ternary Systems" > $GADGET_DIR/strings/0x409/manufacturer
echo "V553 USB Display" > $GADGET_DIR/strings/0x409/product

mkdir -p $GADGET_DIR/configs/c.1/strings/0x409
echo "GUD Display + CDC-NCM Ethernet" > $GADGET_DIR/configs/c.1/strings/0x409/configuration
echo 500 > $GADGET_DIR/configs/c.1/MaxPower

echo "[*] Step 3: Setting up Functions..."
# Function 1: FunctionFS for GUD
mkdir -p $GADGET_DIR/functions/ffs.gud
ln -sf $GADGET_DIR/functions/ffs.gud $GADGET_DIR/configs/c.1/ 2>/dev/null

# Function 2: CDC-NCM Network
mkdir -p $GADGET_DIR/functions/ncm.usb0
echo "12:34:56:78:9a:bc" > $GADGET_DIR/functions/ncm.usb0/host_addr
echo "12:34:56:78:9a:bd" > $GADGET_DIR/functions/ncm.usb0/dev_addr
ln -sf $GADGET_DIR/functions/ncm.usb0 $GADGET_DIR/configs/c.1/ 2>/dev/null


echo "[*] Step 4: Mounting FunctionFS..."
mkdir -p /dev/ffs-gud
umount /dev/ffs-gud 2>/dev/null
mount -t functionfs gud /dev/ffs-gud
if [ $? -ne 0 ]; then
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
    echo "[-] UDC bind failed! Rolling back..."
    echo 0x18d1 > $GADGET_DIR/idVendor
    echo 0xd002 > $GADGET_DIR/idProduct
    rm -f $GADGET_DIR/configs/c.1/ffs.gud 2>/dev/null
    echo "$UDC_DEV" > $GADGET_DIR/UDC 2>/dev/null
    ifconfig usb0 192.168.100.1 netmask 255.255.255.0 up 2>/dev/null
    exit 1
fi

echo "[*] Step 7: Configuring Network..."
sleep 0.5
ifconfig usb0 192.168.100.1 netmask 255.255.255.0 up 2>/dev/null

echo "[+] GUD USB Display + CDC-NCM is now active and operational!"
echo "========================================================"
