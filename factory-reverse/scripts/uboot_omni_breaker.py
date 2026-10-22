#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Omni-Key U-Boot Breaker & Analyzer
Omni-key multi-button U-Boot interceptor and boot analyzer
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = 115200

# Try all known Allwinner Tina U-Boot interrupt keys
KEYS = [b" ", b"2", b"\r", b"s", b"\x03", b"\x1b", b"1", b"d"]

def main():
    if not os.path.exists(PORT):
        print(f"[-] Error: {PORT} not found.")
        sys.exit(1)

    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    attrs = termios.tcgetattr(fd)
    attrs[4] = attrs[5] = termios.B115200
    attrs[0] = 0
    attrs[1] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    attrs[3] = 0
    termios.tcsetattr(fd, termios.TCSANOW, attrs)
    termios.tcflush(fd, termios.TCIOFLUSH)

    print("=" * 65)
    print(" [*] Omni-Key U-Boot Breaker Running")
    print(" [*] Sending alternating keys: Space, '2', Enter, 's', Ctrl+C, Esc...")
    print(" [*] >>> PLEASE UNPLUG TYPE-C, WAIT 3 SECONDS, AND RE-PLUG NOW! <<<")
    print(" [*] >>> Please unplug Type-C now, wait 3 seconds, then plug it back in! <<<")
    print("=" * 65)
    sys.stdout.flush()

    log_f = open("uboot_omni.log", "wb")
    start_time = time.time()
    interrupted = False
    rx_buf = b""
    idx = 0

    while time.time() - start_time < 60:
        rlist, _, _ = select.select([fd], [], [], 0.008)
        if rlist:
            data = os.read(fd, 2048)
            if data:
                log_f.write(data)
                log_f.flush()
                rx_buf += data
                sys.stdout.buffer.write(data)
                sys.stdout.buffer.flush()

                if b"=>" in rx_buf[-20:] or b"sunxi#" in rx_buf[-20:] or b"uboot#" in rx_buf[-20:]:
                    interrupted = True
                    break

        if not interrupted:
            try:
                os.write(fd, KEYS[idx % len(KEYS)])
                idx += 1
            except Exception:
                pass

    if interrupted:
        print("\n\n" + "=" * 65)
        print(" [!!!] U-BOOT INTERRUPTED!")
        print("=" * 65)
        # Test command
        time.sleep(0.1)
        os.write(fd, b"\r\nprintenv bootargs bootcmd\r\n")
        time.sleep(0.5)
        res = b""
        while True:
            r, _, _ = select.select([fd], [], [], 0.2)
            if r:
                c = os.read(fd, 1024)
                if c: res += c
                else: break
            else: break
        print(res.decode('latin1', errors='replace'))
    else:
        print(f"\n[-] Attempt finished. Total bytes logged: {len(rx_buf)}")

    log_f.close()
    os.close(fd)

if __name__ == "__main__":
    main()
