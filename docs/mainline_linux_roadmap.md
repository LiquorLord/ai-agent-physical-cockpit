[English](mainline_linux_roadmap.md) | [简体中文](mainline_linux_roadmap_zh.md)

# Allwinner V553 (sun8iw21) Mainline Linux Kernel Porting Roadmap

**Document Version:** 1.0  
**Applicable Kernel:** Linux 6.6 LTS / 6.12+  
**Target Platform:** Allwinner V553 (sun8iw21) – Cortex-A7 @ 900MHz + E907 RISC-V + VIP9000 NPU  
**Author:** Linux Kernel Maintainer / Board Bringup Expert  

---

## 1. Architecture & Silicon Identity

Allwinner V553 shares the same die (sun8iw21) as V853, forming a **triple-core heterogeneous SoC**:

- **ARM Cortex-A7** @ 900 MHz – Main application processor
- **E907 RISC-V** @ 300 MHz – Real-time co-processor
- **VIP9000 NPU** – Neural network accelerator

| Feature | V553 / V853 |
|---------|-------------|
| Package | BGA 169 |
| Memory Interface | DDR3/DDR3L @ 800 MHz |
| Display Engine | DE2.0 (Display Engine 2.0) |
| Video Codec | H.264/H.265 encode/decode |
| Peripherals | USB2.0, SDIO, SPI, I2C, UART, MIPI-DSI, RGB |

> **Note:** V553 is primarily targeted at the dashcam market, typically paired with a 10.26-inch 1600×600 display. V853 leans more toward AI cameras. The two are fully software-compatible.

---

## 2. Current Upstream Linux Kernel Support Status

The mainline Linux kernel (6.6 LTS / 6.12) already includes the following key drivers:

### 2.1 Device Tree Foundation
- **`arch/arm/boot/dts/allwinner/sun8i-v853.dtsi`** – Base SoC device tree
- Includes nodes for CPU, timers, PLL, GPIO, MMC, USB, SPI, I2C, UART, etc.

### 2.2 Clock Controller
- **`drivers/clk/sunxi-ng/ccu-sun8i-v853.c`** – Complete clock tree
- Supports PLL, AHB, APB, DRAM, DE, DSI clocks

### 2.3 Pin Controller
- **`drivers/pinctrl/sunxi/pinctrl-sun8i-v853.c`** – All GPIO function mappings

### 2.4 MMC Driver
- Uses the `sunxi-mmc` driver, SD/MMC controllers enabled in sun8i-v853.dtsi

### 2.5 Other Driver Status
| Driver | Status | Notes |
|--------|--------|-------|
| USB EHCI/OHCI | ✅ Mainline | Requires PHY configuration |
| SPI | ✅ Mainline | Supports DMA |
| I2C | ✅ Mainline | Supports multiplexing |
| PWM | ✅ Mainline | Used for backlight |
| Watchdog | ✅ Mainline | Supports reset |
| Thermal | ✅ Mainline | Supports temperature sensor |

### 2.6 Modules Not Yet Mainlined
- **Display Engine 2.0 (DE2.0)** – Only partial support (sun4i-drm driver)
- **MIPI-DSI** – Requires panel parameter porting
- **E907 RISC-V** – No mainline remoteproc driver
- **VIP9000 NPU** – No mainline driver (requires vendor solution)

---

## 3. Lightweight Modern Bootloader

It is recommended to use **SyterKit** or **awboot** to boot mainline Linux from a MicroSD card.

### 3.1 SyterKit
- Repository: [https://github.com/sytel/SyterKit](https://github.com/sytel/SyterKit)
- Supports the sun8iw21 platform
- Can directly load `Image` and `sun8i-v553-*.dtb` from a FAT32 partition
- No U-Boot SPL required, size < 64KB

### 3.2 awboot
- Repository: [https://github.com/awboot/awboot](https://github.com/awboot/awboot)
- Designed specifically for Allwinner, supports FEL and SD card booting
- Configurable `bootargs` and `fdtfile`

### 3.3 Recommended Boot Flow
```
SD Card Layout:
- Partition 1: FAT32 (stores Image, dtb, boot.scr)
- Partition 2: ext4 (rootfs)

Boot Steps:
1. BROM loads SyterKit/awboot from SD card
2. Bootloader reads boot.scr or automatically loads Image + dtb
3. Jump to Linux kernel
```

---

## 4. Display Engine DE2.0 & MIPI-DSI Porting Roadmap

### 4.1 Background
V553 dashcams typically use a **BOE 10.26-inch wtl1026boe2l** panel:
- Resolution: 1600×600
- DCLK: 53 MHz
- Data lanes: 4-lane MIPI-DSI
- Vendor driver based on Linux 4.9 `disp2` framework

### 4.2 Mainlining Goal
Port panel parameters from the vendor `disp2` to mainline `sun4i-drm`'s `panel-simple` or a custom panel driver.

### 4.3 Key Parameter Mapping

| Vendor `disp2` Parameter | Mainline `panel-simple` Equivalent | Value |
|--------------------------|------------------------------------|-------|
| `lcd_dclk_freq` | `panel-timing.timings.clock` | 53 MHz |
| `lcd_x` / `lcd_y` | `width-mm` / `height-mm` | 1600×600 |
| `lcd_hbp` / `lcd_hspw` | `hactive` / `hfront-porch` / `hsync-len` | Per datasheet |
| `lcd_vbp` / `lcd_vspw` | `vactive` / `vfront-porch` / `vsync-len` | Per datasheet |
| `lcd_dsi_lane_num` | `dsi-lanes` | 4 |
| `lcd_dsi_clk_division` | `dsi-clk-division` | Typically 1 |
| `lcd_backlight` | `backlight` PWM node | Use `pwm-backlight` |

### 4.4 Porting Steps

1. **Obtain panel timing data** – Extract the `lcd_timing` structure from the vendor 4.9 kernel's `board.dts` or `lcd_*.c`.
2. **Create a panel-simple compatible entry** – Add a `panel_desc` for `wtl1026boe2l` in `drivers/gpu/drm/panel/panel-simple.c`.
3. **Configure the DSI controller** – Enable the `dsi0` node in `sun8i-v553.dtsi`, set `#address-cells`, `#size-cells`, and reference the panel.
4. **Adjust DE2.0 clocks** – Ensure `de` and `tcon` clock frequencies are correct (typically 297 MHz or 594 MHz).
5. **Test** – Verify display output using `modetest` or `weston`.

### 4.5 Example DTS Node
```dts
&dsi0 {
    #address-cells = <1>;
    #size-cells = <0>;
    pinctrl-names = "default";
    pinctrl-0 = <&dsi0_pins>;
    status = "okay";

    panel@0 {
        compatible = "boe,wtl1026boe2l";
        reg = <0>;
        dsi-lanes = <4>;
        power-supply = <®_ldo1>;
        backlight = <&backlight>;
        reset-gpios = <&pio 7 6 GPIO_ACTIVE_LOW>; /* PH6 */
        status = "okay";
    };
};
```

---

## 5. Touchscreen (Goodix GT911) & USB Composite Gadget

### 5.1 Goodix GT911 Touchscreen
- Uses the mainline `goodix` touchscreen driver (`drivers/input/touchscreen/goodix.c`)
- Configure I2C bus and interrupt pin

**DTS Node Example:**
```dts
&i2c0 {
    status = "okay";
    pinctrl-names = "default";
    pinctrl-0 = <&i2c0_pins>;

    goodix@5d {
        compatible = "goodix,gt911";
        reg = <0x5d>;
        interrupt-parent = <&pio>;
        interrupts = <7 10 IRQ_TYPE_EDGE_FALLING>; /* PH10 */
        touchscreen-size-x = <1600>;
        touchscreen-size-y = <600>;
        touchscreen-swapped-x-y;
        status = "okay";
    };
};
```

### 5.2 USB Composite Gadget
- Dashcams typically implement a **UVC (camera) + UAC (audio) + MTP (storage)** composite device over USB.
- Mainline configuration: Use `g_webcam` or custom `configfs` composite gadget.

**Recommended Setup:**
```bash
# Enable in defconfig
CONFIG_USB_GADGET=y
CONFIG_USB_CONFIGFS=y
CONFIG_USB_CONFIGFS_F_UVC=y
CONFIG_USB_CONFIGFS_F_UAC=y
CONFIG_USB_CONFIGFS_F_MTP=y
```

**Userspace script** to create the composite device via `configfs`:
```bash
mkdir -p /sys/kernel/config/usb_gadget/g1
echo 0x1f3a > /sys/kernel/config/usb_gadget/g1/idVendor
echo 0x1000 > /sys/kernel/config/usb_gadget/g1/idProduct
mkdir -p /sys/kernel/config/usb_gadget/g1/configs/c.1
mkdir -p /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409
echo "Dashcam" > /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409/product
# Add UVC, UAC, MTP functions
```

---

## 6. Open-Source Community Collaboration Plan

### 6.1 GitHub Repository Structure
```
repo: allwinner-v553-linux
├── docs/
│   └── mainline_linux_roadmap.md (this document)
├── kernel/
│   ├── configs/
│   │   ├── sun8i-v553_defconfig
│   │   └── sun8i-v553_debug_defconfig
│   └── patches/
│       ├── 0001-drm-panel-add-wtl1026boe2l.patch
│       └── 0002-dts-sun8i-v553-add-panel-and-touch.patch
├── bootloader/
│   ├── syterkit/
│   └── awboot/
├── scripts/
│   ├── build_kernel.sh
│   └── make_sd_card.sh
└── README.md
```

### 6.2 Docker Build Environment
Provide a Dockerfile based on Ubuntu 22.04, pre-installing the cross-compilation toolchain (ARM GCC 12.x) and necessary dependencies.

```dockerfile
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    gcc-arm-linux-gnueabihf \
    libncurses-dev \
    flex bison \
    u-boot-tools \
    device-tree-compiler
```

### 6.3 Kernel Configuration (defconfig)
Based on `sunxi_defconfig`, add the following options:
- `CONFIG_DRM_PANEL_SIMPLE=y`
- `CONFIG_INPUT_GOODIX=y`
- `CONFIG_USB_CONFIGFS_F_UVC=y`
- `CONFIG_USB_CONFIGFS_F_UAC=y`
- `CONFIG_USB_CONFIGFS_F_MTP=y`
- `CONFIG_PWM_SUNXI=y`
- `CONFIG_BACKLIGHT_PWM=y`

### 6.4 Collaboration Workflow
1. **Fork** the official repository and create a feature branch.
2. Submit patches to the `linux-sunxi` mailing list or directly submit PRs.
3. Discuss unresolved issues (e.g., E907 remoteproc) in GitHub Issues.
4. Regularly sync with mainline kernel updates and maintain the patch stack.

---

## Appendix

### A. Reference Resources
- [Allwinner V853 Datasheet (Public)](https://linux-sunxi.org/V853)
- [Linux Mainline sun8i-v853.dtsi](https://github.com/torvalds/linux/blob/master/arch/arm/boot/dts/allwinner/sun8i-v853.dtsi)
- [SyterKit Documentation](https://github.com/sytel/SyterKit)
- [awboot Project](https://github.com/awboot/awboot)

### B. Known Issues
- The `tcon` clock divider for DE2.0 may need manual adjustment on some panels.
- The GT911 touchscreen interrupt may require `edge` trigger configuration.
- USB gadget composite devices need attention to bandwidth during UVC streaming.

---

**This document will be continuously updated. Community contributions are welcome.**  
**Mainline Linux on Allwinner V553 – Let the power of open source drive dashcams.**
