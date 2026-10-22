# 全志 V553 (sun8iw21) 系统架构与硬件底层规范参考手册

[English](V553_SYSTEM_REFERENCE.md) | [简体中文](V553_SYSTEM_REFERENCE_zh.md)

---

## 1. 核心芯片与硬件平台画像

| 硬件维度 | 参数规范 | 工程物理实测与说明 |
| :--- | :--- | :--- |
| **目标设备** | 10.26 英寸智能触屏行车记录仪 | 硬件主板型号：`f900x-v2.2 km2026240602026.01.03jdh` |
| **SoC 主控** | 全志 V553 (`sun8iw21p1`) | 与 V853 同晶圆；异构三核高集成度 SoC |
| **应用主 CPU** | ARM Cortex-A7 @ 900 MHz (单核) | ARMv7-A 架构，支持 VFPv4 与 NEON，32KB L1 I/D，128KB L2 缓存 |
| **实时协处理器** | 平头哥玄铁 E907 RISC-V @ 300 MHz | 独立片内 SRAM `0x1F100000`，核间 Mailbox 通信 |
| **AI 神经网络单元** | VIP9000 @ 1.0 TOPS (INT8) | 嵌入式机器视觉、车道偏离预警与手势识别加速 |
| **系统内存 (RAM)** | 128 MB DDR3 SIP (封装内集成) | 16 位位宽 @ 648 MHz（物理地址空间 `0x40000000 ~ 0x47FFFFFF`） |
| **板载主存储** | 16 MB SPI NOR Flash (`XM25QH128C`) | JEDEC ID `0x204018`，SPI0 片选 CS0 |
| **外部扩展存储** | MicroSD 卡槽 (MMC0 控制器) | 物理管脚 Port F (`PF0..PF5`)，高速 3.3V 模式 |
| **电源管理单元** | 片上集成式内置 PMU (`allwinner,v853-pmu`) | 基址 `0x07090000`，**主板上绝无外部 AXP2101 PMIC** |
| **显示面板** | 京东方 BOE 10.26 英寸长条带状屏 (`wtl1026boe2l`) | 物理分辨率 1600 × 600，4-lane MIPI-DSI，PWM9 背光调光 |
| **触摸屏** | 汇顶 Goodix GT911 电容触控 IC | I2C 从机地址 `0x5d` / `0x14`，原生 1600 × 600 分辨率 |
| **音频管线** | 片内集成音频 Codec + D 类功放 | 具备硬件 AVCC 自动校准，板载驻极体麦克风与扬声器 |
| **摄像头接口** | 双路 ISP 硬件视频接入通道 | 前摄：MIPI-CSI (GC2053 / SC2336)；后摄：TP9950 模拟 AHD 转换芯片 |
| **无线通信模组** | 翱捷/爱科微 AltoBeam ATBM6165 | 802.11 b/g/n 单频 Wi-Fi (SDIO) + 蓝牙 4.2 / BLE (UART) |
| **USB 物理接口** | USB 2.0 High-Speed OTG (480 Mbps) | Type-C 物理接口，底层 UDC 控制器 `4100000.udc-controller` |
| **调试串口** | 板载专用圆形测试焊盘 (`tx0`, `rx0`, `GND`) | UART0 @ 115200 8N1（引脚为 Port E：`PE9` / `PE10`） |

---

## 2. 供电架构与片上内置 PMU 稳压器实测

### 2.1 PMU 架构第一性原理破案
很多全志开源开发板（如基于全志 V851SE 的 TinyVision）在板上外挂了 AXP2101 或 AXP717 专用 PMIC 芯片，并通过 I2C 总线控制供电。
**但在我们这块 V553 `f900x-v2.2` 行车记录仪上，全志采用了片上内置电源管理架构（PMC/PMU）**：
- 设备树节点：`pmu@07090000`
- 匹配兼容标识：`compatible = "allwinner,v853-pmu", "syscon", "simple-mfd"`
- **关键工程结论**：任何编译包含外部 AXP2101 I2C 寻址的第三方 Boot0（如 TinyVision 的固件），在开机运行第 1 毫秒就会因 I2C 寻址 PMIC 超时（`[pmu]: bus init error`, `unknow PMU`）而无法建立 DDR 供电，进而触发内部保护函数直接调用 `return_to_fel()`，**导致 SoC 每次插卡开机必定黑屏死锁在 USB FEL 救砖模式（`1f3a:efe8`）**！

### 2.2 各供电支路物理电压表
| 电源轨名称 | 设定电压值 | 供电目标与子系统 | 启动策略 |
| :--- | :--- | :--- | :--- |
| `pmc-dcdc1` | 3.30 V (`0x325aa0` uV) | 外设 I/O、板载 SPI NOR Flash、MicroSD 槽供电 | 保持常开 (Always-On) |
| `pmc-dcdc2` | 0.90 V (`0xdbba0` uV) | SoC 数字核心逻辑供电 (`VDD-SYS`, `VDD-CPU`) | 保持常开 (Always-On) |
| `pmc-dcdc3` | 1.50 V (`0x16e360` uV) | DDR3 内存颗粒阵列及 PHY 供电 (`VDD-DRAM`) | 保持常开 (Always-On) |
| `pmc-ldoa` | 1.80 V (`0x1b7740` uV) | MIPI-DSI 屏幕高速差分与 MIPI-CSI 模拟供电 | 保持常开 (Always-On) |
| `pmc-aldo` | 1.80 V (`0x1b7740` uV) | 片内 PLL 锁相环及音频 AVCC 模拟供电 | 保持常开 (Always-On) |
| `pmc-rtcldo`| 1.80 V (`0x1b7740` uV) | RTC 实时时钟与深度睡眠逻辑 | 保持常开 (Always-On) |
| `pmc-ldo33` | 3.30 V (`0x325aa0` uV) | 模拟 3.3V，CMOS 传感器模拟供电 | 保持常开 (Always-On) |
| `pmc-ldo09` | 0.90 V (`0xdbba0` uV) | VIP9000 NPU 神经网络核心及 DSP 供电 | 保持常开 (Always-On) |

---

## 3. DDR3 时序与 Boot0 参数物理布局

### 3.1 内存物理特性
- **类型**：DDR3 SIP（SoC 封装内集成）
- **工作主频**：648 MHz（寄存器值 `0x0288` = 648）
- **数据总线位宽**：16-bit
- **容量**：128 MB（1 Gb）
- **物理地址映射**：`0x40000000 ~ 0x47FFFFFF`

### 3.2 原厂微调校准参数块（Boot0 偏移 `0x0030 ~ 0x0080`，共 80 字节）
以下数据块系从原厂板载 SPI NOR Boot0 中提取的 100% 物理真值：
```text
0x0030: 08 00 00 00 88 02 00 00 03 00 00 00 fb 7b 7b 00
0x0040: 01 00 00 00 d2 10 00 00 00 00 80 00 70 1c 00 00
0x0050: 42 00 00 00 18 00 00 00 00 00 00 00 52 a1 48 00
0x0060: 4d a9 b1 01 4f 20 07 00 96 78 78 b4 00 00 00 00
0x0070: 48 48 48 48 48 00 00 00 1e 12 21 16
```
- 内存工作时钟：`dram_clk = 648 MHz`
- 内存物理类别：`dram_type = 3`（严格指定为 DDR3，不可配置为 DDR2）
- 阻抗校准机制：启用片内自动 ZQ 校准

---

## 4. 存储架构与分区表规范

### 4.1 板载 16MB SPI NOR Flash (`XM25QH128C`)
- 物理扇区总数：4,096 个 4KB 扇区 = 16,777,216 字节。
- MTD 分区表（解析自 U-Boot 环境变量）：
  ```text
  mtdparts=spi0.0:128K(env),2368K(boot),12736K(rootfs),512K(userdata),128K(bootlogo),0K(UDISK)
  ```

| 分区标识 | 字节起始偏移 | 分区物理大小 | 分区格式与实际内容 |
| :--- | :--- | :--- | :--- |
| **SPL / Boot0** | `0x00000000` | 512 KB | 全志 Boot0 二进制（带有 `eGON.BT0` 头与 SPI NOR 驱动） |
| **U-Boot ENV** | `0x00080000` | 128 KB | U-Boot 固化持久化环境变量区 |
| **boot** | `0x000A0000` | 2,368 KB | Linux 内核镜像（`zImage` / FIT 压缩镜像，约 2.3MB） |
| **rootfs** | `0x002F0000` | 12,736 KB | SquashFS 压缩根文件系统（含原厂 Tina 与双系统移交挂钩） |
| **userdata** | `0x00F60000` | 512 KB | JFFS2 可读写用户配置分区 |
| **bootlogo** | `0x00FE0000` | 128 KB | 开机 Logo 图像点阵存储区 |

### 4.2 MicroSD 卡物理架构（MMC0）
- 控制器：MMC0 @ Port F 端口（`PF0..PF5`）
- 物理 GPIO 配置字节：`22 00 00 00 77 00 00 00 00 01 00 34`
- 硬件管脚映射：
  - `PF0`：SD0_D1
  - `PF1`：SD0_D0
  - `PF2`：SD0_CLK
  - `PF3`：SD0_CMD
  - `PF4`：SD0_D3
  - `PF5`：SD0_D2

---

## 5. 显示管线与触控子系统

### 5.1 MIPI-DSI 显示面板（京东方 `wtl1026boe2l`）
- **物理规格**：10.26 英寸超宽长条形液晶屏（显示有效区域：260 mm × 98 mm）
- **物理像素**：1600 × 600
- **D-PHY 通道**：4 条数据差分对（Data Lanes） + 1 条时钟差分对（Clock Lane）
- **像素点时钟 (DCLK)**：53.0 MHz
- **行场扫描物理时序**：
  - 水平行时序：有效显示 1600，HBP 160，HFP 40，HSYNC 40（行周期总计 1840）
  - 垂直场时序：有效扫描 600，VBP 20，VFP 10，VSYNC 10（场周期总计 640）
  - 刷新率计算：$53.0\text{ MHz} / (1840 \times 640) \approx 45.0\text{ Hz}$
- **背光控制通道**：PWM9（管脚为 `PB9` / `PD21`），工作频率 50 kHz（周期 `period_ns = 20000`，占空比 `duty_ns = 10000`）
- **帧缓冲 Framebuffer 规格**：`/dev/fb0`，1600 × 600 RGB565（16 位深度，显存占用 1.92 MB），支持硬件在线 90 度旋转。

### 5.2 电容触控芯片（汇顶 Goodix GT911）
- 通信总线：I2C0 / I2C1
- 原生坐标映射范围：1600 × 600
- 中断触发方式：低电平/下降沿触发
- 复位时序：需在驱动加载时由 GPIO 输出特定复位脉冲配置 I2C 从机地址

---

## 6. USB OTG 复合外设与无线驱动

### 6.1 USB 2.0 控制器特性
- 控制器 IP：全志 Sunxi Musb 高速 OTG 控制器
- UDC 驱动标识：`4100000.udc-controller`
- **当前交付的复合外设（`switch_to_gud.sh`）**：
  - **GUD (通用 USB 显示器)**：厂商 ID `0x1d50`，产品 ID `0x614d`（FunctionFS `ffs.gud`）
  - **USB HID 模拟键盘**：字符设备 `/dev/hidg0`，支持按键扫描码直通注入
- **历史路线（仅调试保留，开机未绑定）**：
  - **ADB (Android 调试桥)**：厂商 ID `0x18d1`，产品 ID `0xd002`（FunctionFS `ffs.adb`）
  - **CDC-NCM (高速虚拟网卡)**：网络接口 `usb0`（设备 IP：`192.168.10.1/24`）
  - V553 的 USB OTG 控制器仅暴露 **4 个端点**，GUD + HID + NCM 无法共存。NCM 流式传输视频比 GUD 更快，但需要主机端软件 —— 因此才改为 GUD + HID 方案。

### 6.2 无线网络模组（爱科微 ATBM6165）
- **Wi-Fi**：802.11 b/g/n 1T1R @ 2.4 GHz（SDIO 接口，内核驱动模块 `atbm6165.ko`）
- **蓝牙**：BT 4.2 / BLE（UART 接口，内核模块 `hci_uart.ko`，固件文件 `atbm6062_fw.bin`）
- **主线状态**：板级 DTS 声明了 `mmc1` SDIO 槽（Wi-Fi）与 UART3 蓝牙节点，且已启用 `CONFIG_BT_HCIUART` —— 但 `CONFIG_WLAN` **未设置**，开机也未启动任何 Wi-Fi/BT 用户态服务。无线子系统处于 bring-up 进行中，尚未可用。

---

## 7. 主线高版本 Linux (6.6 LTS / 6.12+) 移植技术路线图

### 7.1 Linux 主线已有代码资产
Linux 官方主线代码树（6.6+）已收录 `sun8i-v853` 系列基础架构支持（`arch/arm/boot/dts/allwinner/sun8i-v853.dtsi`）：
- Cortex-A7 单核及 GIC 中断控制器驱动已原生就绪
- 时钟控制器 (`drivers/clk/sunxi-ng/ccu-sun8i-v853.c`) 已包含 PLL/AHB/APB/DE/DSI 节点
- 引脚复用控制 (`drivers/pinctrl/sunxi/pinctrl-sun8i-v853.c`)
- MMC 控制器 (`drivers/mmc/host/sunxi-mmc.c`)
- USB EHCI/OHCI 与 Musb 驱动已主线化

### 7.2 移植高版本 Linux 必须攻克的四大核心模块
1. **片上 PMU 稳压器驱动 (Internal PMU Regulator Driver)**：
   主线内核需补充 `allwinner,v853-pmu` 的稳压驱动；或在板级 DTS 中配置固定稳压器（Fixed Regulators），复用引导阶段已由硬件配置好的供电网络。
2. **显示引擎 DE2.0 与 MIPI-DSI DRM 驱动**：
   将 `wtl1026boe2l` 屏参写入 DRM `panel-simple` 或编写专用驱动子系统，挂接全志 DRM 驱动。
3. **汇顶 GT911 触控 DTS 绑定**：
   在主线 DTS 中绑定标准 `goodix,gt911` I2C 节点并正确配置中断与复位 GPIO。
4. **Bootloader 适配（U-Boot 2024.x 或 SyterKit）**：
   在引导程序中注入本文第 3 节所列的 648MHz DDR3 物理参数及 MMC0 Port F 管脚，生成可在主线 Linux 下直接引导系统的极简固件。
