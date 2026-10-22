#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Test common root passwords on Linux console
Test common console passwords
"""

import sys
import os
import termios
import select
import time

PORT = "/dev/ttyUSB0"
PASSWORDS = [
    "",             # empty password
    "root",
    "admin",
    "tina",
    "123456",
    "v553",
    "f900x",
    "jdh",
    "km2026",
    "sochip",
    "allwinner",
    "sunxi",
    "1234",
    "888888",
    "password"
]

def send_and_wait(fd, s, wait_sec=0.5):
    os.write(fd, s.encode('latin1') + b'\n')
    time.sleep(wait_sec)
    buf = b''
    while True:
        r, _, _ = select.select([fd], [], [], 0.1)
        if r:
            data = os.read(fd, 1024)
            if data:
                buf += data
            else:
                break
        else:
            break
    return buf.decode('latin1', errors='replace')

def main():
    fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    attrs = termios.tcgetattr(fd)
    attrs[4] = attrs[5] = termios.B115200
    attrs[0] = attrs[1] = attrs[3] = 0
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    termios.tcsetattr(fd, termios.TCSANOW, attrs)
    termios.tcflush(fd, termios.TCIOFLUSH)

    print("[+] Waking up console...")
    res = send_and_wait(fd, "", 0.3)
    print("Response:", repr(res))

    for pwd in PASSWORDS:
        print(f"\n[*] Trying username 'root' with password: {pwd!r}")
        # Send username
        send_and_wait(fd, "\x03\r", 0.2)
        out = send_and_wait(fd, "root", 0.5)
        print("  Login prompt out:", repr(out[-100:] if len(out) > 100 else out))
        out2 = send_and_wait(fd, pwd, 0.8)
        print("  After password out:", repr(out2[-150:] if len(out2) > 150 else out2))

        if "#" in out2 or "root@" in out2 or "$" in out2:
            print(f"\n[!!!] SUCCESS! Root password is: {pwd!r}")
            # Try a command
            cmd_out = send_and_wait(fd, "id; uname -a", 1.0)
            print("Command output:", cmd_out)
            os.close(fd)
            return

    os.close(fd)
    print("\n[-] Finished dictionary test without immediate hit.")

if __name__ == "__main__":
    main()
