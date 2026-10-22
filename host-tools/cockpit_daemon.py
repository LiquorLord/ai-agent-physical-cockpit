#!/usr/bin/env python3
"""
AGY Physical Cockpit - Host Global Keystroke Daemon
全志 V553 物理级座舱宿主机全局按键注入守护进程 (英汉双语 / Bilingual)

Receives hardware touch button events from V553 over USB CDC-NCM (UDP :8888)
and injects real keystrokes directly into the active terminal/editor window
via xdotool (zero-typing hardware cockpit).
"""

import os
import sys
import time
import socket
import subprocess

PORT = 8888
DEBOUNCE_SEC = 1.0
LAST_ACTION_TIMES = {}

def get_display():
    """Detect active X11 display"""
    return os.environ.get("DISPLAY", ":1")

def get_xauthority():
    """Detect XAUTHORITY file"""
    if "XAUTHORITY" in os.environ and os.path.exists(os.environ["XAUTHORITY"]):
        return os.environ["XAUTHORITY"]
    default_auth = os.path.expanduser("~/.Xauthority")
    if os.path.exists(default_auth):
        return default_auth
    return None

def log(msg):
    t = time.strftime("%H:%M:%S")
    print(f"[{t}] [CockpitDaemon] {msg}", flush=True)

def inject_keystroke(action):
    global LAST_ACTION_TIMES
    action = action.strip().lower()
    if not action:
        return

    now = time.time()
    # Normalize key for debounce
    key = "proceed" if ("proceed" in action or "ready" in action) else \
          "grill" if "grill" in action else \
          "plan" if "plan" in action else \
          "stop" if ("ctrl+c" in action or "stop" in action) else action

    if (now - LAST_ACTION_TIMES.get(key, 0)) < DEBOUNCE_SEC:
        log(f"⚠️ Debounced duplicate [{key}] touch event ({now - LAST_ACTION_TIMES.get(key, 0):.2f}s < {DEBOUNCE_SEC}s)")
        return
    LAST_ACTION_TIMES[key] = now

    env = os.environ.copy()
    if "DISPLAY" not in env or not env["DISPLAY"]:
        env["DISPLAY"] = get_display()
    auth = get_xauthority()
    if auth:
        env["XAUTHORITY"] = auth

    # Smart window selection: prioritize Alacritty / terminal windows
    target_win = None
    win_name = "Unknown"

    try:
        active_win = subprocess.check_output(["/usr/bin/xdotool", "getactivewindow"], env=env, stderr=subprocess.DEVNULL).decode().strip()
        active_name = subprocess.check_output(["/usr/bin/xdotool", "getwindowname", active_win], env=env, stderr=subprocess.DEVNULL).decode().strip()
    except Exception:
        active_win = None
        active_name = ""

    is_terminal = any(term in active_name.lower() for term in ["alacritty", "terminal", "kitty", "bash", "zsh", "tmux", "wezterm", "st", "urxvt", "xterm", "code"])

    if is_terminal and active_win:
        target_win = active_win
        win_name = active_name
    else:
        # Search for visible Alacritty / terminal windows
        try:
            candidates = subprocess.check_output(["/usr/bin/xdotool", "search", "--onlyvisible", "--class", "Alacritty"], env=env, stderr=subprocess.DEVNULL).decode().split()
            if candidates:
                target_win = candidates[-1]
                win_name = subprocess.check_output(["/usr/bin/xdotool", "getwindowname", target_win], env=env, stderr=subprocess.DEVNULL).decode().strip()
        except Exception:
            pass

        if not target_win:
            target_win = active_win
            win_name = active_name if active_name else "Default"

    log(f"⚡ Hardware Touch: [{action.upper()}] -> Target Window: {win_name} (ID: {target_win})")

    win_arg = ["--window", target_win] if target_win else []

    try:
        if "proceed" in action or "ready" in action:
            log(f"🟢 [PROCEED] -> Typing 'proceed' + Enter")
            subprocess.run(["/usr/bin/xdotool", "type"] + win_arg + ["--delay", "12", "proceed"], env=env, check=True)
            subprocess.run(["/usr/bin/xdotool", "key"] + win_arg + ["Return"], env=env, check=True)
        elif "grill" in action:
            log(f"🔵 [GRILL-ME] -> Typing '/grill-me' + Enter")
            subprocess.run(["/usr/bin/xdotool", "type"] + win_arg + ["--delay", "12", "/grill-me"], env=env, check=True)
            subprocess.run(["/usr/bin/xdotool", "key"] + win_arg + ["Return"], env=env, check=True)
        elif "plan" in action:
            log(f"🟣 [PLAN] -> Typing '/plan' + Enter")
            subprocess.run(["/usr/bin/xdotool", "type"] + win_arg + ["--delay", "12", "/plan"], env=env, check=True)
            subprocess.run(["/usr/bin/xdotool", "key"] + win_arg + ["Return"], env=env, check=True)
        elif "ctrl+c" in action or "stop" in action:
            log(f"🔴 [STOP / ABORT] -> Sending Ctrl+C")
            subprocess.run(["/usr/bin/xdotool", "key"] + win_arg + ["ctrl+c"], env=env, check=True)
        else:
            log(f"ℹ️ Unhandled action: {action}")
    except Exception as e:
        log(f"[-] xdotool injection failed: {e}")

def main():
    log("=================================================================")
    log(" AGY Physical Cockpit - Host Keystroke Injection Daemon")
    log(f" Listening on 0.0.0.0:{PORT} (UDP) | DISPLAY={get_display()}")
    log("=================================================================")

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        sock.bind(("0.0.0.0", PORT))
    except Exception as e:
        log(f"[-] Fatal: Failed to bind UDP {PORT}: {e}")
        sys.exit(1)

    while True:
        try:
            data, addr = sock.recvfrom(256)
            msg = data.decode("utf-8", errors="ignore")
            inject_keystroke(msg)
        except KeyboardInterrupt:
            log("[*] Shutting down...")
            break
        except Exception as e:
            log(f"[-] Error: {e}")

if __name__ == "__main__":
    main()
