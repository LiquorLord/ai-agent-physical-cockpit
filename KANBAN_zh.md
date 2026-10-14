# AGY 物理级专属座舱 —— 项目看板与进度追踪

> **语言导航 / Language Navigation**:
> - **中文版本**: [KANBAN_zh.md](file:///home/wchen/nas_home/shares/workspace/allwinner/KANBAN_zh.md)
> - **English Version**: [KANBAN.md](file:///home/wchen/nas_home/shares/workspace/allwinner/KANBAN.md)

---

## 1. 项目愿景与架构拓扑

将市售 40 美元全志 V553（`sun8iw21`）行车记录仪硬件（主板版本 `f900x-v2.2 km2026240602026.01.03jdh`）彻底改造为专为 AI 辅助编程（Vibe-Coding）开发者打造的**零认知负担多模态物理伴侣座舱（AGY Physical Cockpit）**。

### 三态安全双启动架构（Dual-Boot Architecture）
1. **插入特制 MicroSD 卡**：100% 从 SD 卡启动专属 AI 智能终端操作系统；永久阻断原厂 `rcS` 继续执行到 `S70app` / `cdr`；1600x600 专属 AI 界面硬锁定屏幕；对外暴露三合一免驱复合 USB 外设（ADB + CDC-NCM 虚拟以太网 + USB HID 键盘）并在 5900 端口提供高吞吐低时延投屏。
2. **插入普通行车记录仪 MicroSD 卡（FAT32/exFAT）**：引导程序与 `S02sdboot` 检测到缺失 `/mnt/sdcard/init_usb_monitor.sh`，立即自动卸载 `/mnt/sdcard`，放行原厂固件启动 `cdr`，将卡挂载至 `/mnt/extsd` 恢复全功能原厂行车记录仪。
3. **未插入任何 MicroSD 卡**：100% 从板载 16MB SPI NOR Flash 启动纯净原厂固件。

---

## 2. 硬件画像与外设映射清单

| 外设子系统 | 规格参数 / 硬件标识 | 物理接口 / 驱动节点 | 当前状态与说明 |
| :--- | :--- | :--- | :--- |
| **SoC 主控** | 全志 V553 (`sun8iw21`) | 单核 Cortex-A7 @ 1.2GHz | 正常运行，看门狗每 3s 稳定喂狗 |
| **超宽显示屏** | 京东方 10.26 英寸 1600x600 | `/dev/fb0`（MIPI-DSI 4-lane，DE2.0） | 支持 `FBIOPAN_DISPLAY` 硬件翻页 |
| **电容触控屏** | 汇顶 Goodix GT911 | `/dev/input/event0`（I2C2 @ `0x5d`） | 原生 600x1600 竖屏坐标采样 |
| **电源按键** | PMC PEK 按键 | `/dev/input/event1` | 硬件电源管理中断正常 |
| **高清摄像头** | CMOS 图像传感器（GC2053） | `/dev/video0`（V4L2 1080p@30） | 就绪，待手势识别接入 |
| **SPI Flash** | 16MB SPI NOR (`XM25QH128C`) | `/dev/mtd*`（SPI0 @ `0x204018`） | 完整原厂固件已备份并校验 |
| **USB 链路** | USB 2.0 High-Speed OTG (480M) | `4100000.udc-controller` | 三合一复合 Gadget 配置就绪 |
| **调试串口** | 板载测试点 115200 8N1 | 宿主机 `/dev/ttyUSB0` | 硬件控制台静默交互就绪 |

---

## 3. 里程碑完成清单

### 阶段一：硬件逆向与脱机烧录工具链
- [x] SPI NOR Flash 完整原厂固件备份（16MB `backup_factory_16MB.bin`，SHA256 校验通过）
- [x] 修补 `sunxi-fel` 工具，注入全志 sun8iw21（V853/V553）SPI0 控制器与 CCU 时钟支持
- [x] 重构 SquashFS 根文件系统，通过 USB FEL 模式成功回写
- [x] 成功获取免密硬件 Root Shell（控制台与 Telnet 均免密直通）

### 阶段二：安全双启动架构与显示驱动
- [x] 制作并格式化 Linux 4.9 内核兼容的 ext4 文件系统特制 MicroSD 卡
- [x] 在 `S02sdboot` 中部署安全双启动切换逻辑（普通卡自动无感回退原厂录像）
- [x] 编写硬件显示激活器（`show_splash`，直接调用 DE2.0 硬件显存翻页 ioctl）
- [x] 在 `init_usb_monitor.sh` 中设计永久监控死循环，100% 隔离阻断原厂 `cdr` 进程
- [x] 1600x600 专属 AI 开机画面在物理屏幕上**确认锁定常驻**
- [x] 编写跨平台 ARMv7 高性能显存接收器（`fb_receiver`，补全 `FBIOPAN_DISPLAY`）

### 阶段三：USB 复合外设与输入子系统
- [x] 修复 `/bin/adb_shell` heredoc 转义缺陷，打通非交互式 `adb shell` 同步指令执行
- [x] 内核配置确认：`CONFIG_USB_F_HID=y` 与 `CONFIG_USB_F_NCM=y` 确认内建支持
- [x] 架构三合一复合 USB 外设（Google ADB `18d1:d002` + CDC-NCM `usb0` + USB HID `/dev/hidg0`）
- [x] 编写并交叉编译 USB HID 硬件键盘报文注入器（`firmware/bin/send_hid_key`）
- [x] 定位汇顶 GT911 电容触摸屏（`/dev/input/event0`，600x1600 原生竖屏坐标）
- [x] 定位 CMOS 高清摄像头传感器（`/dev/video0`，标准 V4L2 架构）

### 阶段四：专属座舱多模态交互闭环（当前攻坚阶段）
- [x] 编写并交叉编译 `touch_cockpit`（在 1600x600 屏幕右侧渲染 Action Bar，将汇顶 GT911 触摸事件映射为动作响应，物理屏幕已成功点亮动作条）
- [x] 编写并交叉编译 `gesture_detector`（V4L2 多平面帧采集架构，160x120 极速降采样，✌️ V / 👌 OK / ✋ Stop 手势分类器）
- [x] 查明全志 `sunxi-vin` 驱动规范（`V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE = 9`，传感器 `p1_mipi` 1280x720 30fps）
- [x] 查明 USB 未识别根因：原厂 4.9.191 内核未内建 `usb_f_hid` / `usb_f_ncm`，导致 ConfigFS 复合设备装载失败从而阻断了 UDC 绑定；同时早期脚本缺少 `/sbin/udevd`
- [x] 深度重构 `init_usb_monitor.sh`：部署全量落盘日志（`boot_debug.log`, `dmesg_boot.log`），显式启动 `S10udev` 与 `S11dev`，回退至原厂验证的 `setusbconfig adb` 高速数据桥接方案
- [x] 编写宿主机端零敲击伴侣守护脚本（`host_tools/cockpit_agent.py`），通过 ADB 桥接电容触控事件
- [ ] 软硬件端到端闭环验证：用户将 SD 卡插回设备并连上 USB 开机，验证 ADB 自动重连与屏幕触控响应

---

## 4. 当前物理机运行与环境快照

- **物理连接状态**：MicroSD 卡已完成最新固件与日志脚本写入，已安全卸载（`sync && udisksctl unmount -b /dev/sdc1`），正等待用户插回 V553 行车记录仪开机。
- **启动引导源**：即将从特制 MicroSD 卡重新引导。
- **ADB 连通性**：后台监听守护已挂起就绪，开机后自动抓取设备。
- **SD 卡已同步文件**：
  - `/mnt/sdcard/init_usb_monitor.sh`（全量持久化日志 `boot_debug.log` + udev 守护启动 + 原厂 UDC 绑定）
  - `/mnt/sdcard/touch_cockpit`（GT911 触控动作条）
  - `/mnt/sdcard/gesture_detector`（视觉手势分类器）
  - `/mnt/sdcard/fb_receiver`（集成 `FBIOPAN_DISPLAY` 硬件翻页）
  - `/mnt/sdcard/show_splash`
  - `/mnt/sdcard/splash.raw`

---

## 5. 当前第一步即时行动

1. 用户将 MicroSD 卡拔出电脑读卡器，插回全志 V553 行车记录仪；
2. 重新连接 USB Type-C 线缆接回电脑开机；
3. 后台监听将自动捕获设备，输出启动日志并打通触控伴侣交互！

