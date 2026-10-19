# 全志 V553 智能物理专属座舱 —— 多模态复合外设与双模架构技术规范

> **语言导航 / Language Navigation**:
> - **中文版本**: [MULTIMODAL_COCKPIT_ARCHITECTURE_zh.md](file:///home/wchen/nas_home/shares/workspace/allwinner/docs/MULTIMODAL_COCKPIT_ARCHITECTURE_zh.md)
> - **English Version**: [MULTIMODAL_COCKPIT_ARCHITECTURE.md](file:///home/wchen/nas_home/shares/workspace/allwinner/docs/MULTIMODAL_COCKPIT_ARCHITECTURE.md)

---

## 1. 架构总揽与核心设计共识

经与用户通过 `/grill-me` 深度对齐，确立将全志 V553（`sun8iw21`）从单一的“录像机/副屏”彻底升维为**多模态端侧 AI 智能物理伴侣座舱（Multimodal Edge-AI Companion Terminal）**。

系统全面摒弃传统的软件级窗口抓屏，采用**全硬件级外设仿真 + 本地边缘智能多模态感知 + 双模无缝漫游架构**。

```
                         ┌────────────────────────────────────────────────────────┐
                         │              Host PC (Linux i3-wm / DRM / KMS)         │
                         └────────────────────────────────────────────────────────┘
                                      ▲                              ▲
                      USB 2.0 High-Speed (480M)          5GHz Wi-Fi (Fallback Wireless)
                                      ▼                              ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                Allwinner V553 SoC (Cortex-A7)                                   │
├────────────────────────────────┬────────────────────────────────┬──────────────────────────────┤
│     USB Composite Gadget       │       On-Device Edge-AI        │     Wireless & Peripherals   │
├────────────────────────────────┼────────────────────────────────┼──────────────────────────────┤
│ 1. GUD DRM Display (0x1d50)    │ • CMOS Cameras (GC2053/TP9950) │ • 5GHz 802.11ac Wi-Fi Display│
│ 2. HID Multi-Touch (Left 1200) │   - Local V4L2 Gesture Pipeline│   - Standalone fallback mode │
│ 3. CDC-NCM Ethernet (192.168)  │   - ✌️/👌/✊ Sub-160ms Gesture │ • Bluetooth 5.0 (BLE)        │
│ 4. Hardware Action Bar (Right) │ • Microphone Array (I2S/PDM)   │   - Proximity Presence Sense │
│   - Proceed / Grill-Me / Stop  │   - Local Keyword Spotting (KWS│   - Wireless media control   │
└────────────────────────────────┴────────────────────────────────┴──────────────────────────────┘
```

---

## 2. 硬件总线速率与通道分配第一性原理

### 2.1 物理总线实测带宽对比

| 物理总线 | 物理介质 / 控制器 | 理论物理速率 | 真实可用有效净荷速率 | 传输时延与抖动 | 稳定性与抗干扰 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **USB 2.0 OTG** | 全志芯片内置 USB PHY | 480 Mbps (60 MB/s) | **320 ~ 360 Mbps (40 ~ 45 MB/s)** | **< 0.125 ms**（超微帧） | 100% 确定性物理线缆，0 丢包 |
| **Wi-Fi 5GHz** | 4-bit SDIO 2.0 @ 50MHz | 200 Mbps (25 MB/s) | **35 ~ 65 Mbps (4 ~ 8 MB/s)** | **5 ~ 25 ms** 抖动 | 易受信道争用与电磁干扰 |

**第一性原理结论**：
有线 USB 2.0 的真实吞吐量是板载 SDIO Wi-Fi 的 **5 ~ 8 倍**，且时延低 **50 倍**。因此，高帧率显示传输必须以 USB 2.0 有线主干为绝对优先核心；Wi-Fi 定位为脱离电脑主机独立供电时的无线漫游后备通道。

### 2.2 多模态通道正交分配矩阵

为防止 USB 2.0 总线争用引发宿主机键鼠卡顿，通道执行严格的**正交化带宽隔离**：

1. **显示通道（USB GUD 显卡，有线主干）**：
   - 采用 Linux 内核主线开源 GUD 协议（`0x1d50:0x614d`）；
   - 支持脏矩形局部微刷新（Damage Rectangles）与整屏更新；
   - 编码为 RGB565 16bpp（$1600 \times 600$ 仅需 1.92MB 显存），平均带宽消耗 16 ~ 40 Mbps，仅占总线带宽的 <10%。
2. **触控通道（双分区混合映射）**：
   - **左侧 $1200 \times 600$**：枚举为标准 USB HID 多点触控屏（Multi-Touch Digitizer），在宿主机 OS 中可直接用手指点击、滑动、拖拽窗口与网页；
   - **右侧 $400 \times 600$**：由板载系统拦截为实体座舱专属 Action Bar（`[PROCEED]`, `[GRILL-ME]`, `[PLAN]`, `[STOP]` 及系统 HUD），点击直接通过底层注入 Antigravity 结对控制指令。
3. **视觉感知（端侧 AI 优先，零总线占用）**：
   - 摄像头原始视频流（1080p @ 30fps，每秒需 60MB 带宽）**绝不送往 USB 总线**；
   - 视频帧留在 V553 芯片本地由 Cortex-A7 / NPU 处理，运行极速手势识别（✌️/👌/✊）与视线感知，仅将识别后的几字节结构化事件指令上传给宿主机。
4. **语音感知（端侧 AI 优先，零总线占用）**：
   - 麦克风音频留存本地进行唤醒词监测（Keyword Spotting），降低总线与宿主机 CPU 开销。
5. **网络与调试通道（CDC-NCM 高速以太网）**：
   - 维持 `192.168.100.1` 虚拟以太网，提供 ADB、免密 Root SSH 与远程状态遥测。

---

## 3. 双模智能无缝漫游（Wired & Wireless Roaming）

1. **有线工作站模式（Connected to PC USB-C）**：
   - 数据与供电同轴直通；
   - 自动激活 GUD 硬件显卡 + HID 触屏 + CDC-NCM 网络；
   - 宿主机识别为硬件扩展显示器（`/dev/dri/card0` 1600x600）。
2. **无线独立漫游模式（Standalone Charger / Power Bank）**：
   - 当检测到 USB 数据枚举超时（仅接入 5V 充电宝/车载供电时）；
   - 板端自动激活 5GHz Wi-Fi AP 或加入局域网；
   - 启动无线副屏服务（基于轻量化低时延 UDP/RTSP 投屏），作为桌面上独立的无线 AI 伴侣与仪表盘。
3. **蓝牙（BLE）近场交互**：
   - 广播 BLE 近场信号；
   - 结合宿主机端守护脚本，实现“人来即刻点亮副屏并解锁工作流、人走自动黑屏保护隐私”。

---

## 4. 固件守护进程与关键服务拓扑

| 服务名称 | 源码路径 | 运行时职责 | 硬件节点 |
| :--- | :--- | :--- | :--- |
| `gud_v553` | [`firmware/apps/gud_v553.c`](file:///home/wchen/nas_home/shares/workspace/allwinner/firmware/apps/gud_v553.c) | GUD 显卡端点协议解析、脏矩形旋转显存映射 | `/dev/ffs-gud` + `/dev/fb0` |
| `touch_cockpit` | [`firmware/apps/touch_cockpit.go`](file:///home/wchen/nas_home/shares/workspace/allwinner/firmware/apps/touch_cockpit.go) | 汇顶 GT911 触控拦截、Action Bar 渲染与 UDP 上报 | `/dev/input/event0` |
| `gesture_detector`| [`firmware/apps/gesture_detector.go`](file:///home/wchen/nas_home/shares/workspace/allwinner/firmware/apps/gesture_detector.go)| 本地摄像头 30fps 手势识别与视线追踪 | `/dev/video12` |
| `switch_to_gud` | [`firmware/apps/switch_to_gud.sh`](file:///home/wchen/nas_home/shares/workspace/allwinner/firmware/apps/switch_to_gud.sh) | 复合 Gadget 动态管理、UDC 控制器状态机 | `/sys/kernel/config/usb_gadget` |
