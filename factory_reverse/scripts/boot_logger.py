#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Serial Boot Logger for Allwinner V553
全志 V553 串口引导抓取工具
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = 115200
LOG_FILE = "boot_capture.log"
DURATION = 60 # Run for 60 seconds

def main():
    if not os.path.exists(PORT):
        print(f"[-] Error: {PORT} does not exist.")
        sys.exit(1)

    print(f"[+] Opening {PORT} at {BAUD} baud...")
    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)

    # Configure termios
    attrs = termios.tcgetattr(fd)
    attrs[4] = termios.B115200
    attrs[5] = termios.B115200
    attrs[0] = 0
    attrs[1] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    attrs[3] = 0
    termios.tcsetattr(fd, termios.TCSANOW, attrs)

    # Flush existing data
    termios.tcflush(fd, termios.TCIOFLUSH)

    print(f"[+] Listening and capturing to {LOG_FILE} for {DURATION} seconds...")
    print("[+] READY! You can press RESET now! / 已就绪，请现在按下 RESET！")
    sys.stdout.flush()

    start_time = time.time()
    total_bytes = 0

    with open(LOG_FILE, "wb") as f:
        while time.time() - start_time < DURATION:
            rlist, _, _ = select.select([fd], [], [], 0.5)
            if rlist:
                data = os.read(fd, 4096)
                if data:
                    f.write(data)
                    f.flush()
                    total_bytes += len(data)
                    sys.stdout.buffer.write(data)
                    sys.stdout.buffer.flush()

    os.close(fd)
    print(f"\n[+] Capture complete! Captured {total_bytes} bytes to {LOG_FILE}.")

if __name__ == "__main__":
    main()
