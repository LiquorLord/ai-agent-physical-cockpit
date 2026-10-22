# AGY Physical Cockpit

[English](README.md) | [简体中文](README_zh.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![SoC: Allwinner V553](https://img.shields.io/badge/SoC-Allwinner%20V553%20(sun8iw21)-blue.svg)]()
[![Display: 1600x600 MIPI](https://img.shields.io/badge/Display-1600x600%20MIPI--DSI-green.svg)]()
[![Dual-Boot: Non-Intrusive](https://img.shields.io/badge/Dual--Boot-100%25%20Non--Intrusive-success.svg)]()

> **"Unplug MicroSD: 100% original $40 factory dashcam. Plug in MicroSD: Dedicated edge AI multimodal physical cockpit for vibe-coding hackers."**

---

## 1. Project Overview

**AGY Physical Cockpit** is an open-source hardware/software transformation project that repurposes ubiquitous, cost-effective Allwinner V553 (`sun8iw21`) automotive smart screens into an intelligent, driverless physical companion for AI-assisted programming (Antigravity CLI / Vibe Coding).

### The Golden Rule: 100% Non-Intrusive Dual-Boot
- **Zero Soldering, Zero Risk**: No hardware modifications required.
- **MicroSD Removed**: Boots the unadulterated factory Linux 4.9 OS from 16MB SPI NOR Flash (`cdr` video recorder, wireless Apple CarPlay / HiCar).
- **MicroSD Inserted**: Allwinner on-chip BROM priority boots directly from SDC0 (Sector 16 eGON header) into our custom AI cockpit OS.

---

## 2. Key Features

- 🖥️ **Ultra-Wide 10.26" Touch Display (1600x600)**:
  - 4-Lane MIPI-DSI high-refresh bar display (`wtl1026boe2l`) with Goodix GT911 capacitive touchscreen.
  - Three-zone HUD layout: Live Thinking Chain, Active Code Diff / Plan Artifacts, and Physical Control Touch Buttons.
- 🔌 **Universal Driverless USB Composite Gadget**:
  - Enumerates over USB Type-C as **USB CDC-NCM Ethernet** (192.168.10.x, ~480Mbps) + **USB HID Keyboard** (`/dev/hidg0`).
  - Supported natively out-of-the-box on **Linux, macOS, and Windows** without installing third-party drivers.
- 🖲️ **Zero-Friction Physical Touch Controls**:
  - Large on-screen buttons: `🟢 PROCEED` (injects Enter / `y`), `🛑 STOP` (injects `Ctrl+C`), and `📋 PLAN` (injects `/plan`).
  - Keep your hands on the workflow without typing confirmations.
- 👁️ **Edge Multimodal AI**:
  - **Vision Engine**: Real-time onboard camera gesture tracking (👌 OK -> Proceed, ✌️ V -> Approve/Plan, ✋ Palm -> Emergency Stop).
  - **Dual-Tier Compute**: on-device Cortex-A7 NEON inference, or hybrid streaming to a local high-performance node.

---

## 3. Directory Structure

```
├── README.md / README_zh.md      # Project documentation (EN/ZH)
├── LICENSE                        # MIT License
├── CONTRIBUTING.md / _zh.md       # Contribution guide
├── CHANGELOG.md / _zh.md          # Version history
├── .github/workflows/ci.yml       # CI (syntax + shellcheck + path/secret guard)
├── cockpit -> host-tools/cockpit.py
├── docs/                          # Bilingual engineering documentation
│   ├── hardware_specs*.md         # Pinouts, MIPI timings, GT911 touch
│   ├── dual_boot_architecture*.md # BROM boot flow & SD layout
│   ├── mainline_linux_roadmap*.md # Mainline Linux 6.x porting roadmap
│   ├── V553_SYSTEM_REFERENCE*.md  # SoC / PMU / SPI NOR / USB reference
│   ├── V553_MAINLINE_DISPLAY_PORTING_GUIDE*.md
│   ├── MULTIMODAL_COCKPIT_ARCHITECTURE*.md
│   ├── MULTIMODAL_RESEARCH_ROADMAP*.md
│   └── GESTURE_ENGINE_ARCHIVE*.md
├── board/                         # Board bring-up (source, no vendored trees)
│   ├── mainline/                  # Linux 6.12 mainline bring-up
│   │   ├── dts/                   # sun8i-v853-v553-cockpit.dts + dtsi
│   │   ├── config/                # mainline_v853_defconfig
│   │   ├── patches/               # DRM/DE2/TCON/DSI patch + 150-file series
│   │   └── scripts/               # switch_to_gud.sh
│   └── buildroot/                 # Buildroot defconfig + modernization patch
├── firmware/                      # On-device runtime
│   ├── apps/                      # Go/C daemons (gud_v553, touch_cockpit, …)
│   ├── boot/                      # init / USB gadget scripts
│   ├── config/                    # cockpit_agents.json (agent picker)
│   └── splash/                    # bootsplash (splash.png)
├── host-tools/                    # Host PC companion tooling
│   ├── cockpit.py / *_agent.py    # unified CLI + PTY/ADB bridge
│   ├── stream_to_monitor.py       # 1600x600 display streamer
│   ├── flash_production_sd.sh     # dd full SD image
│   ├── update_kernel_sd.sh        # update zImage/dtb/userspace on SD
│   ├── gud_turbo/                 # GUD turbo kernel module (source only)
│   └── udev/                      # udev rules + install scripts
├── factory-reverse/               # Factory firmware reverse-engineering
│   ├── scripts/                   # UART interceptors, boot sniffers
│   ├── flasher/                   # flash_tool.go + sunxi-tools patch
│   └── dts/                       # Extracted factory board.dts
└── scripts/                       # Misc UART/dev helpers
    ├── uart_cmd.py
    ├── send_file_serial.py
    ├── reboot_and_capture.py
    └── apply_display_pipeline_fixes.py
```

> **Build toolchains and release images are intentionally NOT committed.**
> Buildroot / mainline kernel source trees, cross-compilers, and compiled
> binaries are pulled in during the build and published via GitHub/Gitee
> Releases — keeping `git clone` small and clean.

---

## 4. Hardware Specifications

| Component | Specification | Details |
| :--- | :--- | :--- |
| **SoC** | Allwinner V553 (`sun8iw21`) | 1x ARM Cortex-A7 @ 900MHz, NEON/VFPv4, E907 RISC-V, 1 TOPS NPU |
| **RAM** | 128 MB DDR3 | Integrated in SoC package |
| **Storage** | 16 MB SPI NOR Flash | `XM25QH128C` (factory firmware partition) |
| **MicroSD** | Up to 128 GB | SDC0 interface (BROM priority boot device) |
| **Display** | 10.26-inch IPS Bar Display | BOE `wtl1026boe2l`, 1600x600, 4-lane MIPI-DSI, DCLK 53.0MHz |
| **Touch** | Capacitive Multi-touch | Goodix GT911 on I2C1 |
| **Camera** | MIPI CSI Camera | `/dev/video0` front-facing wide angle |
| **Audio** | Mic & Speaker | Onboard MEMS microphone & audio codec |
| **USB** | Type-C USB 2.0 OTG | High-Speed 480Mbps, ConfigFS Composite Gadget (NCM+HID) |
| **UART** | Debug Serial Port | PCB test points `tx0`, `rx0`, `GND` @ 115200 8N1 |
| **Board ID** | `f900x-v2.2` | `km2026240602026.01.03jdh` |

---

## 5. Prerequisites

Everything here targets **ARMv7 hard-float** (`arm-linux-gnueabihf`).

| Requirement | Install (Ubuntu/Debian) | Used for |
| :--- | :--- | :--- |
| ARM cross toolchain | `sudo apt install gcc-arm-linux-gnueabihf` | C apps (`gud_v553`), kernel/module build |
| Go toolchain (≥1.22) | `sudo apt install golang-go` | firmware Go apps (`touch_cockpit`, …) |
| Python 3 + deps | `pip install pyserial numpy pillow` | `host-tools/` & `scripts/` |
| `sunxi-fel` | see `factory-reverse/flasher/README.md` | factory SPI flash access |
| Buildroot | `make v553_cockpit_defconfig && make` | full rootfs (external Bootlin toolchain) |

---

## 6. Build

### 6.1 Firmware apps (device binaries)

```bash
cd firmware/apps

# Go apps (static, CGO disabled)
for app in touch_cockpit gesture_detector send_hid_key fb_receiver v4l2_probe show_splash; do
  GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o ../bin/$app $app.go
done

# C apps
arm-linux-gnueabihf-gcc -static -O2 -o ../bin/gud_v553 gud_v553.c
arm-linux-gnueabihf-gcc -static -O2 -o ../bin/gud_test gud_test.c
```

Output lands in `firmware/bin/` (git-ignored).

### 6.2 GUD turbo kernel module

```bash
cd host-tools/gud_turbo
make -C /path/to/your/kernel ARCH=arm CROSS_COMPILE=arm-linux-gnueabihf- M=$PWD modules
```

### 6.3 Mainline Linux 6.12 kernel + DTB

1. Obtain a Linux 6.12 mainline tree.
2. Apply `board/mainline/patches/series/*` in order, then the DRM/DE2/TCON/DSI
   patch `board/mainline/patches/0001-arm-sun8i-v853-v553-Add-full-DRM-DE2-TCON-MIPI-DSI-D.patch`,
   and add `board/mainline/patches/panel-boe-wtl1026boe2l.c`.
3. Use `board/mainline/config/mainline_v853_defconfig`.
4. Build with `make ARCH=arm CROSS_COMPILE=arm-linux-gnueabihf- zImage dtbs`,
   using the DTS in `board/mainline/dts/`.

### 6.4 Buildroot rootfs

```bash
make -C board/buildroot v553_cockpit_defconfig
make -C board/buildroot
```

Uses the external Bootlin ARMv7 EABI-HF glibc toolchain; see
`board/buildroot/patches/0001-v553-gcc13-binutils242-modernization.patch`.

---

## 7. Flashing

### 7.1 Full production SD image (quickest path)

```bash
host-tools/flash_production_sd.sh   # dd a prebuilt .img to /dev/sdX
```

### 7.2 Update kernel / DTB / userspace on an existing SD card

```bash
host-tools/update_kernel_sd.sh      # auto-detects the SD boot partition
```

---

## 8. Host PC Streaming

Connect the device to your PC using a USB Type-C cable:

```bash
# Test display output with a high-speed 30fps stream
./host-tools/stream_to_monitor.py --test --fps 30

# Stream an active X11 desktop / window to the 1600x600 screen
./host-tools/stream_to_monitor.py --display :1.0 --fps 30

# Unified CLI (dual-zone streaming + PTY shell + touch injection)
./cockpit --help
```

---

## 9. License & Acknowledgements

- **License**: [MIT](LICENSE).
- **Acknowledgements**:
  - The `linux-sunxi` community for pioneering Allwinner reverse-engineering.
  - András Szemző and YuzukiHD for `sun8iw21` / V853 upstreaming efforts.
  - The Google DeepMind Antigravity project.
