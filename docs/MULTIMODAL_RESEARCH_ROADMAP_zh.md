# 全志 V553 多模态硬件扩展与驱动研发规格书（手势识别、蓝牙、麦克风）

> **语言导航 / Language Navigation**:
> - **中文版本**: [MULTIMODAL_RESEARCH_ROADMAP_zh.md](MULTIMODAL_RESEARCH_ROADMAP_zh.md)
> - **English Version**: [MULTIMODAL_RESEARCH_ROADMAP.md](MULTIMODAL_RESEARCH_ROADMAP.md)

---

## 1. 概述与研发目标

在完成全志 V553（`sun8iw21p1`）智能座舱终端主线 Linux 6.12 LTS 基础内核引导、DRM/KMS 显示全栈、触控屏、硬件看门狗及免补丁 USB 复合设备（纯净 1200x600 GUD 显卡 + 400x600 HID 按键）的离线迭代后，本规范针对后续三大核心多模态感知能力的硬件拓扑、内核驱动架构与应用层集成进行深度技术剖析：

1. **摄像头智能识别手势（Camera Gesture Recognition）**：
   - 物理传感器管线（前置风挡摄与座舱人脸/手势摄）。
   - V4L2 多平面（MPLANE）流媒体帧捕获与 YCbCr 肤色色度空间形态学特征分类。
   - 全志 V553 片上 Vivante GC Nano NPU（`0x03050000`）硬件加速推理演进路径。
2. **蓝牙驱动（Bluetooth Bringup）**：
   - AltoBeam ATBM6165 三模无线芯片物理互联。
   - 4 线高速硬件流控串口 UART3（引脚 PA16~PA19）驱动与波特率协商。
   - 标准 Linux BlueZ 协议栈（`btattach` / `hci_uart`）与 `hci0` 端点激活。
3. **麦克风音频驱动（Microphone Driver）**：
   - 片上数字麦克风控制器 DMIC（寄存器基地址 `0x02031000`）。
   - 片上模拟音频编解码器 Codec（`0x02030000`）与模拟差分麦克风引脚（MIC1P/MIC1N）。
   - Linux 6.12 ALSA SoC 原生驱动（`sound/soc/sunxi/sun50i-dmic.c`）集成。

---

## 2. 摄像头智能手势识别技术架构

### 2.1 双摄像头物理拓扑与设备树
设备树硬件节点 `/soc/vind@0/` 下挂载了两路互不干扰的独立视频输入传感器：
1. **Sensor 0（风挡前向摄）**：
   - 驱动模块：`p1_mipi.ko`。
   - 物理接口：MIPI-CSI 接口。
   - 设备节点：`/dev/video0`（1280x720 NV12 @ 30fps）。
2. **Sensor 1（座舱人脸/手势后视输入）**：
   - 驱动模块：`tp9950.ko`（Techpoint HD 视频接收器，芯片 ID `0x6028`）。
   - 物理接口：AV-IN 模拟高清转数字接口输入。
   - 设备节点：`/dev/video8`（原生 1920x1080 NV12 @ 30fps）。

### 2.2 V4L2 Multi-Planar 驱动契约与 DMA 对齐
- **缓冲区类型**：必须声明 `V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE`（枚举值 9）。若传入单平面类型（`type = 1`），内核驱动将返回 `EINVAL`（错误码 22）。
- **帧对齐硬指标**：
  - 格式：`V4L2_PIX_FMT_NV12`（`0x3231564E`，Y 平面紧跟交织 UV 平面）。
  - 分辨率：1920 x 1080，但硬件 DMA 强制对齐到 **1088 行**（16 像素硬件步长对齐）。
  - 单帧缓冲区大小：$1920 \times 1088 \times 1.5 = \mathbf{3,133,440\text{ 字节}}$。
- **队列死锁铁律**：内核驱动强制要求最少申请 3 个队列缓冲区（`req.count = 3`）。在触发 `VIDIOC_STREAMON` 之前，必须对全部 3 个缓冲区完成内存映射（`mmap`）并通过 `VIDIOC_QBUF` 预先入队，否则驱动硬件引擎将在起流时死锁冻结。

### 2.3 手势识别状态机与双区座舱联动
预置的轻量化分类器引擎（[`firmware/apps/gesture_detector.go`](../firmware/apps/gesture_detector.go)）通过 YCbCr 色度分析（$Cb \in [77, 127]$, $Cr \in [133, 173]$）结合 $8 \times 8$ 降采样网格实现手掌/握拳形态学判决，并通过 UDP 报文通知 `touch_cockpit` 呈现视觉反馈：
- **手势一：张开手掌 ✋（Open Palm / PROCEED）**：
  - 识别逻辑：检测到手掌轮廓面积占比 $> 35\%$ 且离散凸包外扩。
  - 联动响应：向 UDP `127.0.0.1:8889` 发送 `"ready"`，屏幕四周触发 24px 翡翠绿光晕（`0x07E0`），高亮 Action Bar 按钮 1（`[PROCEED / READY]`），并向终端输出 `proceed\n`。
- **手势二：握紧拳头 ✊（Closed Fist / STOP）**：
  - 识别逻辑：检测到紧凑高密度核心区域且外凸边缘凹缩。
  - 联动响应：向 UDP `127.0.0.1:8889` 发送 `"ctrl+c"`，屏幕四周触发 24px 猩红色光晕（`0xF800`），高亮 Action Bar 按钮 4（`[STOP / ABORT]`），立即中断终端当前任务。

### 2.4 片上 Vivante GC Nano NPU 深度学习演进
全志 V553 集成了一颗 Vivante GC Nano NPU（设备树节点 `npu@3050000`，提供约 0.5~1.0 TOPS 算力）：
- **主线驱动**：主线 Linux 已通过开源 `etnaviv` DRM 驱动支持 Vivante 神经网络加速器核。
- **模型管线**：可将 Google MediaPipe Hands 或 YOLO-Fastest 手势分类网络通过 TIM-VX / VeriSilicon 深度学习编译器量化为 INT8 格式。
- **运行模式**：V4L2 抓取 NV12 图像帧 $\rightarrow$ NPU 硬件缩放与张量输入 $\rightarrow$ 毫秒级输出 21 关键点坐标，彻底杜绝误报。

---

## 3. 蓝牙驱动研发规格（Bluetooth Bringup）

### 3.1 物理互联拓扑
V553 主板搭载 AltoBeam ATBM6165 芯片（Wi-Fi 6 + BLE 5.4 + BR/EDR 三模）：
- **Wi-Fi 通道**：连接至 MMC1 SDIO 接口（引脚 PE0~PE5）。
- **蓝牙通道**：独立连接至 SoC 的 4 线全双工硬件流控串口 **UART3**（基地址 `0x02500c00`，中断号 GIC SPI 5）：
  - `PA16`：UART3 TXD
  - `PA17`：UART3 RXD
  - `PA18`：UART3 RTS（硬件流控发送请求）
  - `PA19`：UART3 CTS（硬件流控允许发送）

### 3.2 设备树配置节点（DTS）
```dts
&uart3 {
    pinctrl-names = "default";
    pinctrl-0 = <&uart3_pa_pins>;
    uart-has-rtscts;
    status = "okay";
};
```

### 3.3 BlueZ 协议栈接入方案
1. **加载驱动**：内核启用 `CONFIG_BT=y`, `CONFIG_BT_HCIUART=y`, `CONFIG_BT_HCIUART_H4=y`。
2. **初始化握手**：
   ```bash
   # 以 115200 初始波特率附着，随后协商切换到 1.5Mbps / 3Mbps 高速数据通道
   hciattach -n -s 115200 /dev/ttyS3 any 1500000 flow
   ```
3. **用户态协议栈**：生成标准 `/dev/hci0` 后，启动 `bluetoothd` 守护进程，支持 BLE HID 设备扫描与手机音视频控制接入。

---

## 4. 麦克风音频驱动研发规格（Microphone Driver）

### 4.1 硬件控制器双架构
V553 SoC 内置两套完整的音频输入控制器：
1. **全志数字麦克风控制器（DMIC Controller）**：
   - 寄存器基地址：`0x02031000`。
   - 时钟源：`CLK_DMIC`（ID 89）、`CLK_BUS_DMIC`（ID 90），复位源 `RST_BUS_DMIC`（ID 36）。
   - 物理引脚：`PH0`~`PH4` 或 `PH9`~`PH10`（时钟线 CLK + 数据线 DATA0）。
   - 适用场景：板载 PDM 数字硅麦阵列，抗电磁干扰极强，信噪比高。
2. **内部模拟音频编解码器（Internal Audio Codec）**：
   - 寄存器基地址：`0x02030000`。
   - 物理输入：`MIC1P` / `MIC1N` 模拟差分麦克风引脚。
   - 内部集成低噪声麦克风偏置供电（MICBIAS）与可编程增益放大器（PGA）。

### 4.2 主线 Linux 6.12 驱动适配
Linux 6.12 源码树中内置官方成熟驱动：
- 驱动源码路径：`sound/soc/sunxi/sun50i-dmic.c`
- 设备树匹配字符串：`compatible = "allwinner,sun50i-h6-dmic"`
- 驱动特性：支持标准 ALSA SoC DMAEngine PCM 录音，采样率覆盖 8kHz~48kHz，支持 16-bit / 24-bit PCM 音频流。

### 4.3 验证与测试命令
```bash
# 录制 16kHz 16位 单声道语音测试文件
arecord -D hw:0,0 -f S16_LE -r 16000 -c 1 /tmp/test_mic.wav
```

---

## 5. 总结与下阶段路线图

| 模块 | 硬件端点 | 内核驱动 | 状态 | 下一步行动 |
| :--- | :--- | :--- | :--- | :--- |
| **手势识别** | `/dev/video8` (TP9950) | `sunxi-vin` / V4L2 MPLANE | 算法就绪 | 接入主线 Media Controller 框架并评估 NPU 模型量化 |
| **蓝牙驱动** | `/dev/ttyS3` (PA16..PA19) | `sun8i-uart` + `hci_uart` | 硬件明确 | 设备树启用 UART3 并运行 `hciattach` 测试 |
| **麦克风** | `0x02031000` (DMIC) | `sound/soc/sunxi/sun50i-dmic.c` | 原生支持 | DTS 添加 DMIC 与 simple-audio-card 节点并录音实测 |
