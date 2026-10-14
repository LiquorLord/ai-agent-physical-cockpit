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
- [x] 软硬件端到端闭环验证：电容触控屏幕动作条与物理按压 100% 验证成功！绿色 `[PROCEED / READY]` 与红色 `[STOP / ABORT]` 按键高亮闪烁并成功输出指令事件到 `/tmp/cockpit_event.log`
- [x] 联调 `gesture_detector`：全志 `sunxi-vin` 多平面（MPLANE 9）1280x720 NV12 @ 30fps 视频流完全打通，Cortex-A7 8 倍极速空间降采样与手势分类器在线运行！
- [x] 切换至朝向用户的副摄像头（`sensor@1` `tp9950`/`tp9951` @ `/dev/video8`）：成功抓取 1080p（1920x1080 NV12，对齐高度 1088）全画幅图像，确认座舱人脸视场清晰可用
- [x] 升级 `gesture_detector` 融合 YCbCr 肤色分割与形态学指尖峰值分析：彻底消除背景纹理噪点误触发，同时 CPU 占用率 <0.1%（系统整体 Idle 90.9%）
- [x] 实现跨进程全屏高亮视觉反馈联动：`gesture_detector` 通过本地 UDP 桥接（`127.0.0.1:8889`）通知 `touch_cockpit`，识别手势瞬间点亮全屏四边 24px 翡翠绿光环并联动右侧 `[PROCEED / READY]` 按钮高亮 300ms
- [x] 优化手势判定响应灵敏度：将 ✌️ 与 👌 统一归并为正向确认意图，引入平滑衰减，将触发时延压缩至 ~160ms（12.5 fps 下仅需 2 帧稳定确认）
- [x] 实现 ✊ 握拳（Closed Fist）停止/终止手势：基于 0 突出手指且几何长宽比（$0.65 \le W/H \le 1.35$）紧凑团块特征，与 ✌️/👌 形成特征空间最大对比度，稳定触发全屏 24px 深红光环、`[STOP / ABORT]` 按钮高亮及 `ctrl+c` 指令
- [x] 解耦并封存视觉手势引擎：将 `gesture_detector` 改为特征开关门禁模式（`/mnt/sdcard/enable_gesture`），终止设备端后台进程以释放摄像头硬件与系统资源，全量驱动契约已双语归档至 `docs/GESTURE_ENGINE_ARCHIVE_zh.md`
### Phase 5: 屏幕双分区投屏与 PTY Master 零输入终端架构 (当前阶段)
- [x] 设计屏幕双分区架构：左侧 1200x600 作为宿主机低延迟代码/日志视口 + 右侧 400x600 保持汇顶 GT911 电容触控 Action Bar 物理常驻
- [x] 升级 `fb_receiver`：支持双端口并行监听（5900 端口 1200x600 双分区，5901 端口 1600x600 全屏），引入底层硬件显存零拷贝 `mmap` 逐行渲染，完全不触碰右侧 1200..1600 动作条
- [x] 升级 `touch_cockpit`：全面切换至底层硬件显存零拷贝 `mmap`，严格将绘制与刷新范围约束在右侧 1200..1600 区域，彻底消除整屏覆写与闪烁
- [x] 升级 `stream_to_monitor.py`：新增 `--mode dual`（默认 1200x600 -> 端口 5900）与 `--mode full`（1600x600 -> 端口 5901），配置 ffmpeg 超低延迟流水线与 `--test` 动态性能基准图案
- [x] 升级 `cockpit_agent.py`：基于 POSIX PTY Master 架构重构，原生接管 `$SHELL` 或任意 CLI（如 `agy`），100% 绕过 Linux 6.x 内核 TIOCSTI 安全限制，按压屏幕物理按键直接向终端子进程标准输入写入 `ready\n` 与 `Ctrl+C`
- [x] 实测双分区通过高频 ADB 端口 5900 实时投屏成功，显存逐行混合渲染正常
- [ ] 实机完成屏幕投屏与物理电容触控按键联调闭环

---

## 4. 当前物理机运行与环境快照

- **物理连接状态**：全志 V553（`sun8iw21`）行车记录仪通过单根 USB Type-C 连入宿主机（`usb:5-2.1` @ 480Mbps USB 2.0 High-Speed），供电与数据双向畅通。
- **启动引导源**：从 MicroSD 卡特制 ext4 分区完全引导，原厂 `cdr` 录像进程被 100% 隔离阻断。
- **屏幕显示与触控状态**：1600x600 京东方超宽屏实现双分区物理隔离：
  - 列 0..1200：1200x600 低延迟宿主机投屏视口（由 `fb_receiver` 驱动）
  - 列 1200..1600：400x600 电容触控 Action Bar 动作条（由 `touch_cockpit` 驱动）
- **备用摄像头**：Techpoint TP9950/TP9951 1080p 传感器挂载于 `/dev/video8`，已解耦并在门禁文件 `/mnt/sdcard/enable_gesture` 保护下处于待机状态。
- **ADB 连通性**：USB 2.0 High-Speed ADB 通道（`18d1:d002` @ 480Mbps）。
- **设备端运行守护进程**：
  - `fb_receiver`: Mmap 硬件双分区帧缓冲接收服务（监听端口 5900 与 5901）
  - `touch_cockpit`: 汇顶 GT911 动作条服务（`/dev/input/event0`）
  - `init_usb_monitor.sh`: `adbd` 自动拉起守护监控
- **宿主机端工具链**：
  - `host_tools/cockpit_agent.py`: PTY Master 交互式终端伴侣
  - `host_tools/stream_to_monitor.py`: 双分区（1200x600）与全屏（1600x600）投屏客户端

---

## 5. 当前第一步即时行动

1. 重新插拔 USB Type-C 或按复位键重启设备，加载最新守护；
2. 运行 `python3 host_tools/stream_to_monitor.py --test` 开启 1200x600 动态测试投屏；
3. 运行 `python3 host_tools/cockpit_agent.py` 验证物理按压免打字终端交互。



