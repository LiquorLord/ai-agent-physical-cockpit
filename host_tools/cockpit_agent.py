#!/usr/bin/env python3
"""
AGY Physical Cockpit - Host Companion Agent
全志 V553 物理座舱宿主机端交互伴侣 (英汉双语 / Bilingual)

Zero-Typing PTY Master & ADB Hardware Event Bridge:
1. PTY Master Mode (Default):
   Forks an interactive shell ($SHELL) or target CLI (e.g., 'agy', 'bash', 'python3')
   in a pseudo-terminal. When the developer touches the physical screen on the dashcam:
   - Emerald Green [PROCEED / READY] -> Injects 'ready\n' directly into child stdin.
   - Crimson Red   [STOP / ABORT]    -> Injects Ctrl+C (b'\x03') directly into child stdin.
   Bypasses Linux 6.x kernel TIOCSTI restrictions with 100% native POSIX reliability.

2. Monitor Mode (--monitor):
   Passively listens to physical touch / hardware events and logs them in real time.
"""

import os
import sys
import pty
import tty
import termios
import select
import fcntl
import struct
import signal
import subprocess
import threading
import time
import argparse

def log(msg):
    timestamp = time.strftime("%H:%M:%S")
    print(f"\r\033[K[{timestamp}] [CockpitAgent] {msg}", flush=True)

def check_adb():
    """Verify ADB connection to the dashcam device / 验证设备 ADB 连接"""
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    lines = [line for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if not lines:
        log("[!] No ADB device detected. Waiting for device...")
        return False
    serial = lines[0].split("\t")[0]
    log(f"[*] Target device connected: {serial}")
    return True

def sync_terminal_window_size(master_fd):
    """Propagate terminal window size to pseudo-terminal / 同步终端窗口行列尺寸"""
    try:
        ws = fcntl.ioctl(sys.stdin.fileno(), termios.TIOCGWINSZ, struct.pack("HHHH", 0, 0, 0, 0))
        fcntl.ioctl(master_fd, termios.TIOCSWINSZ, ws)
    except Exception:
        pass

def event_listener_thread(master_fd=None, stop_event=None):
    """
    Background thread that tails /tmp/cockpit_event.log over High-Speed ADB
    后台线程：实时追踪 Allwinner V553 触摸屏事件并无锁注入 PTY
    """
    while not stop_event.is_set():
        try:
            # Wait for device if disconnected
            subprocess.run(["adb", "wait-for-device"])
            cmd = ["adb", "shell", "tail -n 0 -F /tmp/cockpit_event.log 2>/dev/null"]
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, bufsize=1)

            while not stop_event.is_set():
                line = proc.stdout.readline()
                if not line:
                    break
                action = line.strip().lower()
                if not action:
                    continue

                if "ready" in action:
                    log("🎯 Physical Touch: [READY / PROCEED] pressed! -> Injecting 'ready\\n'")
                    if master_fd is not None:
                        try:
                            os.write(master_fd, b"ready\n")
                        except OSError:
                            break
                elif "ctrl+c" in action or "stop" in action:
                    log("🛑 Physical Touch: [STOP / ABORT] pressed! -> Injecting Ctrl+C")
                    if master_fd is not None:
                        try:
                            os.write(master_fd, b"\x03")
                        except OSError:
                            break
                else:
                    log(f"⚡ Hardware Event: {action}")

            proc.terminate()
            proc.wait()
        except Exception as e:
            if stop_event.is_set():
                break
            log(f"ADB listener error: {e}, reconnecting in 2s...")
            time.sleep(2)

def run_pty_session(cmd):
    """
    Wrap target command in PTY with bidirectional I/O and touch injection
    在伪终端中执行交互式任务，物理按键零输入直接注入标准输入流
    """
    log(f"[*] Launching interactive PTY session: {' '.join(cmd)}")
    log("[*] Touch [PROCEED / READY] on screen to auto-input 'ready'")
    log("[*] Touch [STOP / ABORT] on screen to send Ctrl+C")

    # Save original terminal settings
    old_termios = termios.tcgetattr(sys.stdin.fileno())

    master_fd, slave_fd = pty.openpty()

    # Fork process
    pid = os.fork()

    if pid == 0:
        # Child process
        os.close(master_fd)
        os.setsid()
        fcntl.ioctl(slave_fd, termios.TIOCSCTTY, 0)
        os.dup2(slave_fd, 0)
        os.dup2(slave_fd, 1)
        os.dup2(slave_fd, 2)
        if slave_fd > 2:
            os.close(slave_fd)
        try:
            os.execvp(cmd[0], cmd)
        except Exception as e:
            print(f"Failed to execute {cmd}: {e}")
            sys.exit(1)

    # Parent process
    os.close(slave_fd)
    sync_terminal_window_size(master_fd)

    def handle_sigwinch(signum, frame):
        sync_terminal_window_size(master_fd)

    signal.signal(signal.SIGWINCH, handle_sigwinch)

    stop_event = threading.Event()
    listener = threading.Thread(target=event_listener_thread, args=(master_fd, stop_event), daemon=True)
    listener.start()

    # Put host terminal into raw mode
    tty.setraw(sys.stdin.fileno())

    try:
        while True:
            r, _, _ = select.select([sys.stdin.fileno(), master_fd], [], [])
            if sys.stdin.fileno() in r:
                data = os.read(sys.stdin.fileno(), 1024)
                if not data:
                    break
                os.write(master_fd, data)

            if master_fd in r:
                try:
                    data = os.read(master_fd, 1024)
                    if not data:
                        break
                    os.write(sys.stdout.fileno(), data)
                except OSError:
                    break
    finally:
        # Restore host terminal settings
        termios.tcsetattr(sys.stdin.fileno(), termios.TCSADRAIN, old_termios)
        stop_event.set()
        os.close(master_fd)
        _, status = os.waitpid(pid, 0)
        log(f"[*] Child session finished (exit code {os.waitstatus_to_exitcode(status)})")

def run_monitor_mode():
    """Passive monitoring mode / 被动监听模式"""
    print("=" * 60)
    print("  AGY Physical Cockpit - Hardware Monitor Mode")
    print("  Listening to Goodix GT911 touch events over ADB...")
    print("=" * 60)
    stop_event = threading.Event()
    try:
        event_listener_thread(master_fd=None, stop_event=stop_event)
    except KeyboardInterrupt:
        stop_event.set()
        log("Exiting Monitor Mode...")

def main():
    parser = argparse.ArgumentParser(
        description="AGY Physical Cockpit Host Companion / V553 物理座舱交互伴侣",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        "--monitor", action="store_true",
        help="Run in passive monitor mode without launching an interactive PTY shell"
    )
    parser.add_argument(
        "cmd", nargs="*", default=None,
        help="Command to run in interactive PTY (default: $SHELL or bash)"
    )
    args = parser.parse_args()

    check_adb()

    if args.monitor or not sys.stdin.isatty():
        run_monitor_mode()
    else:
        command = args.cmd if args.cmd else [os.environ.get("SHELL", "/bin/bash")]
        run_pty_session(command)

if __name__ == "__main__":
    main()
