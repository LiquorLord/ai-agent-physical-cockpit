#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Allwinner Cold Boot Logger & U-Boot Interceptor
Allwinner cold-boot full log capture and U-Boot interceptor
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = 115200
LOG_FILE = "boot_full.log"
DURATION = 50

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

    print(f"[+] Serial port {PORT} ready at {BAUD} baud.")
    print(f"[+] Capturing log to: {LOG_FILE}")
    print("[+] >>> READY! PLEASE PLUG IN USB / POWER CABLE NOW! <<<")
    print("[+] >>> Ready! Please plug in the USB power cable now! <<<")
    sys.stdout.flush()

    start_time = time.time()
    first_byte_time = None
    last_tx_time = 0
    total_bytes = 0

    # Characters commonly used to interrupt U-Boot: Space, Enter, 's', Ctrl+C, '2'
    interrupt_chars = [b' ', b'\r', b's', b'\x03', b'2']
    char_idx = 0

    with open(LOG_FILE, "wb") as f:
        while time.time() - start_time < DURATION:
            rlist, _, _ = select.select([fd], [], [], 0.05)
            if rlist:
                data = os.read(fd, 4096)
                if data:
                    if first_byte_time is None:
                        first_byte_time = time.time()
                    f.write(data)
                    f.flush()
                    total_bytes += len(data)
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()

            # For the first 3.5 seconds after first byte is received, send interrupt chars
            now = time.time()
            if first_byte_time and (now - first_byte_time < 3.5):
                if now - last_tx_time >= 0.08:
                    c = interrupt_chars[char_idx % len(interrupt_chars)]
                    char_idx += 1
                    try:
                        os.write(fd, c)
                    except Exception:
                        pass
                    last_tx_time = now

    os.close(fd)
    print(f"\n[+] Finished! Captured {total_bytes} bytes in {LOG_FILE}.")

if __name__ == "__main__":
    main()
