#!/usr/bin/env python3
"""
AI-Agent Physical Cockpit - Host Companion Agent
Allwinner V553 physical cockpit host-side interactive companion

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

def check_connection():
    """Verify connection to V553 dashcam (USB IP or ADB)"""
    res = subprocess.run(["ping", "-c", "1", "-W", "1", "192.168.100.1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if res.returncode == 0:
        log("[*] V553 High-Speed USB link active (192.168.100.1)")
        return True
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    lines = [line for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if lines:
        log(f"[*] Target device connected via ADB: {lines[0].split('\t')[0]}")
        return True
    log("[!] Waiting for V553 USB connection...")
    return False

def sync_terminal_window_size(master_fd):
    """Propagate terminal window size to pseudo-terminal"""
    try:
        ws = fcntl.ioctl(sys.stdin.fileno(), termios.TIOCGWINSZ, struct.pack("HHHH", 0, 0, 0, 0))
        fcntl.ioctl(master_fd, termios.TIOCSWINSZ, ws)
    except Exception:
        pass

LAST_ACTION_TIMES = {}
DEBOUNCE_SEC = 1.2

def handle_action(action, master_fd=None):
    global LAST_ACTION_TIMES
    action = action.strip().lower()
    if not action:
        return
    
    # Normalize key for debouncing
    key = "proceed" if ("proceed" in action or "ready" in action) else \
          "grill" if "grill" in action else \
          "plan" if "plan" in action else \
          "stop" if ("ctrl+c" in action or "stop" in action) else action

    now = time.time()
    last_time = LAST_ACTION_TIMES.get(key, 0)
    if (now - last_time) < DEBOUNCE_SEC:
        log(f"⚠️ Debounced duplicate [{key}] touch event ({now - last_time:.2f}s < {DEBOUNCE_SEC}s)")
        return
    LAST_ACTION_TIMES[key] = now

    if "proceed" in action or "ready" in action:
        log("🟢 Physical Touch: [PROCEED] pressed! -> Injecting 'proceed\\n'")
        if master_fd is not None:
            try:
                os.write(master_fd, b"proceed\n")
            except OSError:
                pass
    elif "grill" in action:
        log("🔵 Physical Touch: [GRILL-ME] pressed! -> Injecting '/grill-me\\n'")
        if master_fd is not None:
            try:
                os.write(master_fd, b"/grill-me\n")
            except OSError:
                pass
    elif "plan" in action:
        log("🟣 Physical Touch: [PLAN] pressed! -> Injecting '/plan\\n'")
        if master_fd is not None:
            try:
                os.write(master_fd, b"/plan\n")
            except OSError:
                pass
    elif "ctrl+c" in action or "stop" in action:
        log("🔴 Physical Touch: [STOP / ABORT] pressed! -> Injecting Ctrl+C (\\x03)")
        if master_fd is not None:
            try:
                os.write(master_fd, b"\x03")
            except OSError:
                pass
    else:
        log(f"⚡ Hardware Event: {action}")

def udp_listener_thread(master_fd=None, stop_event=None):
    """
    High-Speed UDP listener on 0.0.0.0:8888
    Receives instantaneous touch button events from V553 over USB CDC-NCM
    """
    import socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        sock.bind(("0.0.0.0", 8888))
    except Exception as e:
        log(f"[-] UDP bind failed: {e}")
        return
    sock.settimeout(0.5)
    log("[*] UDP Hardware Listener active on port 8888 (USB High-Speed)")

    while not stop_event.is_set():
        try:
            data, addr = sock.recvfrom(256)
            handle_action(data.decode("utf-8", errors="ignore"), master_fd)
        except socket.timeout:
            continue
        except Exception:
            if stop_event.is_set():
                break
    sock.close()

def run_pty_session(cmd):
    """
    Wrap target command in PTY with bidirectional I/O and touch injection
    Run interactive tasks in a pseudo-terminal, injecting physical-key zero-input directly into the standard input stream
    """
    log(f"[*] Launching interactive PTY session: {' '.join(cmd)}")
    log("[*] Touch [PROCEED / READY] on screen to auto-input 'ready'")
    log("[*] Touch [STOP / ABORT] on screen to send Ctrl+C")

    is_tty = sys.stdin.isatty()
    old_termios = None
    if is_tty:
        try:
            old_termios = termios.tcgetattr(sys.stdin.fileno())
        except Exception:
            is_tty = False

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
        # Ensure child process does not inherit parent's TMUX variable
        os.environ.pop("TMUX", None)
        try:
            os.execvp(cmd[0], cmd)
        except Exception as e:
            print(f"Failed to execute {cmd}: {e}")
            sys.exit(1)

    # Parent process
    os.close(slave_fd)

    if is_tty:
        sync_terminal_window_size(master_fd)
        def handle_sigwinch(signum, frame):
            sync_terminal_window_size(master_fd)
        signal.signal(signal.SIGWINCH, handle_sigwinch)
    else:
        try:
            ws = struct.pack("HHHH", 24, 80, 0, 0)
            fcntl.ioctl(master_fd, termios.TIOCSWINSZ, ws)
        except Exception:
            pass

    stop_event = threading.Event()
    t_udp = threading.Thread(target=udp_listener_thread, args=(master_fd, stop_event), daemon=True)
    t_udp.start()

    if is_tty:
        tty.setraw(sys.stdin.fileno())

    try:
        while True:
            r_fds = [master_fd]
            if is_tty:
                r_fds.append(sys.stdin.fileno())

            r, _, _ = select.select(r_fds, [], [])

            if is_tty and sys.stdin.fileno() in r:
                try:
                    data = os.read(sys.stdin.fileno(), 1024)
                    if not data:
                        break
                    os.write(master_fd, data)
                except OSError:
                    break

            if master_fd in r:
                try:
                    data = os.read(master_fd, 1024)
                    if not data:
                        break
                    os.write(sys.stdout.fileno(), data)
                    sys.stdout.flush()
                except OSError:
                    break
    finally:
        if is_tty and old_termios is not None:
            try:
                termios.tcsetattr(sys.stdin.fileno(), termios.TCSADRAIN, old_termios)
            except Exception:
                pass
        stop_event.set()
        try:
            os.close(master_fd)
        except Exception:
            pass
        _, status = os.waitpid(pid, 0)
        log(f"[*] Child session finished (exit code {os.waitstatus_to_exitcode(status)})")

def run_monitor_mode():
    """Passive monitoring mode"""
    print("=" * 60)
    print("  AI-Agent Physical Cockpit - Hardware Monitor Mode")
    print("  Listening to Goodix GT911 touch events over UDP...")
    print("=" * 60)
    stop_event = threading.Event()
    t1 = threading.Thread(target=udp_listener_thread, args=(None, stop_event), daemon=True)
    t1.start()
    try:
        while not stop_event.is_set():
            time.sleep(0.5)
    except KeyboardInterrupt:
        stop_event.set()
        log("Exiting Monitor Mode...")

def switch_device_profile(profile_name):
    """Send profile switch signal to touch_cockpit over UDP or ADB"""
    import socket
    log(f"[*] Switching Cockpit Profile to [{profile_name.upper()}]...")
    # Try direct UDP if IP is up
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.settimeout(0.5)
        s.sendto(f"profile {profile_name}\n".encode(), ("192.168.100.1", 8889))
        s.close()
    except Exception:
        pass
    # Also attempt via ADB if active
    try:
        subprocess.run(
            ["adb", "shell", f"echo 'profile {profile_name}' | nc -u -w1 127.0.0.1 8889 2>/dev/null || true"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=1.0
        )
    except Exception:
        pass

def main():
    parser = argparse.ArgumentParser(
        description="AI-Agent Physical Cockpit Host Companion",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        "--monitor", action="store_true",
        help="Run in passive monitor mode without launching an interactive PTY shell"
    )
    parser.add_argument(
        "-p", "--profile", type=str, choices=["agy", "claude", "dsh", "shell"],
        help="Switch active AI Agent profile (agy, claude, dsh, shell)"
    )
    parser.add_argument(
        "cmd", nargs="*", default=None,
        help="Command to run in interactive PTY (default: $SHELL or bash)"
    )
    args = parser.parse_args()

    if args.profile:
        switch_device_profile(args.profile)
        if not args.cmd and not args.monitor:
            print(f"[+] Active Cockpit profile set to: {args.profile.upper()}")
            sys.exit(0)

    check_connection()

    command = args.cmd if args.cmd else [os.environ.get("SHELL", "/bin/bash")]
    cmd_base = os.path.basename(command[0]).lower()
    if not args.profile:
        if "claude" in cmd_base:
            switch_device_profile("claude")
        elif "dsh" in cmd_base or "deepseek" in cmd_base:
            switch_device_profile("dsh")
        elif "agy" in cmd_base:
            switch_device_profile("agy")

    if args.monitor or not sys.stdin.isatty():
        run_monitor_mode()
    else:
        run_pty_session(command)

if __name__ == "__main__":
    main()
