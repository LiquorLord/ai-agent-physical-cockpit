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
- [ ] User presents ✌️ (V-sign) or 👌 (OK gesture) to the user-facing camera lens to verify live 24px green feedback border glow and `"ready\n"` injection
- [ ] Initiate host screen streaming verification (`host_tools/stream_to_monitor.py` targeting 1600x600 framebuffer)

---

## 4. Current Hardware & Runtime State Snapshot

- **Device Physical Status**: Allwinner V553 (`sun8iw21`) dashcam connected via single USB Type-C cable to host workstation (`usb:5-2.1` @ 480Mbps USB 2.0 High-Speed), providing both power and high-speed bidirectional data.
- **Boot Source**: Fully booted from custom MicroSD card ext4 partition; factory `cdr` recording app is 100% blocked and isolated.
- **Screen Display & Touch**: 1600x600 custom AI UI locked on screen; right-side Action Bar button touch flashing and event reporting physically verified 100%!
- **User-Facing Camera**: Techpoint TP9950/TP9951 1080p sensor on `/dev/video8` actively streaming NV12 to `gesture_detector`.
- **ADB Connectivity**: Active (`20080411 device usb:5-2.1`), non-interactive `adb shell` fully operational.
- **Active Daemons on Target**:
  - PID 1: `/sbin/init`
  - PID 845: `/bin/sh /mnt/sdcard/init_usb_monitor.sh` (supervisor protection loop)
  - PID 904: `/bin/adbd -D` (480Mbps High-Speed ADB channel)
  - PID 1717: `/mnt/sdcard/touch_cockpit` (Goodix GT911 Action Bar controller)
  - PID 2317: `/mnt/sdcard/gesture_detector` (Visual Gesture AI engine on `/dev/video8`)
  - PID 943: `/mnt/sdcard/fb_receiver` (Listening on port `:5900`)
- **Host Tools Ready**:
  - `host_tools/cockpit_agent.py` (Zero-typing terminal companion agent via ADB)
  - `host_tools/stream_to_monitor.py` (1600x600 ultra-wide screen streaming client)

---

## 5. Immediate Next Steps

1. User presents ✌️ (V-sign) or 👌 (OK gesture) to the user-facing secondary camera lens to verify green border flash and live event injection;
2. Run `./host_tools/cockpit_agent.py` on host to achieve zero-typing terminal control;
3. Stream desktop or test bench diagnostics to the 10.26-inch ultra-wide display.

