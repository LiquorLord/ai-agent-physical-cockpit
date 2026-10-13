# Project Specification & Notes / 项目规范与记录

## Project Classification / 项目分类
- **Classification / 项目类型**: 个人项目 (Personal Project)
- **Language Policy / 语言规范**: Bilingual Principle (英汉双语) - All code, comments, documentation, and user guides in both English and Chinese. User chat interactions in Chinese with English technical terms.
  - **Document Separation Standard (双语文档独立成篇规范)**: 杜绝在单一 Markdown 文件中中英文混排。所有文档必须严格拆分为独立的英文版（`*.md`）与中文版（`*_zh.md`），并在文档顶部提供双向跳转导航链接。

---

## Target Device Profile / 目标设备画像
- **Board / Firmware Version (板本/固件号)**: `f900x-v2.2 km2026240602026.01.03jdh`
- **SoC (主控芯片)**: Allwinner V553 (全志 V553)
- **Interface (物理接口)**:
  - USB Type-C (with adjacent recovery/FEL button / 带相邻恢复/FEL按键)
  - Power Key & Reset Key / 电源键与复位键
  - AV-IN Rear Camera Port (2.5mm / 3.5mm) / 后视摄像头接口
  - Debug UART (调试串口): Dedicated PCB test points `tx0`, `rx0`, `GND` (已发现板载测试点)
  - Wireless & Media Features (多媒体特性): Android Auto + Apple AirPlay / CarPlay supported (确认运行全功能 Linux/Tina Linux 系统，具备 Wi-Fi/BT 与硬解流媒体管线)
  - Host Serial Adapters (主机串口端点): `/dev/ttyUSB0`, `/dev/ttyUSB1` (FT232 FTDI available on host)
- **Storage / Flash (存储)**: 16MB SPI NOR Flash (`XM25QH128C`, JEDEC ID `0x204018`)
- **Display Specifications (屏幕规格)**:
  - Panel Model: BOE 10.26-inch ultra-wide bar display (`wtl1026boe2l`)
  - Resolution: 1600 x 600 (MIPI-DSI 4-lane, `/dev/fb0`)
  - Backlight: PWM9 (50kHz hardware frequency modulation)
- **Networking & Wireless (网络与无线)**:
  - Wi-Fi AP: 5GHz Channel 36, SSID `lylink`, WPA2 Passphrase `88888888`, Gateway `192.168.10.1`
- **Milestones & Status (里程碑与状态)**:
  - [x] SPI NOR Flash Full Factory Backup (16MB `backup_factory_16MB.bin`, SHA256 verified)
  - [x] Patched `sunxi-fel` with Allwinner sun8iw21 (V853/V553) SPI0 & CCU support
  - [x] Repacked SquashFS Rootfs & Flashed back via USB FEL mode
  - [x] Direct Root Shell obtained (Passwordless root on console and telnet)
  - [x] USB 2.0 High-Speed OTG ADB enumeration verified (`18d1:d002` @ 480Mbps)
  - [x] Framebuffer specs confirmed: 1600 x 600 RGB565 (Hardware on-the-fly 90° rotation)
  - [x] MicroSD Card prepared with 4.9-compatible ext4 & custom boot engine (`init_usb_monitor.sh`)
  - [x] Dual-Boot Architecture designed: SD Card inserted -> Custom USB Monitor; SD Card removed -> Factory Dashcam
  - [x] Host PC streaming tool created (`stream_to_monitor.py`)
  - [ ] Live testing of SD card detection & screen streaming
