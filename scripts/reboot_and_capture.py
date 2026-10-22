#!/usr/bin/env python3
import os
import sys
import termios
import select
import time

def main():
    port = '/dev/ttyUSB1'
    fd = os.open(port, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    attrs = termios.tcgetattr(fd)
    attrs[0] = 0
    attrs[1] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    attrs[3] = 0
    attrs[4] = termios.B115200
    attrs[5] = termios.B115200
    termios.tcsetattr(fd, termios.TCSANOW, attrs)

    # Flush
    while True:
        r, _, _ = select.select([fd], [], [], 0.05)
        if not r:
            break
        os.read(fd, 1024)

    print("Sending 'sync && reboot'...")
    os.write(fd, b"sync && reboot\n")

    start = time.time()
    boot_log = b""
    while time.time() - start < 15.0:
        r, _, _ = select.select([fd], [], [], 0.5)
        if r:
            try:
                chunk = os.read(fd, 4096)
                if chunk:
                    boot_log += chunk
                    sys.stdout.write(chunk.decode('utf-8', errors='replace'))
                    sys.stdout.flush()
            except BlockingIOError:
                time.sleep(0.01)

    os.close(fd)
    os.makedirs("logs", exist_ok=True)
    with open("logs/boot_panel_fixed.log", "wb") as f:
        f.write(boot_log)
    print("\n[Done] Boot log saved to logs/boot_panel_fixed.log")

if __name__ == '__main__':
    main()
