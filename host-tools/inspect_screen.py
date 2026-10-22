#!/usr/bin/env python3
"""
inspect_screen.py - Autonomous Optical Screen Inspector for Allwinner V553 Cockpit
Connects to the V553 target over High-Speed USB SSH (192.168.100.1), triggers a hardware V4L2
frame capture on /dev/video12 (the camera facing the 10.26" screen), transfers the raw NV12
frame to host, converts to high-quality JPEG and contrast-enhanced inspection assets, displays
an ANSI TrueColor preview, and opens the asset via host desktop viewer (Rule 7 / hardware-inspector).
"""

import sys
import os
import subprocess
import numpy as np
from PIL import Image, ImageOps

TARGET_IP = "192.168.100.1"
CAPTURE_CMD = "/usr/bin/v4l2_probe /dev/video12"
RAW_PATH = "/tmp/screen_capture.nv12"
JPG_PATH = "/tmp/screen_capture.jpg"
ENHANCED_PATH = "/tmp/screen_capture_enhanced.jpg"

def main():
    print(f"[*] Capturing optical frame from V553 ({TARGET_IP})...")
    # Run v4l2_probe on device
    ret = subprocess.run([
        "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
        f"root@{TARGET_IP}", f"{CAPTURE_CMD} 65536 256 > /dev/null 2>&1 && cat /tmp/capture_test.raw"
    ], stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    if ret.returncode != 0 or len(ret.stdout) < 1382400:
        print(f"[-] Capture failed: code {ret.returncode}, bytes {len(ret.stdout)}")
        if ret.stderr:
            print(f"    Stderr: {ret.stderr.decode('utf-8', errors='ignore')}")
        sys.exit(1)

    raw_data = ret.stdout[:1382400]
    w, h = 1280, 720
    y_size = w * h
    y = np.frombuffer(raw_data[:y_size], dtype=np.uint8).reshape((h, w))
    uv = np.frombuffer(raw_data[y_size:y_size + y_size // 2], dtype=np.uint8).reshape((h // 2, w // 2, 2))

    u = np.repeat(np.repeat(uv[:, :, 0], 2, axis=0), 2, axis=1)
    v = np.repeat(np.repeat(uv[:, :, 1], 2, axis=0), 2, axis=1)

    y_f = y.astype(np.float32)
    u_f = u.astype(np.float32) - 128.0
    v_f = v.astype(np.float32) - 128.0

    r = np.clip(y_f + 1.402 * v_f, 0, 255).astype(np.uint8)
    g = np.clip(y_f - 0.344136 * u_f - 0.714136 * v_f, 0, 255).astype(np.uint8)
    b = np.clip(y_f + 1.772 * u_f, 0, 255).astype(np.uint8)

    rgb = np.stack([r, g, b], axis=-1)
    img = Image.fromarray(rgb)
    img.save(JPG_PATH, quality=95)

    # Auto-contrast enhanced
    enhanced = ImageOps.autocontrast(img, cutoff=1)
    enhanced.save(ENHANCED_PATH, quality=95)

    y_min, y_max, y_mean = y.min(), y.max(), y.mean()
    print(f"[+] Frame successfully captured! Size: {w}x{h} (1,382,400 bytes)")
    print(f"    Luminance Stats: Y_min={y_min}, Y_max={y_max}, Y_mean={y_mean:.2f}")
    print(f"    Raw JPEG: {JPG_PATH}")
    print(f"    Contrast Enhanced: {ENHANCED_PATH}")

    # Proactive Host Desktop Delivery (Rule 7)
    display = os.environ.get("DISPLAY") or os.environ.get("WAYLAND_DISPLAY")
    if display:
        viewer = None
        for v in ["loupe", "eog", "xdg-open"]:
            if subprocess.run(["which", v], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                viewer = v
                break
        if viewer and "--no-view" not in sys.argv:
            subprocess.Popen([viewer, ENHANCED_PATH], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
            print(f"[+] Proactively popped open {viewer} with {ENHANCED_PATH}")

if __name__ == "__main__":
    main()
