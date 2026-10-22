#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Cold Boot Capture Tool for Allwinner V553
Allwinner V553 cold-boot log capture tool
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = 115200
LOG_FILE = "boot_full.log"
DURATION = 45

def main():
    if not os.path.exists(PORT):
        print(f"[-] Error: {PORT} not found.")
        sys.exit(1)

    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    attrs = termios.tcgetattr(fd)
    attrs[4] = attrs[5] = termios.B115200
    attrs[0] = attrs[1] = attrs[3] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    termios.tcsetattr(fd, termios.TCSANOW, attrs)
    termios.tcflush(fd, termios.TCIOFLUSH)

    print(f"[+] Listening on {PORT} at {BAUD} baud for {DURATION}s...")
    print("[+] >>> PLEASE UNPLUG AND RE-PLUG THE TYPE-C POWER CABLE NOW! <<<")
    print("[+] >>> Please unplug the Type-C power cable now, then plug it back in! <<<")
    sys.stdout.flush()

    start_time = time.time()
    total_bytes = 0

    with open(LOG_FILE, "wb") as f:
        while time.time() - start_time < DURATION:
            rlist, _, _ = select.select([fd], [], [], 0.3)
            if rlist:
                data = os.read(fd, 4096)
                if data:
                    f.write(data)
                    f.flush()
                    total_bytes += len(data)
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()

    os.close(fd)
    print(f"\n[+] Cold boot capture complete! Saved {total_bytes} bytes to {LOG_FILE}.")

if __name__ == "__main__":
    main()
