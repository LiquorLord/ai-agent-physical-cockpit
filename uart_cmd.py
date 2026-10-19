#!/usr/bin/env python3
import os
import sys
import termios
import select
import time

def run_cmd(cmd, timeout=5.0):
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

    # Flush input buffer
    while True:
        r, _, _ = select.select([fd], [], [], 0.05)
        if not r:
            break
        os.read(fd, 1024)

    full_cmd = f"{cmd}; echo '---CMD_DONE:$?---'\n".encode()
    os.write(fd, full_cmd)

    start = time.time()
    res = b""
    while time.time() - start < timeout:
        r, _, _ = select.select([fd], [], [], 0.2)
        if r:
            chunk = os.read(fd, 4096)
            if chunk:
                res += chunk
                if b"---CMD_DONE:" in res and b"---\r\n" in res:
                    break
        else:
            if b"---CMD_DONE:" in res:
                break

    os.close(fd)
    out = res.decode('utf-8', errors='replace')
    # Filter echo and marker
    lines = out.splitlines()
    filtered = []
    marker_seen = False
    for line in lines:
        if "---CMD_DONE:" in line:
            marker_seen = True
            continue
        filtered.append(line)
    return "\n".join(filtered)

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: uart_cmd.py <command>")
        sys.exit(1)
    cmd = " ".join(sys.argv[1:])
    out = run_cmd(cmd, timeout=10.0)
    print(out)
