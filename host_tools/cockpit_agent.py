#!/usr/bin/env python3
# ============================================================================
# AGY Physical Cockpit Host Companion / 全志 V553 座舱宿主机端交互守护
# Listens to touch/gesture triggers over high-speed ADB and injects commands
# ============================================================================

import os
import sys
import time
import subprocess
import threading

def log(msg):
    timestamp = time.strftime("%H:%M:%S")
    print(f"[{timestamp}] [CockpitAgent] {msg}", flush=True)

def inject_terminal_input(text):
    """Inject keystrokes into current terminal via TIOCSTI or direct print"""
    log(f"⚡ INJECTING ACTION: '{text.strip()}' into terminal session...")
    try:
        import fcntl
        import termios
        for char in text:
            fcntl.ioctl(sys.stdin, termios.TIOCSTI, char.encode())
        return True
    except Exception as e:
        # Fallback: print to stdout
        sys.stdout.write(text)
        sys.stdout.flush()
        return False

def monitor_touch_events():
    """Stream events from /tmp/touch.log on the Allwinner V553 device"""
    log("Waiting for Allwinner V553 ADB connection...")
    subprocess.run(["adb", "wait-for-device"])
    log("Device connected via High-Speed ADB! Starting live event listener...")

    cmd = ["adb", "shell", "tail -n 0 -F /tmp/cockpit_event.log 2>/dev/null"]
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1)

    for line in proc.stdout:
        line = line.strip()
        if not line:
            continue
        if line == "ready" or "ready" in line:
            log("🎯 Physical Touch: [READY / PROCEED] pressed!")
            inject_terminal_input("ready\n")
        elif line == "ctrl+c" or "ctrl+c" in line:
            log("🛑 Physical Touch: [STOP / ABORT] pressed!")
            inject_terminal_input("\x03")
        else:
            log(f"⚡ Event: {line}")


def main():
    print("=" * 60)
    print("  AGY Physical Cockpit - Host Companion Agent")
    print("  Zero-Typing Hardware Interaction Bridge")
    print("=" * 60)
    while True:
        try:
            monitor_touch_events()
        except KeyboardInterrupt:
            log("Exiting Cockpit Agent...")
            break
        except Exception as e:
            log(f"Error: {e}, reconnecting in 2 seconds...")
            time.sleep(2)

if __name__ == "__main__":
    main()
