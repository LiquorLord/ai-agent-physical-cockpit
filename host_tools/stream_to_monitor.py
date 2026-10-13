#!/usr/bin/env python3
"""
USB Monitor Streamer for Allwinner V553 (1600x600 BOE Display)
全志 V553 便携副屏实时串流投屏脚本 (英汉双语 / Bilingual)
"""

import sys
import os
import subprocess
import time
import argparse

DEFAULT_RES = "1600x600"
DEFAULT_FPS = 30
DEFAULT_PORT = 5900
DEFAULT_DISPLAY = os.environ.get("DISPLAY", ":1")

def check_adb():
    """Verify ADB connection to the dashcam device / 验证设备 ADB 连接"""
    res = subprocess.run(["adb", "devices"], capture_output=True, text=True)
    lines = [line for line in res.stdout.strip().split("\n") if "\tdevice" in line]
    if not lines:
        print("[!] No ADB device detected. Checking serial or waiting for USB...")
        return False
    serial = lines[0].split("\t")[0]
    print(f"[*] Found target display device: {serial}")
    return True

def setup_forwarding(port=DEFAULT_PORT):
    """Setup ADB port forwarding for display streaming / 设置 ADB 端口映射"""
    subprocess.run(["adb", "forward", f"tcp:{port}", f"tcp:{port}"], check=True)
    print(f"[*] Port forwarding enabled: localhost:{port} -> device:{port}")

def stream_x11(display=":0.0", x_offset=0, y_offset=0, fps=DEFAULT_FPS, port=DEFAULT_PORT):
    """
    Stream X11 desktop region directly to the USB monitor
    将本地 X11 桌面指定区域实时串流到 USB 显示屏
    """
    print(f"[*] Starting display streaming from {display} ({DEFAULT_RES} @ {fps}fps)...")
    ffmpeg_cmd = [
        "ffmpeg",
        "-loglevel", "warning",
        "-f", "x11grab",
        "-video_size", DEFAULT_RES,
        "-framerate", str(fps),
        "-i", f"{display}+{x_offset},{y_offset}",
        "-f", "rawvideo",
        "-pix_fmt", "rgb565le",
        f"tcp://127.0.0.1:{port}"
    ]
    print(f"[*] Executing pipeline: {' '.join(ffmpeg_cmd)}")
    try:
        subprocess.run(ffmpeg_cmd)
    except KeyboardInterrupt:
        print("\n[*] Streaming stopped by user.")

def stream_test_pattern(fps=DEFAULT_FPS, port=DEFAULT_PORT):
    """
    Stream live dynamic test pattern for latency & FPS benchmarking
    输出动态测试图案进行帧率与低时延基准测试
    """
    print(f"[*] Streaming dynamic test pattern ({DEFAULT_RES} @ {fps}fps)...")
    ffmpeg_cmd = [
        "ffmpeg",
        "-loglevel", "warning",
        "-re",
        "-f", "lavfi",
        "-i", f"testsrc=size={DEFAULT_RES}:rate={fps}",
        "-f", "rawvideo",
        "-pix_fmt", "rgb565le",
        f"tcp://127.0.0.1:{port}"
    ]
    try:
        subprocess.run(ffmpeg_cmd)
    except KeyboardInterrupt:
        print("\n[*] Benchmark stopped by user.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="V553 USB Monitor Streamer")
    parser.add_argument("--test", action="store_true", help="Stream dynamic test pattern instead of desktop")
    parser.add_argument("--display", default=DEFAULT_DISPLAY, help=f"X11 display identifier (default: {DEFAULT_DISPLAY})")
    parser.add_argument("--x", type=int, default=0, help="Screen capture X offset")
    parser.add_argument("--y", type=int, default=0, help="Screen capture Y offset")
    parser.add_argument("--fps", type=int, default=DEFAULT_FPS, help="Stream target framerate (default: 30)")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help="Streaming port (default: 5900)")
    args = parser.parse_args()

    if not check_adb():
        print("[!] Please ensure the device is booted with USB Type-C plugged in.")
        sys.exit(1)

    setup_forwarding(args.port)

    if args.test:
        stream_test_pattern(fps=args.fps, port=args.port)
    else:
        stream_x11(display=args.display, x_offset=args.x, y_offset=args.y, fps=args.fps, port=args.port)
