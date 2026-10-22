# AGY Physical Cockpit Dual-Boot Architecture
**Allwinner V553 (sun8iw21) – eGON BROM Interception & Non-Intrusive Dual-Boot**

[English](dual_boot_architecture.md) | [简体中文](dual_boot_architecture_zh.md)

| **Version** | **Date**       | **Author** | **Status** |
|-------------|----------------|------------|------------|
| 1.0         | 2026-10-13     | AGY Team   | Approved   |

---

## Table of Contents
1. [Executive Summary & Dual-Boot Philosophy](#1-executive-summary--dual-boot-philosophy)
2. [Allwinner Silicon BROM Hardware Priority](#2-allwinner-silicon-brom-hardware-priority)
3. [Software Layer Fallback (Dual Safety Net)](#3-software-layer-fallback-dual-safety-net)
4. [MicroSD Partition & Filesystem Architecture](#4-microsd-partition--filesystem-architecture)
5. [Watchdog Management](#5-watchdog-management)
6. [Open-Source User Experience](#6-open-source-user-experience)

---

## 1. Executive Summary & Dual-Boot Philosophy

The AGY Physical Cockpit implements a **100% non-intrusive dual-boot** mechanism on the Allwinner V553 (sun8iw21) SoC:
- **MicroSD inserted**: The system boots a custom modern AI-smart terminal OS (Linux with USB Composite Gadget, touch UI, edge vision).
- **MicroSD removed**: The system boots the **100% unadulterated factory dashcam firmware** (`cdr` recording, wireless Apple CarPlay / HiCar).

No modification is ever made to the internal SPI NOR Flash. The factory firmware remains pristine. The dual-boot decision is made at cold-boot time by the hardware BROM, without any software-level tampering.

---

## 2. Allwinner Silicon BROM Hardware Priority

The Allwinner V553 (sun8iw21) contains a **hardwired Boot ROM (BROM)** that enforces a strict boot order:

```
SDC0 (MMC/SD card 0) ──> SPI NOR (SPI0) ──> USB FEL Mode
```

The BROM scans **SDC0** first. If a valid **eGON header** is found at **sector 16 (8 KB offset)** on the inserted MicroSD, the BROM loads the corresponding boot image directly into SRAM and executes it.
- **Why sector 16?** The BROM's eGON signature check is hard-coded to look for the magic `eGON` signature at offset 0x2000 (8192 bytes) from the start of the SD card. This is a silicon-level constant that cannot be altered by software.
- **Pure hardware-level cold boot interception:** Because the BROM checks SDC0 **before** SPI NOR, the MicroSD can **completely override** the factory boot flow without touching the SPI Flash. The factory firmware in SPI NOR is never read or executed.
- **Result:** The custom OS runs in a completely isolated environment. The factory dashcam firmware is never touched unless the MicroSD is removed.

---

## 3. Software Layer Fallback (Dual Safety Net)

Although the hardware BROM interception is the primary mechanism, a **software fallback** is provided for cases where the MicroSD is inserted but boots through the SPI NOR bootloader. This ensures the system always switches seamlessly.

The SPI NOR rootfs contains a dual-boot init script:  
`/etc/init.d/S02sdboot`

This script performs the following logic:

```bash
# S02sdboot – dual-boot init script
if [ -b /dev/mmcblk0p1 ]; then
    # MicroSD detected with a valid partition
    mount /dev/mmcblk0p1 /mnt/sdcard
    if [ -x /mnt/sdcard/init_usb_monitor.sh ]; then
        # Hand off control to the custom OS init script
        exec /mnt/sdcard/init_usb_monitor.sh
    fi
fi
# If no MicroSD or no custom script, continue factory boot
# (S70app / cdr runs as normal)
```

- **Detection:** The script checks for the existence of `/dev/mmcblk0p1` (the first partition of the SD card).
- **Handoff:** If the custom OS's `init_usb_monitor.sh` is present and executable, it is executed **in place** of the factory `S70app` (`cdr`).
- **Bypass:** The factory dashcam application (`S70app`) is never started. The custom OS takes full control.
- **Safety:** If the MicroSD is removed or the script is missing, the factory boot chain continues unmodified.

---

## 4. MicroSD Partition & Filesystem Architecture

The MicroSD card is formatted with an ext4 filesystem compatible with both modern and legacy kernels:

| **Offset** | **Size**       | **Content**                     |
|------------|----------------|---------------------------------|
| 0x0000     | 8 KB           | Partition table (MBR)           |
| 0x2000     | 8 KB           | Sector 16: eGON header (SyterKit / U-Boot SPL) |
| 0x100000   | Remainder      | Partition 1: ext4 filesystem (`/mnt/sdcard`) |

**Filesystem configuration:**
- **Type:** ext4 (Linux 4.9 and 6.x compatible).
- **Critical configuration:** `64bit` feature **disabled** (`-O ^64bit`). This avoids the 64-bit inode trap that causes incompatibility with 32-bit legacy toolchains and older kernels.
- **Mount options:** `noatime, nodiscard, data=ordered`.

---

## 5. Watchdog Management

The Allwinner V553 integrates a **sun8i-wdt** hardware watchdog timer. In factory firmware, the watchdog resets the SoC after **60 seconds** of inactivity. During custom OS boot, the watchdog is continuously fed to prevent unexpected resets.

**Strategy:**
1. **Userspace daemon:** `init_usb_monitor.sh` starts an asynchronous background feeding loop:
   ```bash
   ( while true; do echo 1 > /dev/watchdog 2>/dev/null; sleep 3; done ) &
   ```
2. **Application-level integration:** Core binaries (`fb_receiver`, `cockpit-core`) can also manage `/dev/watchdog` directly.
3. **Fallback safety:** If the custom OS crashes or ungracefully stops, the hardware watchdog triggers and reboots the board cleanly.

---

## 6. Open-Source User Experience

The AGY Physical Cockpit dual-boot system is designed for **zero-soldering, zero-risk** developer use:

- **No hardware modification:** The MicroSD slot is the only interface needed.
- **No firmware flashing required:** The custom OS lives entirely on the MicroSD.
- **Instant recovery:** If the custom OS is removed, the factory dashcam boots immediately.
- **Community promise:** *"Plug in MicroSD -> modern AI terminal. Unplug it -> original factory dashcam. Zero soldering, zero flashing, zero risk."*
