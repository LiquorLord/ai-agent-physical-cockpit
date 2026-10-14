#!/usr/bin/env python3
"""
USB Monitor Streamer for Allwinner V553 (1600x600 BOE MIPI Display)
全志 V553 便携副屏/座舱双分区实时串流投屏脚本 (英汉双语 / Bilingual)

Architecture:
- Dual-Zone Mode (Default): Streams 1200x600 viewport to port 5900, leaving right 400x600
  Action Bar untouched and fully responsive for touch controls.
- Full-Screen Mode: Streams full 1600x600 display to port 5901.
"""

import sys
import os
import subprocess
import time
import argparse

DUAL_RES = "1200x600"
FULL_RES = "1600x600"
DUAL_PORT = 5900
FULL_PORT = 5901
DEFAULT_FPS = 30
DEFAULT_DISPLAY = os.environ.get("DISPLAY", ":0.0")

def check_adb():
    """Verify ADB connection to the Allwinner V553 device / 验证设备 ADB 连接"""
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    lines = [line for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if not lines:
        print("[!] No ADB device detected. Checking serial or waiting for USB...")
        return False
    serial = lines[0].split("\t")[0]
    print(f"[*] Found target display device: {serial}")
    return True

def setup_forwarding(port):
    """Setup ADB port forwarding for display streaming / 设置 ADB 端口映射"""
    subprocess.run(["adb", "forward", f"tcp:{port}", f"tcp:{port}"], check=True)
    print(f"[*] Port forwarding verified: localhost:{port} -> device:{port}")

def stream_x11(display=":0.0", x_offset=0, y_offset=0, res=DUAL_RES, fps=DEFAULT_FPS, port=DUAL_PORT):
    """
    Stream X11 desktop region directly to the USB monitor
    将本地 X11 桌面指定区域实时串流到 USB 显示屏
    """
    print(f"[*] Starting X11 desktop streaming from {display} ({res} @ {fps}fps, port {port})...")
    ffmpeg_cmd = [
        "ffmpeg",
        "-loglevel", "warning",
        "-f", "x11grab",
        "-video_size", res,
        "-framerate", str(fps),
        "-i", f"{display}+{x_offset},{y_offset}",
        "-f", "rawvideo",
        "-pix_fmt", "rgb565le",
        f"tcp://127.0.0.1:{port}?tcp_nodelay=1&send_buffer_size=1048576"
    ]
    print(f"[*] Executing pipeline: {' '.join(ffmpeg_cmd)}")
    try:
        subprocess.run(ffmpeg_cmd)
    except KeyboardInterrupt:
        print("\n[*] Streaming stopped by user.")

def stream_test_pattern(res=DUAL_RES, fps=DEFAULT_FPS, port=DUAL_PORT):
    """
    Stream live dynamic test pattern for latency & FPS benchmarking
    输出动态测试图案进行帧率与低时延基准测试
    """
    print(f"[*] Streaming dynamic test pattern ({res} @ {fps}fps, port {port})...")
    # testsrc2 provides counter, color gradient, and dynamic timecode
    ffmpeg_cmd = [
        "ffmpeg",
        "-loglevel", "warning",
        "-re",
        "-f", "lavfi",
        "-i", f"testsrc2=size={res}:rate={fps}",
        "-f", "rawvideo",
        "-pix_fmt", "rgb565le",
        f"tcp://127.0.0.1:{port}?tcp_nodelay=1&send_buffer_size=1048576"
    ]
    try:
        subprocess.run(ffmpeg_cmd)
    except KeyboardInterrupt:
        print("\n[*] Benchmark stopped by user.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="AGY Physical Cockpit / V553 USB Monitor Streamer (Bilingual)",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    parser.add_argument(
        "--mode", choices=["dual", "full"], default="dual",
        help="Streaming mode: 'dual' (1200x600 viewport + right Action Bar) or 'full' (1600x600)"
    )
    parser.add_argument("--test", action="store_true", help="Stream dynamic test pattern instead of desktop")
    parser.add_argument("--display", default=DEFAULT_DISPLAY, help=f"X11 display identifier")
    parser.add_argument("--x", type=int, default=0, help="Screen capture X offset")
    parser.add_argument("--y", type=int, default=0, help="Screen capture Y offset")
    parser.add_argument("--fps", type=int, default=DEFAULT_FPS, help="Stream target framerate")
    parser.add_argument("--port", type=int, default=None, help="Custom streaming port override")
    args = parser.parse_args()

    if not check_adb():
        print("[!] Please ensure the device is booted with USB Type-C plugged in.")
        sys.exit(1)

    if args.mode == "dual":
        target_res = DUAL_RES
        target_port = DUAL_PORT if args.port is None else args.port
    else:
        target_res = FULL_RES
        target_port = FULL_PORT if args.port is None else args.port

    setup_forwarding(target_port)

    if args.test:
        stream_test_pattern(res=target_res, fps=args.fps, port=target_port)
    else:
        stream_x11(display=args.display, x_offset=args.x, y_offset=args.y, res=target_res, fps=args.fps, port=target_port)
