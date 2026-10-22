# Allwinner V553 (sun8iw21) Dashcam Hardware Platform – Board f900x-v2.2

[English](hardware_specs.md) | [简体中文](hardware_specs_zh.md)

## 1. System Architecture & SoC Profile

### 1.1 SoC Overview

| Feature | Specification |
|--------|---------------|
| **SoC** | Allwinner V553 (sun8iw21) |
| **CPU** | ARM Cortex-A7 @ 900 MHz (single-core) |
| **NPU** | VIP9000 @ 1 TOPS (8-bit integer) |
| **Co-processor** | E907 RISC-V @ 200 MHz (system control, low-power tasks) |
| **Memory** | 128 MB DDR3 (integrated in package, 16-bit bus @ 1600 MHz) |
| **Boot ROM** | 32 KB |
| **SPI NOR Flash** | XM25QH128C (16 MB, 128 Mb) – holds bootloader, kernel, rootfs |
| **Package** | eBGA 0.65 mm pitch, 12×12 mm |

### 1.2 Memory Map

| Region | Address Range | Size | Description |
|--------|---------------|------|-------------|
| DDR3 | 0x40000000 – 0x47FFFFFF | 128 MB | System RAM |
| SPI NOR (XM25QH128C) | 0x0 – 0x0FFFFFFF (via SPI0) | 16 MB | Boot, kernel, rootfs |
| SRAM (A7) | 0x1F000000 – 0x1F00FFFF | 64 KB | CPU local SRAM |
| SRAM (E907) | 0x1F100000 – 0x1F10FFFF | 64 KB | RISC-V local SRAM |

### 1.3 Boot Sequence

1. **ROM boot** → load from SPI NOR (SPI0, CS0) at 0x0.
2. **SPL** (secondary program loader) initialises DDR3, loads `u-boot`.
3. **U-Boot** loads Linux kernel (FIT image) from SPI NOR.
4. **Kernel** boots with device tree `sun8iw21p1-f900x-v2.2.dtb`.

---

## 2. Display & MIPI-DSI Subsystem

### 2.1 Panel Information

| Parameter | Value |
|-----------|-------|
| **Panel model** | BOE wtl1026boe2l (10.26-inch bar display) |
| **Resolution** | 1600 × 600 pixels |
| **Technology** | IPS, 24-bit colour |
| **Interface** | 4-lane MIPI-DSI (D-PHY v1.1) |
| **Pixel clock (DCLK)** | 53.0 MHz |
| **Horizontal timing** | 1600 active + 160 HBP + 40 HFP + 40 HSYNC = 1840 total |
| **Vertical timing** | 600 active + 20 VBP + 10 VFP + 10 VSYNC = 640 total |
| **Frame rate** | 53.0 MHz / (1840 × 640) ≈ 45 Hz |
| **Backlight** | PWM9 (GPIO bank PB9) @ 50 kHz, active-high |
| **Framebuffer** | RGB565 (default) or ARGB8888 (via kernel config) |

### 2.2 MIPI-DSI Configuration (DTS snippet)

```dts
&dsi {
    pinctrl-names = "default";
    pinctrl-0 = <&dsi_pins>;
    status = "okay";

    panel@0 {
        compatible = "boe,wtl1026boe2l";
        reg = <0>;
        dsi,format = <MIPI_DSI_FMT_RGB565>;
        dsi,lanes = <4>;
        dsi,hs-clk = <53000000>;
        dsi,burst-mode = <DSI_BURST_MODE>;
        dsi,hsa-len = <40>;
        dsi,hbp-len = <160>;
        dsi,hfp-len = <40>;
        dsi,vsa-len = <10>;
        dsi,vbp-len = <20>;
        dsi,vfp-len = <10>;
        width-mm = <260>;
        height-mm = <98>;
        backlight = <&pwm9>;
    };
};

&pwm9 {
    pinctrl-names = "default";
    pinctrl-0 = <&pwm9_pin>;
    status = "okay";
    clock-frequency = <50000>;
    period-ns = <20000>;   /* 50 kHz */
    duty-ns = <10000>;     /* 50% default */
};
```

### 2.3 Framebuffer Configuration

- **Default depth**: RGB565 (16 bpp) → framebuffer size = 1600 × 600 × 2 = 1,920,000 bytes.
- **Optional**: ARGB8888 (32 bpp) → 3,840,000 bytes (requires contiguous DMA memory).
- **Virtual resolution** may be extended for double-buffering (e.g., 1600 × 1200).

---

## 3. Touchscreen

### 3.1 Goodix GT911

| Parameter | Value |
|-----------|-------|
| **Controller** | Goodix GT911 (capacitive touch) |
| **Interface** | I2C1 (SCL: PB0, SDA: PB1) |
| **Interrupt pin** | PB2 (GPIO, rising edge) |
| **Reset pin** | PB3 (GPIO, active-low) |
| **Max touch points** | 5 |
| **Firmware** | Built-in, updated via I2C |

### 3.2 DTS Snippet

```dts
&i2c1 {
    pinctrl-names = "default";
    pinctrl-0 = <&i2c1_pins>;
    status = "okay";
    clock-frequency = <400000>;

    goodix_gt911@5d {
        compatible = "goodix,gt911";
        reg = <0x5d>;
        interrupt-parent = <&pio>;
        interrupts = <2 0>;          /* PB2, IRQ_TYPE_EDGE_RISING */
        reset-gpios = <&pio 3 0>;    /* PB3, active low */
        touchscreen-size-x = <1600>;
        touchscreen-size-y = <600>;
        touchscreen-max-fingers = <5>;
        irq-gpios = <&pio 2 0>;      /* PB2 */
    };
};
```

---

## 4. Multimedia

> **Software status:** the mainline 6.12 board DTS does not yet enable a camera (CSI)
> driver (`CONFIG_MEDIA_CAMERA_SUPPORT` is unset), so `/dev/video0` only exists on the
> factory Linux 4.9 kernel. On mainline, audio is the **DMIC** sound card (`sound_dmic`);
> the analog codec / I2S path below is the factory configuration and has no mainline
> application yet.

### 4.1 Camera (MIPI CSI)

| Parameter | Value |
|-----------|-------|
| **Interface** | 2-lane MIPI CSI (D-PHY) |
| **Sensor** | Onboard 2 MP (e.g., GC2053 / OV2735) |
| **Device node** | `/dev/video0` (V4L2) |
| **Max resolution** | 1920 × 1080 @ 30 fps |
| **Pixel format** | UYVY / YUYV / RAW10 |
| **ISP** | Built-in (Allwinner ISP) |

### 4.2 Audio

| Feature | Specification |
|---------|---------------|
| **Codec** | On-chip I2S + DAC/ADC (24-bit, 48 kHz) |
| **Microphone** | Single analog mic (bias via MICBIAS) |
| **Speaker** | None (line-out for external amp) |
| **Device node** | `snd-soc-sun8iw21` (ALSA) |

### 4.3 DTS Snippet (Audio)

```dts
&i2s0 {
    pinctrl-names = "default";
    pinctrl-0 = <&i2s0_pins>;
    status = "okay";
};

&codec {
    status = "okay";
    mic-det-gpios = <&pio 4 0>;   /* PE0, optional */
};
```

---

## 5. Interfaces & Pinouts

### 5.1 USB Type-C OTG

| Pin | Function | Notes |
|-----|----------|-------|
| USB_DM | USB D- (480 Mbps) | Differential pair |
| USB_DP | USB D+ | |
| VBUS | 5 V input (OTG) | Overvoltage protection |
| CC1/CC2 | Type-C configuration | Pull-down for host, pull-up for device |
| ID | OTG ID (GND for host) | Connected to FEL button (see below) |

- **FEL button**: Connected to USB ID pin. When pressed, ID is grounded → forces host mode and enters FEL (boot ROM download) mode.
- **Power/Reset keys**: Two buttons: `POWER` (PB8) and `RESET` (PB9). `POWER` triggers PMIC shutdown sequence; `RESET` forces SoC reset.

### 5.2 UART0 Debug Testpoints

| Testpoint | Signal | Voltage | Baud rate | Format |
|-----------|--------|---------|-----------|--------|
| **TX0** | UART0 TX (PB6) | 3.3V LVTTL | 115200 | 8N1 |
| **RX0** | UART0 RX (PB7) | 3.3V LVTTL | 115200 | 8N1 |
| **GND** | Ground | 0V | – | – |

**Location**: Edge connector J1 (pins 1–3).  
**Pinout diagram** (top view of board):

```
J1 (UART0)
┌─────────┐
│ 1 TX0   │
│ 2 RX0   │
│ 3 GND   │
└─────────┘
```

### 5.3 Additional GPIOs

| Pin | Function | Notes |
|-----|----------|-------|
| PB0 | I2C1_SCL | Touchscreen |
| PB1 | I2C1_SDA | Touchscreen |
| PB2 | GT911_INT | Touch IRQ |
| PB3 | GT911_RST | Touch reset |
| PB6 | UART0_TX | Debug |
| PB7 | UART0_RX | Debug |
| PB8 | POWER_KEY | PMIC wake |
| PB9 | PWM9 | Backlight |
| PE0 | MIC_DET | Microphone detect |

---

## 6. Power Management & Thermal Characteristics

### 6.1 Power Rails

| Rail | Voltage | Current (max) | Domain |
|------|---------|---------------|--------|
| VCC_CORE | 1.1 V | 1.5 A | Cortex-A7, NPU, DDR3 |
| VCC_PLL | 1.8 V | 0.3 A | PLLs, MIPI, USB |
| VCC_IO | 3.3 V | 0.5 A | GPIO, SPI, I2C, UART |
| VCC_DDR | 1.35 V | 0.8 A | DDR3 VDD |
| VCC_DRAM | 0.675 V | 0.2 A | DDR3 VTT |
| VCC_USB | 5.0 V | 1.0 A | USB VBUS (OTG) |
| VCC_PANEL | 3.3 V | 0.4 A | Display interface |
| VCC_BL | 12.0 V | 0.2 A | Backlight LED string |

### 6.2 PMIC

- **AXP717** (or compatible) – integrated buck/boost regulators.
- **Default sequencing**: VCC_CORE → VCC_PLL → VCC_IO → VCC_DDR → VCC_PANEL.
- **Suspend**: CPU can enter WFI (Wait For Interrupt); NPU and RISC-V can be power-gated.

### 6.3 Thermal Characteristics

| Parameter | Value |
|-----------|-------|
| **SoC Tj (max)** | 85 °C (commercial grade) |
| **Board ambient** | –20 °C to +60 °C (dashcam environment) |
| **Cooling** | Passive (copper plane on PCB, no fan) |
| **Thermal sensor** | Internal SoC diode (read via `thermal` sysfs) |
| **Throttling** | CPU frequency drops to 600 MHz at 75 °C; NPU disabled at 80 °C |

### 6.4 Thermal DTS Example

```dts
thermal-zones {
    soc-thermal {
        polling-delay-passive = <1000>;
        polling-delay = <5000>;
        trips {
            cpu-alert {
                temperature = <75000>;
                hysteresis = <5000>;
                type = "passive";
            };
            cpu-crit {
                temperature = <85000>;
                hysteresis = <5000>;
                type = "critical";
            };
        };
        cooling-maps {
            map0 {
                trip = <&cpu_alert>;
                cooling-device = <&cpu 0 1>; /* reduce freq */
            };
        };
    };
};
```

---

## Appendix: Register Map (Key SoC Registers)

| Register | Address | Description |
|----------|---------|-------------|
| `PLL_VIDEO` | 0x01C20000 | Video PLL (DSI clock source) |
| `DSI_BASE` | 0x01C21000 | MIPI-DSI controller |
| `PWM9` | 0x01C22000 + 0x20 | Backlight PWM control |
| `I2C1` | 0x01C2B000 | Touch I2C bus |
| `UART0` | 0x01C28000 | Debug UART |
| `SPI0` | 0x01C05000 | NOR flash interface |
| `CSI0` | 0x01C0E000 | Camera CSI |
| `NPU` | 0x01C40000 | VIP9000 NPU engine |
| `E907` | 0x01C50000 | RISC-V co-processor |

---

*Document version: 1.0 – Generated for board f900x-v2.2 (km2026240602026.01.03jdh).*  
*Author: Hardware Engineering Team, Allwinner Technology.*
