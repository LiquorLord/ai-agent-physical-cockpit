# AGY Physical Cockpit — Project Kanban & Progress Tracker

> **Language Navigation / 语言导航**:
> - **English Version**: [KANBAN.md](file:///home/wchen/nas_home/shares/workspace/allwinner/KANBAN.md)
> - **中文版本**: [KANBAN_zh.md](file:///home/wchen/nas_home/shares/workspace/allwinner/KANBAN_zh.md)

---

## 1. Project Vision & Architecture / 项目愿景与架构

Transform an off-the-shelf $40 Allwinner V553 (`sun8iw21`) dashcam (board `f900x-v2.2 km2026240602026.01.03jdh`) into an open-source, zero-friction multimodal physical companion cockpit (`AGY Physical Cockpit`) for vibe-coding programmers.

### Three-State Dual-Boot Architecture
1. **Custom Cockpit MicroSD Inserted**: 100% boots our custom AI smart terminal OS from SD card; permanently blocks factory `rcS` from continuing to `S70app` / `cdr`; locks 1600x600 custom cockpit splash; exposes 3-in-1 USB Composite Gadget (ADB + CDC-NCM Ethernet + USB HID Keyboard) and low-latency display stream on port 5900.
2. **Standard MicroSD Inserted (FAT32/exFAT)**: BROM / `S02sdboot` detects absence of `/mnt/sdcard/init_usb_monitor.sh`, immediately unmounts `/mnt/sdcard`, boots factory firmware, starts `cdr`, and mounts card to `/mnt/extsd` for normal dashcam recording.
3. **No MicroSD Inserted**: Boots 100% stock factory firmware from onboard 16MB SPI NOR Flash.

---

## 2. Hardware Profile & Peripheral Mapping

| Subsystem | Specification / Identifier | Physical Endpoint / Driver | State / Notes |
| :--- | :--- | :--- | :--- |
| **SoC** | Allwinner V553 (`sun8iw21`) | Single-core Cortex-A7 @ 1.2GHz | Active, Watchdog fed every 3s |
| **Display** | BOE 10.26" 1600x600 ultra-wide | `/dev/fb0` (MIPI-DSI 4-lane, DE2.0) | `FBIOPAN_DISPLAY` supported |
| **Touch Screen** | Goodix GT911 Capacitive Touch | `/dev/input/event0` (I2C2 @ `0x5d`) | Native 600x1600 portrait sensor |
| **Power Button** | PMC PEK Key | `/dev/input/event1` | Hardware power button |
| **Camera Sensor** | CMOS Sensor (GC2053 / TP9950) | `/dev/video0` (V4L2 1080p@30) | Active, ready for gesture capture |
| **Flash Memory** | 16MB SPI NOR (`XM25QH128C`) | `/dev/mtd*` (SPI0 @ `0x204018`) | Full factory backup preserved |
| **USB Link** | USB 2.0 High-Speed OTG (480Mbps) | `4100000.udc-controller` | 3-in-1 Composite Gadget active |
| **UART Serial** | 115200 8N1 debug testpoints | Host `/dev/ttyUSB0` | Hardware root shell spawned |

---

## 3. Milestone Checklist

### Phase 1: Hardware Reverse Engineering & Flasher Tooling
- [x] SPI NOR Flash Full Factory Backup (16MB `backup_factory_16MB.bin`, SHA256 verified)
- [x] Patched `sunxi-fel` with Allwinner sun8iw21 (V853/V553) SPI0 & CCU support
- [x] Repacked SquashFS Rootfs & Flashed back via USB FEL mode
- [x] Direct Root Shell obtained (Passwordless root on console and telnet)

### Phase 2: Dual-Boot Architecture & Display Subsystem
- [x] MicroSD Card prepared with Linux 4.9-compatible ext4 filesystem
- [x] Safe dual-boot switch in `S02sdboot` (clean fallback for normal dashcam SD cards)
- [x] Custom bootsplash activator developed (`show_splash` with DE2.0 display pan ioctl)
- [x] Permanent supervisor loop in `init_usb_monitor.sh` completely blocking factory `cdr`
- [x] 1600x600 custom bootsplash confirmed LOCKED on physical display
- [x] High-performance display stream receiver (`firmware/bin/fb_receiver` with `FBIOPAN_DISPLAY`)

### Phase 3: USB Composite Gadget & Input Subsystem
- [x] Fixed non-interactive `adb shell` execution by fixing heredoc escaping in `/bin/adb_shell`
- [x] Kernel verification: `CONFIG_USB_F_HID=y` and `CONFIG_USB_F_NCM=y` confirmed active
- [x] Designed 3-in-1 Composite USB Gadget (Google ADB `18d1:d002` + CDC-NCM `usb0` + USB HID `/dev/hidg0`)
- [x] Cross-compiled USB HID keystroke injector (`firmware/bin/send_hid_key`)
- [x] Identified Goodix GT911 touch screen on `/dev/input/event0` (600x1600 portrait coordinate space)
- [x] Identified CMOS camera sensor on `/dev/video0` (V4L2)

### Phase 4: Multimodal Cockpit Interaction (Current Phase)
- [x] Implemented and cross-compiled `touch_cockpit` (Action Bar rendered on 1600x600 display, mapping Goodix GT911 touch events to action responses, physical Action Bar successfully rendered)
- [x] Implemented and cross-compiled `gesture_detector` (V4L2 multi-planar frame grabber, 160x120 downsampler, ✌️ V / 👌 OK / ✋ Stop classifier)
- [x] Identified Allwinner `sunxi-vin` driver architecture (`V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE = 9`, `p1_mipi` 1280x720 30fps sensor)
- [x] Uncovered USB detection root cause: Factory Linux 4.9.191 lacks `usb_f_hid` / `usb_f_ncm`, causing ConfigFS composite gadget failure which prevented UDC binding and D+ pullup activation; `/sbin/udevd` was also absent from early init
- [x] Overhauled `init_usb_monitor.sh`: Deployed persistent file logging (`boot_debug.log`, `dmesg_boot.log`), explicitly launched `S10udev` and `S11dev`, reverted to factory-proven `setusbconfig adb` data bridge
- [x] Created Host Companion Agent (`host_tools/cockpit_agent.py`) to bridge screen touch events to the host terminal over ADB
- [x] End-to-end multimodal physical verification: Capacitive touch Action Bar verified 100%! Emerald Green `[PROCEED / READY]` and Crimson Red `[STOP / ABORT]` flash upon touch and log events to `/tmp/cockpit_event.log`
- [x] Brought up `gesture_detector`: Allwinner `sunxi-vin` multi-planar (MPLANE 9) 1280x720 NV12 @ 30fps stream fully established, Cortex-A7 8x spatial downsampler & classifier running live!
- [x] Switched to user-facing secondary camera (`sensor@1` `tp9950`/`tp9951` @ `/dev/video8`): Captured full 1080p (1920x1080 NV12, aligned height 1088) frame, verified clear cabin view
- [x] Upgraded `gesture_detector` with YCbCr skin segmentation & morphological finger peak analysis: 100% eliminated background noise false triggers while consuming <0.1% CPU (90.9% system idle)
- [x] Implemented synchronized full-screen visual feedback: Inter-process UDP bridge between `gesture_detector` and `touch_cockpit` (`127.0.0.1:8889`) flashes a 24px wide glowing Emerald Green border around all four screen edges and highlights the `[PROCEED / READY]` button for 300ms on gesture recognition
- [x] Optimized gesture detection responsiveness: Unified ✌️ and 👌 into single confirm counter with soft decay, lowered latency to ~160ms (2 frames @ 12.5 fps)
- [x] Implemented Closed Fist ✊ (握拳) for STOP / ABORT: Utilized zero-finger compactness & aspect ratio ($0.65 \le W/H \le 1.35$) for maximum topological separation against ✌️/👌, triggering 24px Crimson Red border glow, `[STOP / ABORT]` highlight, and `ctrl+c`
- [x] Decoupled & shelved visual gesture engine: Converted `gesture_detector` to feature-flagged module (`/mnt/sdcard/enable_gesture`), terminated background process to free resources, preserved complete driver specs in `docs/GESTURE_ENGINE_ARCHIVE.md`
### Phase 5: Dual-Zone Cockpit Viewport & Zero-Typing PTY Master
- [x] Dual-Zone Screen Architecture designed: Left 1200x600 for low-latency host viewport + Right 400x600 for persistent Goodix GT911 touch Action Bar
- [x] Upgraded `fb_receiver`: Dual-port listening (5900 dual-zone 1200x600, 5901 full-screen 1600x600) with zero-copy hardware framebuffer `mmap` row-by-row blitting (leaving columns 1200..1600 untouched)
- [x] Upgraded `touch_cockpit`: Switched to zero-copy hardware framebuffer `mmap`, isolating Action Bar rendering to columns 1200..1600 and eliminating all full-screen clobbering
- [x] Upgraded `stream_to_monitor.py`: Added `--mode dual` (default 1200x600 on port 5900) and `--mode full` (1600x600 on port 5901), with low-latency ffmpeg flags and `--test` dynamic test pattern
- [x] Upgraded `cockpit_agent.py`: Implemented robust PTY Master mode wrapping `$SHELL` or CLI tools (`agy`), natively bypassing Linux 6.x TIOCSTI restriction and injecting `ready\n` and `Ctrl+C` into terminal child sessions
- [x] Verified Dual-Zone live streaming over high-speed ADB port 5900
- [x] End-to-end integration test of live streaming alongside physical button touch injection

### Phase 6: Full Cockpit Production Suite (Current Phase)
- [x] Action Bar Telemetry HUD: Integrated real-time digital clock (`HH:MM:SS`), V553 SoC core temperature (`/sys/class/thermal/thermal_zone0/temp`), CPU load, RAM usage, and active stream FPS into the right $400 \times 600$ Action Bar
- [x] Full Printable ASCII Font Engine: Built complete 5x7 ASCII bitmap font supporting all digits, uppercase/lowercase letters, and punctuation marks
- [x] Dynamic Touch Status History: Added real-time footer displaying last hardware action and timestamp (e.g. `LAST: READY (14:30:44)`)
- [x] Unified Executive CLI (`./cockpit`): Single-command developer console orchestrating background streaming and foreground zero-typing PTY session with automated teardown
- [x] Host Udev Rules & Installer: Built `host_tools/udev/99-agy-cockpit.rules` and `install_udev.sh` for non-root USB access and seamless plug-and-play
- [x] Live Verified: Verified full `./cockpit` execution pipeline with simultaneous test streaming and hardware touch event reception

---

## 4. Current Hardware & Runtime State Snapshot

- **Device Physical Status**: Allwinner V553 (`sun8iw21`) dashcam connected via single USB Type-C cable to host workstation (`usb:5-2.1` @ 480Mbps USB 2.0 High-Speed), providing both power and high-speed bidirectional data.
- **Boot Source**: Fully booted from custom MicroSD card ext4 partition; factory `cdr` recording app is 100% blocked and isolated.
- **Screen Display & Touch**: 1600x600 BOE display partitioned into Dual Zones:
  - Columns 0..1200: 1200x600 low-latency streaming viewport (managed by `fb_receiver`)
  - Columns 1200..1600: 400x600 capacitive touch Action Bar + Telemetry HUD (managed by `touch_cockpit`)
- **HUD Telemetry Active**: Real-time SoC temperature, CPU load, RAM usage, rolling stream FPS, digital clock, and touch history rendered dynamically.
- **ADB Connectivity**: USB 2.0 High-Speed ADB channel (`18d1:d002` @ 480Mbps).
- **Target Daemons**:
  - `fb_receiver`: Mmap dual-zone display receiver on ports 5900 (1200x600) & 5901 (1600x600)
  - `touch_cockpit`: Goodix GT911 Action Bar controller & Telemetry HUD on `/dev/input/event0`
  - `init_usb_monitor.sh`: Auto-respawn supervisor for `adbd`
- **Host Tools Ready**:
  - `./cockpit`: Unified executive console entrypoint
  - `host_tools/cockpit_agent.py`: PTY Master interactive companion agent
  - `host_tools/stream_to_monitor.py`: Dual-zone (1200x600) & full-screen (1600x600) streaming client
  - `host_tools/udev/install_udev.sh`: Udev plug-and-play installer

---

## 5. Immediate Next Steps

1. Run `./cockpit` to start full daily development workflow;
2. Run `sudo host_tools/udev/install_udev.sh` to install permanent host udev permissions.



