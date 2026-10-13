#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Allwinner UART0 Monitor & Shell Tool
全志 UART0 串口监控与控制台工具
Standard library only / 仅依赖 Python 标准库 (termios, fcntl, select)
"""

import sys
import os
import tty
import termios
import select
import time
from datetime import datetime

DEFAULT_BAUD = 115200
DEFAULT_PORT = "/dev/ttyUSB0"

def get_baud_const(baud):
    baud_dict = {
        9600: termios.B9600,
        19200: termios.B19200,
        38400: termios.B38400,
        57600: termios.B57600,
        115200: termios.B115200,
        921600: getattr(termios, 'B921600', termios.B115200),
        1500000: getattr(termios, 'B1500000', termios.B115200)
    }
    return baud_dict.get(baud, termios.B115200)

def main():
    port = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_PORT
    baud = int(sys.argv[2]) if len(sys.argv) > 2 else DEFAULT_BAUD

    if not os.path.exists(port):
        print(f"[-] Serial port {port} not found! / 未找到串口设备 {port}")
        sys.exit(1)

    print(f"[+] Opening {port} at {baud} baud... / 正在打开串口 {port}，波特率 {baud}...")

    # Open serial port
    fd = os.open(port, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    
    # Configure serial attributes
    attrs = termios.tcgetattr(fd)
    baud_const = get_baud_const(baud)
    attrs[4] = baud_const # ispeed
    attrs[5] = baud_const # ospeed
    
    # Raw mode: 8N1, no flow control
    attrs[0] = 0 # iflag
    attrs[1] = 0 # oflag
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL # cflag
    attrs[3] = 0 # lflag
    termios.tcsetattr(fd, termios.TCSANOW, attrs)

    log_filename = f"uart_log_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
    print(f"[+] Logging serial output to: {log_filename}")
    print("[+] Press Ctrl+C to exit. / 按 Ctrl+C 退出。")
    print("=" * 60)

    with open(log_filename, "ab") as log_file:
        try:
            while True:
                rlist, _, _ = select.select([fd, sys.stdin.fileno()], [], [], 0.1)
                for ready_fd in rlist:
                    if ready_fd == fd:
                        data = os.read(fd, 4096)
                        if data:
                            sys.stdout.buffer.write(data)
                            sys.stdout.buffer.flush()
                            log_file.write(data)
                            log_file.flush()
                    elif ready_fd == sys.stdin.fileno():
                        user_input = sys.stdin.buffer.read(1)
                        if user_input:
                            os.write(fd, user_input)
        except KeyboardInterrupt:
            print("\n[+] Exiting UART monitor. / 退出串口监控。")
        finally:
            os.close(fd)

if __name__ == "__main__":
    main()
