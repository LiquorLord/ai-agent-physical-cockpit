#!/usr/bin/env python3
import os
import sys
import termios
import select
import time

def run_cmd(cmd, timeout=5.0):
    port = '/dev/ttyUSB1' if os.path.exists('/dev/ttyUSB1') else '/dev/ttyUSB0'
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

    token = f"CMD_DONE_{int(time.time()*1000)}"
    # Send stty -echo to prevent serial echo interference, then run command
    full_cmd = f"{cmd}\necho {token}:$?\n".encode()
    os.write(fd, full_cmd)

    start = time.time()
    res = b""
    token_b = token.encode()
    while time.time() - start < timeout:
        r, _, _ = select.select([fd], [], [], 0.2)
        if r:
            try:
                chunk = os.read(fd, 4096)
                if chunk:
                    res += chunk
                    # Look for token at beginning of a line (after newline)
                    if (b"\n" + token_b + b":") in res and b"\n" in res.split(b"\n" + token_b + b":")[-1]:
                        break
            except BlockingIOError:
                time.sleep(0.01)
                continue
        else:
            if (b"\n" + token_b + b":") in res:
                break

    os.close(fd)
    out = res.decode('utf-8', errors='replace')
    lines = out.splitlines()
    filtered = []
    for line in lines:
        if token in line or (cmd in line and len(filtered) == 0):
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
