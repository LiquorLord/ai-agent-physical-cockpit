#!/usr/bin/env python3
"""
AGY Physical Cockpit - Unified Executive CLI
全志 V553 物理级专属座舱一键式统一入口 (英汉双语 / Bilingual)

Integrates:
1. Low-latency Dual-Zone display streaming (1200x600 viewport)
2. Interactive Zero-Typing PTY Master shell session
3. Live capacitive touch hardware event injection
4. Automatic graceful teardown of background streaming
"""

import os
import sys
import time
import signal
import subprocess
import argparse

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
STREAMER_SCRIPT = os.path.join(SCRIPT_DIR, "stream_to_monitor.py")
AGENT_SCRIPT = os.path.join(SCRIPT_DIR, "cockpit_agent.py")

def log(msg):
    timestamp = time.strftime("%H:%M:%S")
    print(f"\033[1;36m[{timestamp}] [Cockpit]\033[0m {msg}", flush=True)

def check_device():
    """Verify device connectivity over ADB"""
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    devices = [line.split()[0] for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if not devices:
        log("Waiting for Allwinner V553 device over USB...")
        subprocess.run(["adb", "wait-for-device"])
        log("Device connected!")
    # Setup port forwarding
    subprocess.run(["adb", "forward", "tcp:5900", "tcp:5900"], check=True)
    subprocess.run(["adb", "forward", "tcp:5901", "tcp:5901"], check=True)
    return True

def get_window_geometry(win_name):
    """Find X11 window offset and size via xwininfo"""
    try:
        out = subprocess.check_output(["xwininfo", "-name", win_name], text=True)
        x, y, w, h = 0, 0, 1200, 600
        for line in out.splitlines():
            line = line.strip()
            if line.startswith("Absolute upper-left X:"):
                x = int(line.split(":")[1].strip())
            elif line.startswith("Absolute upper-left Y:"):
                y = int(line.split(":")[1].strip())
            elif line.startswith("Width:"):
                w = int(line.split(":")[1].strip())
            elif line.startswith("Height:"):
                h = int(line.split(":")[1].strip())
        return x, y, w, h
    except Exception as e:
        log(f"Window search error: {e}")
        return None

def start_stream_subprocess(mode="dual", stream_type="test", window=None, display=None, fps=25, x=0, y=0):
    """Launch stream_to_monitor.py as a background subprocess"""
    cmd = [sys.executable, STREAMER_SCRIPT, "--mode", mode, "--fps", str(fps)]

    if stream_type == "test":
        cmd.append("--test")
    elif stream_type == "window" and window:
        geom = get_window_geometry(window)
        if geom:
            wx, wy, _, _ = geom
            cmd.extend(["--x", str(wx), "--y", str(wy)])
            log(f"[*] Locked on window '{window}' at coordinates ({wx}, {wy})")
        else:
            log(f"[!] Window '{window}' not found, falling back to desktop origin.")
    elif display:
        cmd.extend(["--display", display, "--x", str(x), "--y", str(y)])

    proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    return proc

def main():
    parser = argparse.ArgumentParser(
        description="AGY Physical Cockpit - Unified Executive Companion",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""Examples:
  ./cockpit                     # Run default shell with live test benchmark pattern
  ./cockpit --display :0.0      # Stream host desktop region to 1200x600 viewport
  ./cockpit --window Terminal   # Lock stream onto a specific X11 window
  ./cockpit -- agy              # Run 'agy' AI assistant inside the zero-typing cockpit
  ./cockpit -- btop             # Stream performance metrics while monitoring touch
"""
    )
    parser.add_argument("--test", action="store_true", default=True, help="Stream dynamic benchmark test pattern (default)")
    parser.add_argument("--display", help="Stream X11 display region (e.g. :0.0)")
    parser.add_argument("--window", help="Stream a specific X11 window by name")
    parser.add_argument("--fps", type=int, default=25, help="Streaming framerate (default: 25)")
    parser.add_argument("--mode", choices=["dual", "full"], default="dual", help="Display mode (default: dual)")
    parser.add_argument("--no-stream", action="store_true", help="Do not start display stream (PTY only)")
    parser.add_argument("--popup", action="store_true", help="Spawn interactive cockpit terminal in desktop GUI window")
    parser.add_argument("cmd", nargs="*", default=None, help="Target command to run in PTY (default: $SHELL)")
    args = parser.parse_args()

    if args.popup:
        display = os.environ.get("DISPLAY", ":0")
        env = os.environ.copy()
        env["DISPLAY"] = display
        # Build command without --popup
        cmd = [sys.executable, os.path.realpath(__file__)]
        if args.display:
            cmd.extend(["--display", args.display])
        if args.window:
            cmd.extend(["--window", args.window])
        if args.no_stream:
            cmd.append("--no-stream")
        if args.cmd:
            cmd.append("--")
            cmd.extend(args.cmd)
        
        cwd = os.path.dirname(os.path.dirname(os.path.realpath(__file__)))
        term = "gnome-terminal"
        term_cmd = [
            term,
            f"--working-directory={cwd}",
            "--title=AGY Physical Cockpit",
            "--geometry=100x30",
            "--",
            *cmd
        ]
        log(f"Launching desktop popup cockpit terminal on {display} via {term}...")
        subprocess.Popen(term_cmd, env=env, cwd=cwd)
        return

    print("=" * 65)
    print("  \033[1;32mAGY PHYSICAL COCKPIT\033[0m — Zero-Typing Developer Console")
    print("  Allwinner V553 (1600x600 BOE Display • GT911 Capacitive Touch)")
    print("=" * 65)

    check_device()

    stream_proc = None
    if not args.no_stream:
        stream_type = "test"
        if args.window:
            stream_type = "window"
        elif args.display:
            stream_type = "display"

        log(f"Starting background display stream ({args.mode} mode @ {args.fps}fps)...")
        stream_proc = start_stream_subprocess(
            mode=args.mode,
            stream_type=stream_type,
            window=args.window,
            display=args.display,
            fps=args.fps
        )
        time.sleep(0.5)

    try:
        # Import and run cockpit_agent PTY session
        sys.path.insert(0, SCRIPT_DIR)
        import cockpit_agent
        target_cmd = args.cmd if args.cmd else [os.environ.get("SHELL", "/bin/bash")]
        cockpit_agent.run_pty_session(target_cmd)
    finally:
        if stream_proc:
            log("Shutting down background display streamer...")
            stream_proc.terminate()
            try:
                stream_proc.wait(timeout=2)
            except subprocess.TimeoutExpired:
                stream_proc.kill()
            log("Cockpit session closed cleanly.")

if __name__ == "__main__":
    main()
