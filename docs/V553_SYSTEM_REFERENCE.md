# Allwinner V553 (sun8iw21) System Architecture & Hardware Reference

[English](V553_SYSTEM_REFERENCE.md) | [简体中文](V553_SYSTEM_REFERENCE_zh.md)

---

## 1. Executive Silicon & Platform Profile

| Parameter | Specification | Notes / Hardware Truth |
| :--- | :--- | :--- |
| **Target Device** | 10.26-inch Smart Dashcam | Board model `f900x-v2.2 km2026240602026.01.03jdh` |
| **SoC** | Allwinner V553 (`sun8iw21p1`) | Compatible with V853; Heterogeneous Tri-Core SoC |
| **Main Application CPU** | ARM Cortex-A7 @ 900 MHz (Single-Core) | ARMv7-A, VFPv4, NEON, 32KB L1 I/D, 128KB L2 Cache |
| **Real-Time Co-processor** | T-Head XuanTie E907 RISC-V @ 300 MHz | Internal SRAM `0x1F100000`, Inter-core mailbox |
| **NPU / AI Engine** | VIP9000 @ 1.0 TOPS (INT8) | Embedded Vision AI / Lane Departure Detection |
| **DRAM Memory** | 128 MB DDR3 SIP (System-in-Package) | 16-bit bus @ 648 MHz (Base `0x40000000 ~ 0x47FFFFFF`) |
| **Primary Storage** | 16 MB SPI NOR Flash (`XM25QH128C`) | JEDEC ID `0x204018`, SPI0 CS0 |
| **Removable Storage** | MicroSD Slot (MMC0) | Port F (`PF0..PF5`), High-Speed 3.3V |
| **Power Management** | Internal On-Chip PMU (`allwinner,v853-pmu`) | Base `0x07090000`, NO external AXP2101 PMIC |
| **Display Panel** | BOE 10.26-inch Ultra-Wide Bar (`wtl1026boe2l`) | 1600 × 600, 4-lane MIPI-DSI, PWM9 backlight |
| **Touchscreen** | Goodix GT911 Capacitive Touch | I2C address `0x5d` / `0x14`, 1600 × 600 resolution |
| **Audio Subsystem** | Internal Audio Codec + Class-D Amp | AVCC calibration, onboard mic + speaker |
| **Cameras / Sensors** | Dual-Channel ISP Video Input | Front: MIPI-CSI (GalaxyCore GC2053 / SC2336); Rear: Analog AHD via TP9950 |
| **Wireless (Wi-Fi/BT)** | AltoBeam ATBM6165 | 802.11 b/g/n (SDIO) + Bluetooth 4.2 / BLE (UART) |
| **USB Interface** | USB 2.0 High-Speed OTG (480 Mbps) | Type-C connector, UDC controller `4100000.udc-controller` |
| **Debug UART** | Dedicated PCB Test Points (`tx0`, `rx0`, `GND`) | UART0 @ 115200 8N1 (Port E: `PE9` / `PE10`) |

---

## 2. Power Architecture & Internal PMU Regulators

### 2.1 The PMU Ground Truth
Unlike many Allwinner SBCs (such as TinyVision V851SE) that rely on external AXP2101 or AXP717 PMICs over I2C, **the V553 `f900x-v2.2` dashcam exclusively utilizes the SoC's built-in power management unit (PMC/PMU)**:
- Device Tree Node: `pmu@07090000`
- Compatible: `"allwinner,v853-pmu", "syscon", "simple-mfd"`
- **Critical Bootloader Consequence**: Any generic Boot0 binary compiled with external AXP2101 I2C communication fails during bus initialization (`[pmu]: bus init error`, `unknow PMU`), resulting in missing DDR core voltage and an immediate trap into USB FEL mode (`return_to_fel()`).

### 2.2 Voltage Rail Mapping
| Rail Name | Default Voltage | Destination / Subsystem | Status |
| :--- | :--- | :--- | :--- |
| `pmc-dcdc1` | 3.30 V (`0x325aa0` uV) | Peripheral I/O, SPI Flash, MicroSD VDD | Always-On / Boot-On |
| `pmc-dcdc2` | 0.90 V (`0xdbba0` uV) | SoC Core Logic (`VDD-SYS`, `VDD-CPU`) | Always-On / Boot-On |
| `pmc-dcdc3` | 1.50 V (`0x16e360` uV) | DDR3 DRAM Array & PHY (`VDD-DRAM`) | Always-On / Boot-On |
| `pmc-ldoa` | 1.80 V (`0x1b7740` uV) | MIPI-DSI / MIPI-CSI Analog PHY | Always-On / Boot-On |
| `pmc-aldo` | 1.80 V (`0x1b7740` uV) | Internal PLL & Audio AVCC | Always-On / Boot-On |
| `pmc-rtcldo`| 1.80 V (`0x1b7740` uV) | RTC & Deep Standby Logic | Always-On / Boot-On |
| `pmc-ldo33` | 3.30 V (`0x325aa0` uV) | Analog 3.3V, Sensor Power | Always-On / Boot-On |
| `pmc-ldo09` | 0.90 V (`0xdbba0` uV) | NPU & Internal DSP Core | Always-On / Boot-On |

---

## 3. DRAM Timings & Boot0 Parameter Layout

### 3.1 Physical Characteristics
- **Type**: DDR3 SIP (System-in-Package)
- **Clock Frequency**: 648 MHz (Register value `0x0288` = 648)
- **Bus Width**: 16-bit
- **Capacity**: 128 MB (1 Gb)
- **Address Space**: `0x40000000 ~ 0x47FFFFFF`

### 3.2 Calibrated `dram_para` Block (Offset `0x0030 ~ 0x0080` in Boot0)
The following 80-byte block represents the exact physical calibration extracted from factory SPI NOR Boot0:
```text
0x0030: 08 00 00 00 88 02 00 00 03 00 00 00 fb 7b 7b 00
0x0040: 01 00 00 00 d2 10 00 00 00 00 80 00 70 1c 00 00
0x0050: 42 00 00 00 18 00 00 00 00 00 00 00 52 a1 48 00
0x0060: 4d a9 b1 01 4f 20 07 00 96 78 78 b4 00 00 00 00
0x0070: 48 48 48 48 48 00 00 00 1e 12 21 16
```
- DRAM Clock: `dram_clk = 648 MHz`
- DRAM Type: `dram_type = 3` (DDR3)
- ZQ Calibration: Internal ZQ calibration enabled

---

## 4. Storage Architecture & Partition Tables

### 4.1 Onboard 16MB SPI NOR Flash (`XM25QH128C`)
- Total Sectors: 4,096 sectors @ 4 KB = 16,777,216 bytes.
- MTD Partition Map (from U-Boot environment):
  ```text
  mtdparts=spi0.0:128K(env),2368K(boot),12736K(rootfs),512K(userdata),128K(bootlogo),0K(UDISK)
  ```

| Partition Name | Offset (Byte) | Length | Format / Content |
| :--- | :--- | :--- | :--- |
| **SPL / Boot0** | `0x00000000` | 512 KB | Allwinner Boot0 (`eGON.BT0`) with SPI NOR driver |
| **U-Boot ENV** | `0x00080000` | 128 KB | U-Boot persistent environment variables |
| **boot** | `0x000A0000` | 2,368 KB | Linux Kernel (`zImage` / FIT Image, ~2.3MB) |
| **rootfs** | `0x002F0000` | 12,736 KB | Compressed SquashFS (Factory Tina Linux + custom dual-boot hooks) |
| **userdata** | `0x00F60000` | 512 KB | JFFS2 writable user configuration |
| **bootlogo** | `0x00FE0000` | 128 KB | Boot splash screen bitmap |

### 4.2 MicroSD Card Physical Architecture (MMC0)
- Controller: MMC0 @ Port F (`PF0..PF5`)
- GPIO Configuration Word: `22 00 00 00 77 00 00 00 00 01 00 34`
- Hardware Pinout:
  - `PF0`: SD0_D1
  - `PF1`: SD0_D0
  - `PF2`: SD0_CLK
  - `PF3`: SD0_CMD
  - `PF4`: SD0_D3
  - `PF5`: SD0_D2

---

## 5. Display & Touchscreen Pipeline

### 5.1 MIPI-DSI Panel (BOE `wtl1026boe2l`)
- **Dimensions**: 10.26-inch ultra-wide bar display (active area: 260 mm × 98 mm)
- **Native Resolution**: 1600 × 600 pixels
- **D-PHY Lanes**: 4 Data Lanes + 1 Clock Lane
- **Pixel Clock (DCLK)**: 53.0 MHz
- **Timings**:
  - Horizontal: Active 1600, HBP 160, HFP 40, HSYNC 40 (Total 1840)
  - Vertical: Active 600, VBP 20, VFP 10, VSYNC 10 (Total 640)
  - Refresh Rate: $53.0\text{ MHz} / (1840 \times 640) \approx 45.0\text{ Hz}$
- **Backlight Control**: PWM9 (`PB9` / `PD21`), Frequency: 50 kHz (`period_ns = 20000`, `duty_ns = 10000`)
- **Framebuffer Configuration**: `/dev/fb0`, 1600 × 600 RGB565 (16 bpp, 1.92 MB buffer) with hardware on-the-fly 90° rotation engine.

### 5.2 Touch Controller (Goodix GT911)
- Bus: I2C0 / I2C1
- Coordinates: 1600 × 600 native multi-touch
- Interrupt: Active-low falling edge
- Reset Pin: Dedicated GPIO reset toggle during bringup

---

## 6. USB OTG Subsystem & Peripheral Drivers

### 6.1 USB Controller Specifications
- IP Core: Allwinner Sunxi Musb High-Speed OTG Controller
- UDC Name: `4100000.udc-controller`
- Supported Modes:
  - **ADB (Android Debug Bridge)**: Vendor ID `0x18d1`, Product ID `0xd002` (FunctionFS `ffs.adb`)
  - **CDC-NCM (High-Speed Ethernet)**: Virtual network interface `usb0` (Target IP: `192.168.100.1/24`)
  - **USB HID Keyboard**: Virtual input device `/dev/hidg0` for keystroke injection

### 6.2 Wireless Subsystem (AltoBeam ATBM6165)
- **Wi-Fi**: 802.11 b/g/n 1T1R @ 2.4 GHz (SDIO interface, module `atbm6165.ko`)
- **Bluetooth**: BT 4.2 / BLE (UART interface, module `hci_uart.ko`, firmware `atbm6062_fw.bin`)

---

## 7. Mainline Linux (6.6 LTS / 6.12+) Migration Roadmap

### 7.1 Existing Upstream Tree Support
Mainline Linux already possesses basic SoC infrastructure for `sun8i-v853` (`arch/arm/boot/dts/allwinner/sun8i-v853.dtsi`):
- CPU (Cortex-A7) & GIC
- Clocks (`drivers/clk/sunxi-ng/ccu-sun8i-v853.c`)
- Pin Controller (`drivers/pinctrl/sunxi/pinctrl-sun8i-v853.c`)
- MMC Controller (`drivers/mmc/host/sunxi-mmc.c`)
- USB EHCI/OHCI & Musb UDC

### 7.2 Key Gaps to Close for Mainline Port
1. **Internal PMU Regulator Driver**:
   Mainline requires an explicit regulator driver for `allwinner,v853-pmu` or board-level fixed regulator bindings if voltages are left pre-configured by U-Boot.
2. **Display Engine 2.0 (DE2.0) & MIPI-DSI DRM Driver**:
   Port panel timings for `boe,wtl1026boe2l` into DRM panel-simple or dedicated driver (`panel-boe-wtl1026boe2l.c`).
3. **Goodix Touchscreen DTS Binding**:
   Standard `goodix,gt911` I2C node matching board interrupt/reset GPIOs.
4. **Bootloader Strategy**:
   - Modern U-Boot 2024.x or SyterKit configured with our exact 648 MHz DDR3 timings and MMC0 Port F pinout.
