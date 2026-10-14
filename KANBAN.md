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
- [ ] End-to-end multimodal physical verification: User inserts MicroSD card and plugs in USB to verify ADB reconnect and live touch action response

---

## 4. Current Hardware & Runtime State Snapshot

- **Device Physical Status**: MicroSD card updated with new boot engine & diagnostic logs, safely unmounted (`sync && udisksctl unmount -b /dev/sdc1`), waiting for user to insert into V553 dashcam.
- **Boot Source**: About to boot from custom MicroSD card.
- **ADB Connectivity**: Host background listener active, waiting for device attachment.
- **Synchronized SD Card Files**:
  - `/mnt/sdcard/init_usb_monitor.sh` (Persistent `boot_debug.log` + udev daemons + factory UDC bind)
  - `/mnt/sdcard/touch_cockpit` (GT911 Action Bar controller)
  - `/mnt/sdcard/gesture_detector` (Visual gesture classifier)
  - `/mnt/sdcard/fb_receiver` (DE2.0 `FBIOPAN_DISPLAY` frame receiver)
  - `/mnt/sdcard/show_splash`
  - `/mnt/sdcard/splash.raw`

---

## 5. Immediate Next Steps

1. User removes MicroSD card from PC reader and inserts it into Allwinner V553 dashcam;
2. Reconnect USB Type-C cable to host PC to power on;
3. Host background listener will automatically detect device, stream boot logs, and verify touch interactions!

   - Push to `/mnt/sdcard/touch_cockpit` and test physical touch tap.

2. **Step 2: Build & Deploy `gesture_detector`**:
   - Write `firmware/apps/gesture_detector.go`:
     - Opens `/dev/video0` via standard V4L2 ioctl.
     - Downsamples frames to 160x120 grayscale.
     - Detects ✌️ V-sign and 👌 OK gestures with temporal debounce filter.
     - On gesture confirmation: renders green screen feedback and executes `/mnt/sdcard/send_hid_key "ready\n"`.
   - Cross-compile for ARMv7 and push to `/mnt/sdcard/gesture_detector`.

3. **Step 3: End-to-End Multimodal Physical Verification**:
   - Verify that gesturing ✌️ or 👌 in front of the cockpit screen automatically inputs `"ready"` into the developer's terminal prompt.
