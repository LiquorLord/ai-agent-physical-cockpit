#!/usr/bin/env python3
"""
Real-time Serial Port Logger & Debug Capture for Allwinner V553 Cockpit
Captures UART0 @ 115200 8N1 directly into logs/boot_capture.log and stdout.
"""

import os
import sys
import time
import termios

PORT = "/dev/ttyUSB1" if os.path.exists("/dev/ttyUSB1") else "/dev/ttyUSB0"
if len(sys.argv) > 1:
    PORT = sys.argv[1]

BAUD = termios.B115200
LOG_FILE = "logs/boot_capture.log"

def main():
    os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)
    print(f"[*] Monitoring serial port: {PORT}")
    while not os.path.exists(PORT):
        time.sleep(0.5)

    try:
        fd = os.open(PORT, os.O_RDWR | os.O_NOCTTY | os.O_NONBLOCK)
    except Exception as e:
        print(f"[-] Error opening {PORT}: {e}")
        sys.exit(1)

    attrs = termios.tcgetattr(fd)
    # iflag
    attrs[0] = termios.IGNPAR
    # oflag
    attrs[1] = 0
    # cflag: CS8, CREAD, CLOCAL
    attrs[2] = termios.CS8 | termios.CREAD | termios.CLOCAL
    # lflag: raw
    attrs[3] = 0
    # ispeed & ospeed
    attrs[4] = BAUD
    attrs[5] = BAUD
    # VMIN = 1, VTIME = 0
    attrs[6][termios.VMIN] = 1
    attrs[6][termios.VTIME] = 0

    termios.tcsetattr(fd, termios.TCSANOW, attrs)
    termios.tcflush(fd, termios.TCIOFLUSH)

    print(f"[*] Serial monitor active! Streaming live to {LOG_FILE} and console...")
    print(f"[*] >>> WAITING FOR V553 POWER-CYCLE / BOOT LOGS <<<")
    sys.stdout.flush()

    with open(LOG_FILE, "a", buffering=1, encoding="utf-8", errors="replace") as log_f:
        header = f"\n=== BOOT CAPTURE SESSION STARTED AT {time.strftime('%Y-%m-%d %H:%M:%S')} ===\n"
        log_f.write(header)
        log_f.flush()
        
        while True:
            try:
                data = os.read(fd, 1024)
                if data:
                    text = data.decode("utf-8", errors="replace")
                    sys.stdout.write(text)
                    sys.stdout.flush()
                    log_f.write(text)
                    log_f.flush()
                else:
                    time.sleep(0.01)
            except BlockingIOError:
                time.sleep(0.01)
            except KeyboardInterrupt:
                print("\n[*] Monitor stopped by user.")
                break
            except Exception as e:
                print(f"\n[-] Read error: {e}")
                time.sleep(0.5)

if __name__ == "__main__":
    main()
