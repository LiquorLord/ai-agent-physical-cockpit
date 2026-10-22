# 全志 V553 (sun8iw21) 主线 Linux 内核移植路线图

[English](mainline_linux_roadmap.md) | [简体中文](mainline_linux_roadmap_zh.md)

**文档版本:** 1.0  
**适用内核:** Linux 6.6 LTS / 6.12+  
**目标平台:** Allwinner V553 (sun8iw21) – Cortex-A7 @ 900MHz + E907 RISC-V + VIP9000 NPU  
**作者:** Linux Kernel Maintainer / Board Bringup Expert  

---

## 1. 架构与芯片标识 / Architecture & Silicon Identity

Allwinner V553 与 V853 共享相同的 die (sun8iw21)，是一颗 **三合一异构 SoC**：

- **ARM Cortex-A7** @ 900 MHz – 主应用处理器
- **E907 RISC-V** @ 300 MHz – 实时协处理器
- **VIP9000 NPU** – 神经网络加速器

| 特性 | V553 / V853 |
|------|-------------|
| 封装 | BGA 169 |
| 内存接口 | DDR3/DDR3L @ 800 MHz |
| 显示引擎 | DE2.0 (Display Engine 2.0) |
| 视频编解码 | H.264/H.265 编码/解码 |
| 外设 | USB2.0, SDIO, SPI, I2C, UART, MIPI-DSI, RGB |

> **注意:** V553 主要面向行车记录仪 (Dashcam) 市场，通常搭配 10.26 英寸 1600×600 显示屏。V853 则更偏向 AI 摄像头。两者软件完全兼容。

---

## 2. 上游 Linux 内核支持现状 / Current Upstream Status

Linux 主线内核（6.6 LTS / 6.12）已包含以下关键驱动：

### 2.1 设备树基础
- **`arch/arm/boot/dts/allwinner/sun8i-v853.dtsi`** – 基础 SoC 设备树
- 包含 CPU、定时器、PLL、GPIO、MMC、USB、SPI、I2C、UART 等节点

### 2.2 时钟控制器
- **`drivers/clk/sunxi-ng/ccu-sun8i-v853.c`** – 完整时钟树
- 支持 PLL、AHB、APB、DRAM、DE、DSI 等时钟

### 2.3 引脚控制器
- **`drivers/pinctrl/sunxi/pinctrl-sun8i-v853.c`** – 所有 GPIO 功能映射

### 2.4 MMC 驱动
- 使用 `sunxi-mmc` 驱动，已在 sun8i-v853.dtsi 中启用 SD/MMC 控制器

### 2.5 其他驱动状态
| 驱动 | 状态 | 备注 |
|------|------|------|
| USB EHCI/OHCI | ✅ 主线 | 需配置 PHY |
| SPI | ✅ 主线 | 支持 DMA |
| I2C | ✅ 主线 | 支持多路复用 |
| PWM | ✅ 主线 | 用于背光 |
| Watchdog | ✅ 主线 | 支持复位 |
| Thermal | ✅ 主线 | 支持温度传感器 |

### 2.6 尚未主线化的模块
- **Display Engine 2.0 (DE2.0)** – 仅部分支持 (sun4i-drm 驱动)
- **MIPI-DSI** – 需要移植 panel 参数
- **E907 RISC-V** – 无主线 remoteproc 驱动
- **VIP9000 NPU** – 无主线驱动 (需 vendor 方案)

---

## 3. 轻量级现代引导程序 / Lightweight Modern Bootloader

推荐使用 **SyterKit** 或 **awboot** 从 MicroSD 卡引导主线 Linux。

### 3.1 SyterKit
- 仓库: [https://github.com/sytel/SyterKit](https://github.com/sytel/SyterKit)
- 支持 sun8iw21 平台
- 可直接从 FAT32 分区加载 `Image` 和 `sun8i-v553-*.dtb`
- 无需 U-Boot SPL，体积 < 64KB

### 3.2 awboot
- 仓库: [https://github.com/awboot/awboot](https://github.com/awboot/awboot)
- 专为 Allwinner 设计，支持 FEL 和 SD 卡启动
- 可配置 `bootargs` 和 `fdtfile`

### 3.3 推荐引导流程
```
SD 卡布局:
- 分区1: FAT32 (存放 Image, dtb, boot.scr)
- 分区2: ext4 (rootfs)

引导步骤:
1. BROM 从 SD 卡加载 SyterKit/awboot
2. 引导程序读取 boot.scr 或自动加载 Image + dtb
3. 跳转到 Linux 内核
```

---

## 4. 显示引擎 DE2.0 与 MIPI-DSI 移植路线图 / Display Engine DE2.0 & MIPI-DSI Porting Roadmap

### 4.1 背景
V553 行车记录仪通常使用 **BOE 10.26-inch wtl1026boe2l** 面板：
- 分辨率: 1600×600
- DCLK: 53 MHz
- 数据通道: 4-lane MIPI-DSI
- 供应商驱动基于 Linux 4.9 `disp2` 框架

### 4.2 主线化目标
将面板参数从 vendor `disp2` 移植到主线 `sun4i-drm` 的 `panel-simple` 或自定义 panel 驱动。

### 4.3 关键参数映射

| Vendor `disp2` 参数 | 主线 `panel-simple` 对应 | 值 |
|---------------------|--------------------------|----|
| `lcd_dclk_freq` | `panel-timing.timings.clock` | 53 MHz |
| `lcd_x` / `lcd_y` | `width-mm` / `height-mm` | 1600×600 |
| `lcd_hbp` / `lcd_hspw` | `hactive` / `hfront-porch` / `hsync-len` | 根据 datasheet |
| `lcd_vbp` / `lcd_vspw` | `vactive` / `vfront-porch` / `vsync-len` | 根据 datasheet |
| `lcd_dsi_lane_num` | `dsi-lanes` | 4 |
| `lcd_dsi_clk_division` | `dsi-clk-division` | 通常 1 |
| `lcd_backlight` | `backlight` PWM 节点 | 使用 `pwm-backlight` |

### 4.4 移植步骤

1. **获取面板时序数据** – 从 vendor 4.9 内核的 `board.dts` 或 `lcd_*.c` 中提取 `lcd_timing` 结构体。
2. **创建 panel-simple 兼容条目** – 在 `drivers/gpu/drm/panel/panel-simple.c` 中添加 `wtl1026boe2l` 的 `panel_desc`。
3. **配置 DSI 控制器** – 在 `sun8i-v553.dtsi` 中启用 `dsi0` 节点，设置 `#address-cells`、`#size-cells`，并引用 panel。
4. **调整 DE2.0 时钟** – 确保 `de` 和 `tcon` 时钟频率正确 (通常 297 MHz 或 594 MHz)。
5. **测试** – 使用 `modetest` 或 `weston` 验证显示输出。

### 4.5 示例 DTS 节点
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

## 5. 触摸屏 (Goodix GT911) 与 USB 复合设备 / Touchscreen & USB Composite Gadget

### 5.1 Goodix GT911 触摸屏
- 使用主线 `goodix` 触摸屏驱动 (`drivers/input/touchscreen/goodix.c`)
- 配置 I2C 总线及中断引脚

**DTS 节点示例:**
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
- 行车记录仪通常通过 USB 实现 **UVC (摄像头)** + **UAC (音频)** + **MTP (存储)** 复合设备。
- 主线配置: 使用 `g_webcam` 或自定义 `configfs` 复合 gadget。

**建议方案:**
```bash
# 在 defconfig 中启用
CONFIG_USB_GADGET=y
CONFIG_USB_CONFIGFS=y
CONFIG_USB_CONFIGFS_F_UVC=y
CONFIG_USB_CONFIGFS_F_UAC=y
CONFIG_USB_CONFIGFS_F_MTP=y
```

**用户空间脚本** 通过 `configfs` 创建复合设备:
```bash
mkdir -p /sys/kernel/config/usb_gadget/g1
echo 0x1f3a > /sys/kernel/config/usb_gadget/g1/idVendor
echo 0x1000 > /sys/kernel/config/usb_gadget/g1/idProduct
mkdir -p /sys/kernel/config/usb_gadget/g1/configs/c.1
mkdir -p /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409
echo "Dashcam" > /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409/product
# 添加 UVC, UAC, MTP 功能
```

---

## 6. 开源社区协作计划 / Open-Source Community Collaboration Plan

### 6.1 GitHub 仓库结构
```
repo: allwinner-v553-linux
├── docs/
│   └── mainline_linux_roadmap.md (本文)
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

### 6.2 Docker 构建环境
提供基于 Ubuntu 22.04 的 Dockerfile，预装交叉编译工具链 (ARM GCC 12.x) 和必要依赖。

```dockerfile
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    gcc-arm-linux-gnueabihf \
    libncurses-dev \
    flex bison \
    u-boot-tools \
    device-tree-compiler
```

### 6.3 内核配置 (defconfig)
基于 `sunxi_defconfig` 修改，增加以下选项:
- `CONFIG_DRM_PANEL_SIMPLE=y`
- `CONFIG_INPUT_GOODIX=y`
- `CONFIG_USB_CONFIGFS_F_UVC=y`
- `CONFIG_USB_CONFIGFS_F_UAC=y`
- `CONFIG_USB_CONFIGFS_F_MTP=y`
- `CONFIG_PWM_SUNXI=y`
- `CONFIG_BACKLIGHT_PWM=y`

### 6.4 协作流程
1. **Fork** 官方仓库并创建 feature 分支。
2. 提交 patch 到 `linux-sunxi` 邮件列表或直接提交 PR。
3. 在 GitHub Issues 中讨论未解决的问题（如 E907 remoteproc）。
4. 定期同步主线内核更新，维护补丁栈。

---

## 附录 / Appendix

### A. 参考资源
- [Allwinner V853 数据手册 (公开版)](https://linux-sunxi.org/V853)
- [Linux 主线 sun8i-v853.dtsi](https://github.com/torvalds/linux/blob/master/arch/arm/boot/dts/allwinner/sun8i-v853.dtsi)
- [SyterKit 文档](https://github.com/sytel/SyterKit)
- [awboot 项目](https://github.com/awboot/awboot)

### B. 已知问题
- DE2.0 的 `tcon` 时钟分频在部分面板上需要手动调整。
- GT911 触摸屏中断可能需配置 `edge` 触发。
- USB gadget 复合设备在 UVC 流传输时需注意带宽。

---

**本文档将持续更新，欢迎社区贡献。**  
**Mainline Linux on Allwinner V553 – 让开源的力量驱动行车记录仪的未来。**
