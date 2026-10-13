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
  - **Dual-Tier Compute**:
    - *Notebook Mode*: Offline on-device inference using ARM Cortex-A7 NEON.
    - *Workstation Mode*: Hybrid streaming to local high-performance AI nodes (`192.168.1.188` / RTX 4090) for full speech recognition & LLM processing.

---

## 3. Directory Structure

```
├── README.md                      # English project documentation and quickstart
├── README_zh.md                   # Chinese project documentation and quickstart
├── GEMINI.md                      # Persistent guidelines & project memory
├── docs/                          # Detailed engineering documentation
│   ├── hardware_specs.md          # Hardware pinouts, MIPI timings, GT911 touch (EN)
│   ├── hardware_specs_zh.md       # Hardware specifications (ZH)
│   ├── dual_boot_architecture.md  # BROM hardware boot flow & SD layout (EN)
│   ├── dual_boot_architecture_zh.md # Dual-boot architecture (ZH)
│   ├── mainline_linux_roadmap.md  # Mainline Linux 6.x & SyterKit porting (EN)
│   └── mainline_linux_roadmap_zh.md # Mainline Linux roadmap (ZH)
├── factory_reverse/               # Factory reverse engineering & recovery
│   ├── backups/                   # Pristine factory 16MB SPI dump & SHA256
│   ├── dts/                       # Extracted Device Tree sources (DTS/DTB)
│   ├── flasher/                   # SPI flash tools & patched sunxi-tools
│   ├── rootfs/                    # Factory SquashFS dumps & unpacked rootfs
│   ├── scripts/                   # UART interceptors & boot sniffers
│   └── logs/                      # Raw boot logs and hardware traces
├── firmware/                      # Target OS & runtime for MicroSD
│   ├── apps/                      # Core device daemons (fb_receiver, cockpit-core)
│   ├── bin/                       # Precompiled ARMv7 static binaries
│   ├── boot/                      # Custom bootloader & initialization scripts
│   └── splash/                    # 1600x600 Bootsplash assets
├── host_tools/                    # Host PC companion tools & sidecar
│   └── stream_to_monitor.py       # Desktop & X11 display streamer
└── scratch/                       # Temporary workspace for rapid prototyping
```

---

## 4. Hardware Specifications

| Component | Specification | Details |
| :--- | :--- | :--- |
| **SoC** | Allwinner V553 (`sun8iw21`) | 1x ARM Cortex-A7 @ 900MHz, NEON/VFPv4, E907 RISC-V, 1 TOPS NPU |
| **RAM** | 128 MB DDR3 | Integrated in SoC package |
| **Storage** | 16 MB SPI NOR Flash | `XM25QH128C` (Factory firmware partition) |
| **MicroSD** | Up to 128 GB | SDC0 interface (BROM priority boot device) |
| **Display** | 10.26-inch IPS Bar Display | BOE `wtl1026boe2l`, 1600x600, 4-lane MIPI-DSI, DCLK 53.0MHz |
| **Touch** | Capacitive Multi-touch | Goodix GT911 on I2C1 |
| **Camera** | MIPI CSI Camera | `/dev/video0` front-facing wide angle |
| **Audio** | Mic & Speaker | Onboard MEMS microphone & audio codec |
| **USB** | Type-C USB 2.0 OTG | High-Speed 480Mbps, ConfigFS Composite Gadget (NCM+HID) |
| **UART** | Debug Serial Port | PCB test points `tx0`, `rx0`, `GND` @ 115200 8N1 |
| **Board ID**| `f900x-v2.2` | `km2026240602026.01.03jdh` |

---

## 5. Quickstart

### 5.1 Preparing the MicroSD Card
1. Format a MicroSD card partition as Ext4:
   ```bash
   sudo mkfs.ext4 -O ^64bit -L "COCKPIT" /dev/sdX1
   ```
2. Copy the firmware runtime into `/mnt/sdcard/`:
   ```bash
   cp firmware/bin/fb_receiver /mnt/sdcard/
   cp firmware/splash/splash.raw /mnt/sdcard/
   cp firmware/boot/init_usb_monitor.sh /mnt/sdcard/
   ```

### 5.2 Host PC Streaming
Connect the device to your PC using a USB Type-C cable:
```bash
# Test display output with high-speed 30fps stream
./host_tools/stream_to_monitor.py --test --fps 30

# Stream active X11 desktop / window to the 1600x600 screen
./host_tools/stream_to_monitor.py --display :1.0 --fps 30
```

---

## 6. License & Acknowledgements

- **License**: MIT License.
- **Acknowledgements**:
  - The `linux-sunxi` community for pioneering Allwinner reverse-engineering.
  - András Szemző and YuzukiHD for `sun8iw21` / V853 upstreaming efforts.
  - The Google DeepMind Antigravity project.
