# AGY 物理级专属智能交互座舱 (AGY Physical Cockpit)

[English](README.md) | [简体中文](README_zh.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![SoC: Allwinner V553](https://img.shields.io/badge/SoC-Allwinner%20V553%20(sun8iw21)-blue.svg)]()
[![Display: 1600x600 MIPI](https://img.shields.io/badge/Display-1600x600%20MIPI--DSI-green.svg)]()
[![Dual-Boot: Non-Intrusive](https://img.shields.io/badge/Dual--Boot-100%25%20Non--Intrusive-success.svg)]()

> **“拔下 SD 卡，是 $40 美元原厂行车记录仪；插上 SD 卡，是 10.26 寸专属边缘 AI 交互座舱，Vibe-Coding 程序员的物理级生产力玩具！”**

---

## 1. 项目概述

**AGY Physical Cockpit** 是一个开源软硬件结合项目，旨在将市面上极具性价比的全志 V553 (`sun8iw21`) 车载智能条形屏重构为免驱、多模态感知的物理级 AI 辅助编程专属伴侣终端（配套 Antigravity CLI / Vibe Coding 体验）。

### 核心铁律：100% 物理级无损双系统
- **零焊接、零硬件改装**：无需任何物理破坏或飞线。
- **拔出 MicroSD 卡**：完全运行 16MB 板载 SPI NOR Flash 中的原厂 Linux 4.9 系统（`cdr` 行车录像、无线 Apple CarPlay / 华为 HiCar）。
- **插入 MicroSD 卡**：全志片上 BROM 硬件第一优先级直接引导 SDC0（扇区 16 eGON 引导头），无缝进入定制的 AI 智能座舱系统。

---

## 2. 核心特性

- 🖥️ **10.26 英寸超宽带状触摸屏 (1600x600)**:
  - 4-Lane MIPI-DSI 高刷带状屏 (`wtl1026boe2l`)，搭配汇顶科技 Goodix GT911 电容多点触摸。
  - 黄金三区 HUD 布局：实时思考链展示、代码 Diff 与规划图纸、常驻物理触控交互面板。
- 🔌 **通用免驱 USB 复合设备 (Composite Gadget)**:
  - 通过 USB Type-C 接口免驱枚举为 **USB CDC-NCM 高速虚拟以太网**（192.168.10.x，约 480Mbps）与 **USB HID 虚拟键盘** (`/dev/hidg0`)。
  - 原生免驱支持 **Linux、macOS 与 Windows**，即插即用，无需安装第三方驱动。
- 🖲️ **物理触控交互（彻底告别确认打字）**:
  - 屏幕右侧常驻大尺寸触控按键：`🟢 PROCEED`（向主机注入回车键或 `y`）、`🛑 STOP`（向主机注入 `Ctrl+C`）、`📋 PLAN`（向输入框注入 `/plan`）。
  - 专注编写代码，无需频繁敲字命令智能体继续。
- 👁️ **端侧多模态边缘感知**:
  - **视觉引擎**：板载摄像头实时手势识别（👌 OK 手势触发继续、✌️ V 手势触发批准/规划、✋ 掌心手势触发紧急停止）。
  - **双模算力架构**：
    - *便携本模式*：ARM Cortex-A7 NEON 离线轻量推理。
    - *工作站模式*：混合直通模式，高速推送音视频至局域网强算力节点（`192.168.1.188` / RTX 4090），执行全量语音听写与大模型推理。

---

## 3. 仓库目录结构

```
├── README.md                      # 英文项目主文档与快速指南
├── README_zh.md                   # 中文项目主文档与快速指南
├── GEMINI.md                      # 项目持久化规范与技术记忆
├── docs/                          # 深度技术白皮书
│   ├── hardware_specs.md          # 硬件规格白皮书（英文版）
│   ├── hardware_specs_zh.md       # 硬件规格白皮书（中文版）
│   ├── dual_boot_architecture.md  # 硬件双系统机理（英文版）
│   ├── dual_boot_architecture_zh.md # 硬件双系统机理（中文版）
│   ├── mainline_linux_roadmap.md  # 主线 Linux 6.x 移植路线（英文版）
│   └── mainline_linux_roadmap_zh.md # 主线 Linux 6.x 移植路线（中文版）
├── factory_reverse/               # 原厂逆向工程与恢复资产
│   ├── backups/                   # 原厂 16MB SPI 固件金标备份与 SHA256
│   ├── dts/                       # 反编译设备树源文件（DTS/DTB）
│   ├── flasher/                   # SPI 刷机工具与打补丁的 sunxi-tools
│   ├── rootfs/                    # 原厂 SquashFS 镜像及解包文件系统
│   ├── scripts/                   # 串口阻截、冷启动捕获与分析脚本
│   └── logs/                      # 原始启动日志与硬件追踪记录
├── firmware/                      # 运行于 MicroSD 卡的智能终端固件
│   ├── apps/                      # 端侧核心应用源码（fb_receiver, cockpit-core）
│   ├── bin/                       # 编译好的 ARMv7 静态二进制文件
│   ├── boot/                      # 自定义引导脚本与启动服务配置
│   └── splash/                    # 1600x600 专属开机画面资产
├── host_tools/                    # 主机端推流与协同守护工具
│   └── stream_to_monitor.py       # 虚拟副屏推流与测试图样发生器
└── scratch/                       # 临时调试与实验工作区
```

---

## 4. 硬件规格清单

| 硬件部件 | 规格参数 | 详情说明 |
| :--- | :--- | :--- |
| **主控芯片 (SoC)** | 全志 V553 (`sun8iw21`) | 单核 ARM Cortex-A7 @ 900MHz, NEON/VFPv4, E907 RISC-V, 1 TOPS NPU |
| **运行内存 (RAM)** | 128 MB DDR3 | SoC 芯片封装内集成 |
| **板载存储 (Flash)** | 16 MB SPI NOR Flash | `XM25QH128C`（原厂固件存放区） |
| **扩展存储** | 支持高达 128 GB MicroSD | SDC0 接口（全志 BROM 硬件第一启动项） |
| **显示屏幕** | 10.26 英寸 IPS 超宽带状屏 | BOE `wtl1026boe2l`，1600x600，4-lane MIPI-DSI，DCLK 53.0MHz |
| **触摸面板** | 电容式多点触摸 | 汇顶科技 Goodix GT911，挂载于 I2C1 |
| **摄像头** | 前置超广角摄像头 | MIPI CSI 接口，`/dev/video0` |
| **音频系统** | 板载麦克风与扬声器接口 | 集成 MEMS 麦克风与音频编解码芯片 |
| **USB 接口** | Type-C USB 2.0 OTG | 高速 480Mbps，支持 ConfigFS 复合设备（NCM+HID） |
| **调试串口** | Debug UART0 | PCB 板载测试点 `tx0`, `rx0`, `GND` @ 115200 8N1 |
| **主板型号** | `f900x-v2.2` | 丝印批次 `km2026240602026.01.03jdh` |

---

## 5. 快速上手

### 5.1 制作双系统启动卡
1. 将 MicroSD 卡分区格式化为 Ext4（关闭 64bit inode，适配 32 位系统）：
   ```bash
   sudo mkfs.ext4 -O ^64bit -L "COCKPIT" /dev/sdX1
   ```
2. 将固件核心文件拷贝至 SD 卡根目录：
   ```bash
   cp firmware/bin/fb_receiver /mnt/sdcard/
   cp firmware/splash/splash.raw /mnt/sdcard/
   cp firmware/boot/init_usb_monitor.sh /mnt/sdcard/
   ```

### 5.2 主机端推流
使用 USB Type-C 数据线将设备连接至电脑：
```bash
# 测试 1600x600 @ 30fps 高速动态测试图样
./host_tools/stream_to_monitor.py --test --fps 30

# 推送当前电脑的 X11 桌面或窗口至设备屏幕
./host_tools/stream_to_monitor.py --display :1.0 --fps 30
```

---

## 6. 开源协议与致谢

- **开源协议**：MIT License。
- **致谢与参考**：
  - `linux-sunxi` 社区对全志芯片逆向工程的先驱贡献。
  - András Szemző 与 YuzukiHD 对 `sun8iw21` / V853 主线内核的支持与提交。
  - Google DeepMind Antigravity 项目团队。
