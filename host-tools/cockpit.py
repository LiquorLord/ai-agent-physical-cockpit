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
import shutil
import subprocess
import argparse

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
STREAMER_SCRIPT = os.path.join(SCRIPT_DIR, "host_tools", "stream_to_monitor.py") if os.path.exists(os.path.join(SCRIPT_DIR, "host_tools", "stream_to_monitor.py")) else os.path.join(SCRIPT_DIR, "stream_to_monitor.py")
AGENT_SCRIPT = os.path.join(SCRIPT_DIR, "host_tools", "cockpit_agent.py") if os.path.exists(os.path.join(SCRIPT_DIR, "host_tools", "cockpit_agent.py")) else os.path.join(SCRIPT_DIR, "cockpit_agent.py")

def log(msg):
    timestamp = time.strftime("%H:%M:%S")
    print(f"\033[1;36m[{timestamp}] [Cockpit]\033[0m {msg}", flush=True)

def check_device():
    """Verify device connectivity (USB CDC-NCM IP or ADB)"""
    if subprocess.run(["ping", "-c", "1", "-W", "1", "192.168.100.1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
        log("[*] V553 High-Speed USB link active (192.168.100.1)")
        return True
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    devices = [line.split()[0] for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if not devices:
        log("Waiting for Allwinner V553 device over USB...")
        subprocess.run(["adb", "wait-for-device"])
        log("Device connected via ADB!")
    subprocess.run(["adb", "forward", "tcp:5900", "tcp:5900"], check=False)
    subprocess.run(["adb", "forward", "tcp:5901", "tcp:5901"], check=False)
    return True

def is_gud_active():
    """Check if V553 GUD hardware display is detected and active"""
    import glob
    for path in glob.glob("/sys/class/drm/card*-USB-1/status"):
        try:
            with open(path) as f:
                if f.read().strip() == "connected":
                    return True
        except Exception:
            pass
    return False

def get_window_geometry(win_name):
    """Find X11 window offset and size via xwininfo"""
    try:
        out = subprocess.check_output(["xwininfo", "-name", win_name], env=os.environ, text=True)
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

def start_stream_subprocess(mode="dual", stream_type="test", window=None, display=None, fps=10, x=0, y=0):
    """Launch stream_to_monitor.py as a background subprocess"""
    current_display = display or os.environ.get("DISPLAY", ":1")
    cmd = [sys.executable, STREAMER_SCRIPT, "--mode", mode, "--fps", str(fps), "--display", current_display]

    if stream_type == "test":
        cmd.append("--test")
    elif stream_type == "window" and window:
        geom = get_window_geometry(window)
        if geom:
            wx, wy, _, _ = geom
            cmd.extend(["--x", str(wx), "--y", str(wy)])
            log(f"[*] Locked on window '{window}' at coordinates ({wx}, {wy}) on display {current_display}")
        else:
            log(f"[!] Window '{window}' not found, falling back to desktop origin.")
    elif display:
        cmd.extend(["--x", str(x), "--y", str(y)])

    proc = subprocess.Popen(cmd, env=os.environ, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    return proc

def wait_for_window(win_name, timeout=3.0):
    """Wait for X11 window to appear and return geometry"""
    start = time.time()
    while time.time() - start < timeout:
        geom = get_window_geometry(win_name)
        if geom:
            return geom
        time.sleep(0.15)
    return None

def configure_i3_window(win_name):
    """Float and size target window precisely in i3 window manager"""
    if shutil.which("i3-msg"):
        subprocess.run(
            ["i3-msg", f'[title="{win_name}"] floating enable, border pixel 0, resize set 1200 600'],
            capture_output=True,
            env=os.environ
        )

def main():
    parser = argparse.ArgumentParser(
        description="AGY Physical Cockpit - Unified Executive Companion",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""Examples:
  ./cockpit                     # Run 'agy' AI assistant in dedicated 1200x600 window streamed to physical display
  ./cockpit --inline            # Run interactive shell inside current terminal
  ./cockpit --test              # Stream dynamic test benchmark pattern
  ./cockpit --display :0.0      # Stream host desktop region to 1200x600 viewport
  ./cockpit --window Terminal   # Lock stream onto a specific X11 window
  ./cockpit -- btop             # Stream performance metrics while monitoring touch
"""
    )
    parser.add_argument("--inline", action="store_true", help="Run PTY session inline in current terminal (without dedicated GUI window)")
    parser.add_argument("--test", action="store_true", help="Stream dynamic benchmark test pattern")
    parser.add_argument("--display", help="Stream X11 display region (e.g. :1.0)")
    parser.add_argument("--window", help="Stream a specific X11 window by name")
    parser.add_argument("--fps", type=int, default=10, help="Streaming framerate (default: 10, prevents USB bus saturation)")
    parser.add_argument("--mode", choices=["dual", "full"], default="dual", help="Display mode (default: dual)")
    parser.add_argument("-p", "--profile", choices=["agy", "claude", "dsh", "shell"], help="AI Agent tactile profile (default: auto-detected from command)")
    parser.add_argument("--set-profile", choices=["agy", "claude", "dsh", "shell"], help="Switch hardware profile on physical dashcam and exit")
    parser.add_argument("--no-stream", action="store_true", help="Do not start display stream (PTY only)")
    parser.add_argument("cmd", nargs="*", default=None, help="Target command to run in PTY (default: agy)")
    args = parser.parse_args()

    if args.set_profile:
        subprocess.run([sys.executable, AGENT_SCRIPT, "-p", args.set_profile])
        return

    print("=" * 65)
    print("  \033[1;32mAGY PHYSICAL COCKPIT\033[0m — Zero-Typing Developer Console")
    print("  Allwinner V553 (1600x600 BOE Display • GT911 Capacitive Touch)")
    print("  Tactile Actions: [PROCEED] [GRILL-ME] [PLAN] [STOP]")
    print("=" * 65)

    gud_mode = is_gud_active()
    if gud_mode:
        log("\033[1;32m[+] Native Hardware USB Display (GUD) detected!\033[0m")
        setup_script = os.path.join(SCRIPT_DIR, "setup_gud_screen.sh")
        if os.path.exists(setup_script):
            subprocess.run([setup_script], check=False)

    default_agy = shutil.which("agy") or os.path.expanduser("~/.local/bin/agy")
    target_cmd = args.cmd if args.cmd else [default_agy]

    stream_proc = None
    term_proc = None

    if not args.inline and not args.test and not args.display and not args.window:
        # Dedicated Desktop Cockpit Window Mode (Default)
        win_title = "AGY-Cockpit"
        agent_exec = [sys.executable, AGENT_SCRIPT]
        if args.profile:
            agent_exec.extend(["-p", args.profile])
        agent_exec.extend(target_cmd)
        
        display = os.environ.get("DISPLAY") or ":1"
        os.environ["DISPLAY"] = display
        env = os.environ.copy()
        env["DISPLAY"] = display
        # Unset TMUX to allow clean attachment into existing or new sessions
        env.pop("TMUX", None)
        # Use Mesa software GL to protect against host NVIDIA driver/library mismatches
        env["__GLX_VENDOR_LIBRARY_NAME"] = "mesa"
        env["LIBGL_ALWAYS_SOFTWARE"] = "1"

        term_cmd = None
        alacritty_bin = shutil.which("alacritty")
        if alacritty_bin:
            try:
                t_check = subprocess.run([alacritty_bin, "-e", "true"], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=1.5)
                if t_check.returncode == 0:
                    term_cmd = [
                        alacritty_bin,
                        "--title", win_title,
                        "-o", 'window.decorations="None"',
                        "-o", 'window.padding.x=0',
                        "-o", 'window.padding.y=0',
                        "-e", *agent_exec
                    ]
                    log(f"Launching dedicated borderless Cockpit Terminal via Alacritty...")
            except Exception:
                pass

        if not term_cmd and shutil.which("xfce4-terminal"):
            term_cmd = [
                "xfce4-terminal",
                f"--title={win_title}",
                "--hide-menubar",
                "--hide-borders",
                "--hide-toolbar",
                "--geometry=120x30",
                "-x", *agent_exec
            ]
            log(f"Launching dedicated borderless Cockpit Terminal via xfce4-terminal...")
        elif not term_cmd and shutil.which("gnome-terminal"):
            term_cmd = [
                "gnome-terminal",
                f"--title={win_title}",
                "--geometry=120x30",
                "--", *agent_exec
            ]
            log(f"Launching dedicated Cockpit Terminal via gnome-terminal...")
        elif not term_cmd:
            term_cmd = agent_exec

        term_proc = subprocess.Popen(term_cmd, env=env)

        if gud_mode:
            time.sleep(0.3)
            if shutil.which("i3-msg"):
                subprocess.run(
                    ["i3-msg", f'[title="{win_title}"] floating disable, move container to workspace 4, focus'],
                    capture_output=True,
                    env=os.environ
                )
            log("Cockpit window assigned directly to GUD hardware display on Workspace 4.")
            log("Zero floating windows on primary monitors • Native GPU hardware output.")
        else:
            # Wait for window and configure i3
            time.sleep(0.4)
            configure_i3_window(win_title)
            geom = wait_for_window(win_title, timeout=3.0)
            if geom:
                wx, wy, ww, wh = geom
                log(f"Window '{win_title}' mapped at ({wx}, {wy}) size {ww}x{wh}")
            else:
                log(f"[!] Warning: Could not locate window '{win_title}' geometry, streaming desktop origin")

            if not args.no_stream:
                log(f"Starting display stream ({args.mode} mode @ {args.fps}fps) locked on '{win_title}'...")
                stream_proc = start_stream_subprocess(
                    mode=args.mode,
                    stream_type="window",
                    window=win_title,
                    fps=args.fps
                )

        log("Cockpit active! Touch screen buttons on the dashcam to interact.")
        log("Press Ctrl+C in this terminal to shut down.")

        try:
            term_proc.wait()
        except KeyboardInterrupt:
            log("Interrupted by user. Shutting down...")
            if term_proc.poll() is None:
                term_proc.terminate()
        finally:
            if stream_proc:
                log("Shutting down background display streamer...")
                stream_proc.terminate()
                try:
                    stream_proc.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    stream_proc.kill()
            log("Cockpit session closed cleanly.")
        return

    # Fallback / Inline Mode
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
        sys.path.insert(0, os.path.join(SCRIPT_DIR, "host_tools"))
        sys.path.insert(0, SCRIPT_DIR)
        import cockpit_agent
        if args.profile:
            cockpit_agent.switch_device_profile(args.profile)
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
