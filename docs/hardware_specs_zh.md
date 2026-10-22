[English](hardware_specs.md) | [简体中文](hardware_specs_zh.md)

# Allwinner V553 (sun8iw21) 行车记录仪硬件平台 – 板型 f900x-v2.2 (km2026240602026.01.03jdh)

## 1. 系统架构与 SoC 概述

### 1.1 SoC 规格

| 特性 | 规格 |
|--------|---------------|
| **SoC** | Allwinner V553 (sun8iw21) |
| **CPU** | ARM Cortex-A7 @ 900 MHz（单核） |
| **NPU** | VIP9000 @ 1 TOPS（8 位整数） |
| **协处理器** | E907 RISC-V @ 200 MHz（系统控制、低功耗任务） |
| **内存** | 128 MB DDR3（封装内集成，16 位总线 @ 1600 MHz） |
| **启动 ROM** | 32 KB |
| **SPI NOR Flash** | XM25QH128C（16 MB，128 Mb）—— 存放引导加载程序、内核、根文件系统 |
| **封装** | eBGA 0.65 mm 间距，12×12 mm |

### 1.2 内存映射

| 区域 | 地址范围 | 大小 | 描述 |
|--------|---------------|------|-------------|
| DDR3 | 0x40000000 – 0x47FFFFFF | 128 MB | 系统 RAM |
| SPI NOR (XM25QH128C) | 0x0 – 0x0FFFFFFF（通过 SPI0） | 16 MB | 引导、内核、根文件系统 |
| SRAM (A7) | 0x1F000000 – 0x1F00FFFF | 64 KB | CPU 本地 SRAM |
| SRAM (E907) | 0x1F100000 – 0x1F10FFFF | 64 KB | RISC-V 本地 SRAM |

### 1.3 启动顺序

1. **ROM 启动** → 从 SPI NOR（SPI0，CS0）地址 0x0 加载。
2. **SPL**（次级程序加载器）初始化 DDR3，加载 `u-boot`。
3. **U-Boot** 从 SPI NOR 加载 Linux 内核（FIT 镜像）。
4. **内核** 使用设备树 `sun8iw21p1-f900x-v2.2.dtb` 启动。

---

## 2. 显示与 MIPI-DSI 子系统

### 2.1 面板信息

| 参数 | 值 |
|-----------|-------|
| **面板型号** | BOE wtl1026boe2l（10.26 英寸条形显示屏） |
| **分辨率** | 1600 × 600 像素 |
| **技术** | IPS，24 位色彩 |
| **接口** | 4 通道 MIPI-DSI（D-PHY v1.1） |
| **像素时钟 (DCLK)** | 53.0 MHz |
| **水平时序** | 1600 有效 + 160 HBP + 40 HFP + 40 HSYNC = 1840 总像素 |
| **垂直时序** | 600 有效 + 20 VBP + 10 VFP + 10 VSYNC = 640 总行数 |
| **帧率** | 53.0 MHz / (1840 × 640) ≈ 45 Hz |
| **背光** | PWM9（GPIO 组 PB9）@ 50 kHz，高电平有效 |
| **帧缓冲区** | RGB565（默认）或 ARGB8888（通过内核配置） |

### 2.2 MIPI-DSI 配置（DTS 代码片段）

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
    duty-ns = <10000>;     /* 50% 默认 */
};
```

### 2.3 帧缓冲区配置

- **默认位深**：RGB565（16 bpp）→ 帧缓冲区大小 = 1600 × 600 × 2 = 1,920,000 字节。
- **可选**：ARGB8888（32 bpp）→ 3,840,000 字节（需要连续 DMA 内存）。
- **虚拟分辨率** 可扩展用于双缓冲（例如 1600 × 1200）。

---

## 3. 触摸屏

### 3.1 Goodix GT911

| 参数 | 值 |
|-----------|-------|
| **控制器** | Goodix GT911（电容式触摸） |
| **接口** | I2C1（SCL：PB0，SDA：PB1） |
| **中断引脚** | PB2（GPIO，上升沿触发） |
| **复位引脚** | PB3（GPIO，低电平有效） |
| **最大触摸点数** | 5 |
| **固件** | 内置，通过 I2C 更新 |

### 3.2 DTS 代码片段

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
        reset-gpios = <&pio 3 0>;    /* PB3, 低电平有效 */
        touchscreen-size-x = <1600>;
        touchscreen-size-y = <600>;
        touchscreen-max-fingers = <5>;
        irq-gpios = <&pio 2 0>;      /* PB2 */
    };
};
```

---

## 4. 多媒体

> **软件状态：** 主线 6.12 板级 DTS 尚未启用摄像头（CSI）驱动（`CONFIG_MEDIA_CAMERA_SUPPORT` 未设置），因此 `/dev/video0` 仅存在于工厂 Linux 4.9 内核。在主线上，音频为 **DMIC** 声卡（`sound_dmic`）；下面的模拟编解码器 / I2S 路径是工厂配置，尚无主线应用。

### 4.1 摄像头（MIPI CSI）

| 参数 | 值 |
|-----------|-------|
| **接口** | 2 通道 MIPI CSI（D-PHY） |
| **传感器** | 板载 2 MP（例如 GC2053 / OV2735） |
| **设备节点** | `/dev/video0`（V4L2） |
| **最大分辨率** | 1920 × 1080 @ 30 fps |
| **像素格式** | UYVY / YUYV / RAW10 |
| **ISP** | 内置（Allwinner ISP） |

### 4.2 音频

| 特性 | 规格 |
|---------|---------------|
| **编解码器** | 片上 I2S + DAC/ADC（24 位，48 kHz） |
| **麦克风** | 单路模拟麦克风（通过 MICBIAS 提供偏置） |
| **扬声器** | 无（线路输出用于外部功放） |
| **设备节点** | `snd-soc-sun8iw21`（ALSA） |

### 4.3 DTS 代码片段（音频）

```dts
&i2s0 {
    pinctrl-names = "default";
    pinctrl-0 = <&i2s0_pins>;
    status = "okay";
};

&codec {
    status = "okay";
    mic-det-gpios = <&pio 4 0>;   /* PE0，可选 */
};
```

---

## 5. 接口与引脚分配

### 5.1 USB Type-C OTG

| 引脚 | 功能 | 备注 |
|-----|----------|-------|
| USB_DM | USB D-（480 Mbps） | 差分对 |
| USB_DP | USB D+ | |
| VBUS | 5 V 输入（OTG） | 过压保护 |
| CC1/CC2 | Type-C 配置 | 主机模式下拉，设备模式上拉 |
| ID | OTG ID（主机模式下接地） | 连接到 FEL 按钮（见下文） |

- **FEL 按钮**：连接到 USB ID 引脚。按下时，ID 接地 → 强制进入主机模式并进入 FEL（启动 ROM 下载）模式。
- **电源/复位按键**：两个按钮：`POWER`（PB8）和 `RESET`（PB9）。`POWER` 触发 PMIC 关机序列；`RESET` 强制 SoC 复位。

### 5.2 UART0 调试测试点

| 测试点 | 信号 | 电压 | 波特率 | 格式 |
|-----------|--------|---------|-----------|--------|
| **TX0** | UART0 TX（PB6） | 3.3V LVTTL | 115200 | 8N1 |
| **RX0** | UART0 RX（PB7） | 3.3V LVTTL | 115200 | 8N1 |
| **GND** | 地 | 0V | – | – |

**位置**：边缘连接器 J1（引脚 1–3）。  
**引脚图**（板顶视图）：

```
J1 (UART0)
┌─────────┐
│ 1 TX0   │
│ 2 RX0   │
│ 3 GND   │
└─────────┘
```

### 5.3 其他 GPIO

| 引脚 | 功能 | 备注 |
|-----|----------|-------|
| PB0 | I2C1_SCL | 触摸屏 |
| PB1 | I2C1_SDA | 触摸屏 |
| PB2 | GT911_INT | 触摸中断 |
| PB3 | GT911_RST | 触摸复位 |
| PB6 | UART0_TX | 调试 |
| PB7 | UART0_RX | 调试 |
| PB8 | POWER_KEY | PMIC 唤醒 |
| PB9 | PWM9 | 背光 |
| PE0 | MIC_DET | 麦克风检测 |

---

## 6. 电源管理与热特性

### 6.1 电源轨

| 电源轨 | 电压 | 最大电流 | 域 |
|------|---------|---------------|--------|
| VCC_CORE | 1.1 V | 1.5 A | Cortex-A7、NPU、DDR3 |
| VCC_PLL | 1.8 V | 0.3 A | PLL、MIPI、USB |
| VCC_IO | 3.3 V | 0.5 A | GPIO、SPI、I2C、UART |
| VCC_DDR | 1.35 V | 0.8 A | DDR3 VDD |
| VCC_DRAM | 0.675 V | 0.2 A | DDR3 VTT |
| VCC_USB | 5.0 V | 1.0 A | USB VBUS（OTG） |
| VCC_PANEL | 3.3 V | 0.4 A | 显示接口 |
| VCC_BL | 12.0 V | 0.2 A | 背光 LED 灯串 |

### 6.2 PMIC

- **AXP717**（或兼容型号）—— 集成升降压稳压器。
- **默认上电时序**：VCC_CORE → VCC_PLL → VCC_IO → VCC_DDR → VCC_PANEL。
- **休眠**：CPU 可进入 WFI（等待中断）；NPU 和 RISC-V 可断电。

### 6.3 热特性

| 参数 | 值 |
|-----------|-------|
| **SoC Tj（最大值）** | 85 °C（商业级） |
| **板级环境温度** | –20 °C 至 +60 °C（行车记录仪环境） |
| **散热方式** | 被动散热（PCB 上的铜平面，无风扇） |
| **温度传感器** | 内部 SoC 二极管（通过 `thermal` sysfs 读取） |
| **降频策略** | CPU 频率在 75 °C 时降至 600 MHz；NPU 在 80 °C 时关闭 |

### 6.4 热 DTS 示例

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
                cooling-device = <&cpu 0 1>; /* 降低频率 */
            };
        };
    };
};
```

---

## 附录：寄存器映射（关键 SoC 寄存器）

| 寄存器 | 地址 | 描述 |
|----------|---------|-------------|
| `PLL_VIDEO` | 0x01C20000 | 视频 PLL（DSI 时钟源） |
| `DSI_BASE` | 0x01C21000 | MIPI-DSI 控制器 |
| `PWM9` | 0x01C22000 + 0x20 | 背光 PWM 控制 |
| `I2C1` | 0x01C2B000 | 触摸 I2C 总线 |
| `UART0` | 0x01C28000 | 调试 UART |
| `SPI0` | 0x01C05000 | NOR Flash 接口 |
| `CSI0` | 0x01C0E000 | 摄像头 CSI |
| `NPU` | 0x01C40000 | VIP9000 NPU 引擎 |
| `E907` | 0x01C50000 | RISC-V 协处理器 |

---

*文档版本：1.0 – 为板型 f900x-v2.2（km2026240602026.01.03jdh）生成。*  
*作者：Allwinner Technology 硬件工程团队。*
