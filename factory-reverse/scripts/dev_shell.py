#!/usr/bin/env python3
"""
Target Device Serial Shell Controller
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
BAUD = termios.B115200

def run_cmd(cmd, timeout=3.0):
    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    try:
        attrs = termios.tcgetattr(fd)
        attrs[0] = 0
        attrs[1] = 0
        attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
        attrs[3] = 0
        attrs[4] = BAUD
        attrs[5] = BAUD
        attrs[6][termios.VMIN] = 0
        attrs[6][termios.VTIME] = 1
        termios.tcsetattr(fd, termios.TCSANOW, attrs)

        # Flush input buffer
        termios.tcflush(fd, termios.TCIFLUSH)

        # Send command with newline
        os.write(fd, (cmd + "\n").encode())

        output = b""
        start_time = time.time()
        while time.time() - start_time < timeout:
            r, _, _ = select.select([fd], [], [], 0.3)
            if r:
                chunk = os.read(fd, 4096)
                if not chunk:
                    break
                output += chunk
                # Check if we returned to prompt
                if b"root@(none):" in output and output.endswith(b"# "):
                    break
            else:
                if b"root@(none):" in output and output.endswith(b"# "):
                    break

        text = output.decode(errors="replace")
        return text
    finally:
        os.close(fd)

if __name__ == "__main__":
    if len(sys.argv) > 1:
        cmd = " ".join(sys.argv[1:])
    else:
        cmd = "uname -a"
    result = run_cmd(cmd)
    print(result)
