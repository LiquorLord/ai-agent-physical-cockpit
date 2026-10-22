# Allwinner V553 Multimodal Hardware Expansion & Driver Research Specification (Gestures, Bluetooth, Microphone)

> **Language Navigation / 语言导航**:
> - **English Version**: [MULTIMODAL_RESEARCH_ROADMAP.md](MULTIMODAL_RESEARCH_ROADMAP.md)
> - **中文版本**: [MULTIMODAL_RESEARCH_ROADMAP_zh.md](MULTIMODAL_RESEARCH_ROADMAP_zh.md)

---

## 1. Executive Summary & Research Objectives

Following the successful offline iteration of the Allwinner V553 (`sun8iw21p1`) smart cockpit terminal baseline on mainline Linux 6.12 LTS—including DRM/KMS display full-stack integration, capacitive touchscreen bringup, hardware watchdog restart protection, and the patch-free USB composite gadget (pure 1200x600 GUD display + 400x600 HID action bar)—this document presents an authoritative technical investigation and architectural roadmap for three core multimodal sensing subsystems:

1. **Camera Intelligent Gesture Recognition**:
   - Physical sensor pipelines (windshield front-facing and cabin gesture cameras).
   - V4L2 multi-planar (`V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE`) streaming capture and YCbCr skin chrominance morphological classification.
   - Evolution path to on-chip Vivante GC Nano NPU (`0x03050000`) hardware-accelerated deep learning inference.
2. **Bluetooth Driver Bringup**:
   - Physical interconnect to the AltoBeam ATBM6165 tri-mode wireless chipset.
   - 4-wire high-speed hardware flow-control UART3 (`PA16..PA19`) and baud rate negotiation.
   - Standard Linux BlueZ stack integration (`btattach` / `hci_uart`) and `/dev/hci0` interface activation.
3. **Microphone Audio Driver**:
   - On-chip Digital Microphone (DMIC) controller at physical base address `0x02031000`.
   - On-chip internal audio Codec (`0x02030000`) and analog differential inputs (`MIC1P`/`MIC1N`).
   - Upstream Linux 6.12 ALSA SoC native driver (`sound/soc/sunxi/sun50i-dmic.c`) bringup.

---

## 2. Camera Intelligent Gesture Recognition Architecture

### 2.1 Dual Camera Physical Topology & Device Tree
Under `/soc/vind@0/`, the device tree specifies two independent video capture sensors:
1. **Sensor 0 (Windshield Forward-Facing Camera)**:
   - Kernel module: `p1_mipi.ko`.
   - Physical interface: MIPI-CSI.
   - Device node: `/dev/video0` (1280x720 NV12 @ 30fps).
2. **Sensor 1 (Cabin Gesture & Face Camera)**:
   - Kernel module: `tp9950.ko` (Techpoint HD video receiver, chip ID `0x6028`).
   - Physical interface: AV-IN composite HD input.
   - Device node: `/dev/video8` (native 1920x1080 NV12 @ 30fps).

### 2.2 V4L2 Multi-Planar Driver Contract & DMA Alignment
- **Buffer Type**: Must declare `V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE` (enum value 9). Passing single-planar capture (`type = 1`) returns `EINVAL` (error code 22).
- **DMA Buffer Alignment**:
  - Pixel Format: `V4L2_PIX_FMT_NV12` (`0x3231564E`, Y plane followed by interleaved UV plane).
  - Native Resolution: 1920 x 1080, with hardware DMA stride aligned to **1088 lines** (16-pixel boundary alignment).
  - Single Frame Size: $1920 \times 1088 \times 1.5 = \mathbf{3,133,440\text{ bytes}}$.
- **Hardware Queue Requirement**: The driver requires a minimum queue allocation of 3 buffers (`req.count = 3`). Crucially, all 3 buffers must be memory-mapped (`mmap`) and queued (`VIDIOC_QBUF`) **prior to calling `VIDIOC_STREAMON`**; otherwise, the hardware capture engine freezes.

### 2.3 Gesture Classifier State Machine & Cockpit HUD Coupling
The lightweight gesture engine ([`firmware/apps/gesture_detector.go`](../firmware/apps/gesture_detector.go)) processes downsampled frames ($8 \times 8$ grid) with YCbCr chrominance filtering ($Cb \in [77, 127]$, $Cr \in [133, 173]$), transmitting UDP datagrams to `touch_cockpit` on `127.0.0.1:8889`:
- **Gesture 1: Open Palm ✋ (PROCEED / READY)**:
  - Detection: Palm contour area $> 35\%$ with expanded convex hull.
  - Action: Emits `"ready"` to UDP `:8889` $\rightarrow$ triggers 24px emerald green border glow (`0x07E0`) on screen, flashes Action Bar Button 1 (`[PROCEED / READY]`), and outputs `proceed\n` to developer terminal.
- **Gesture 2: Closed Fist ✊ (STOP / ABORT)**:
  - Detection: High-density compact core with concave contour contraction.
  - Action: Emits `"ctrl+c"` to UDP `:8889` $\rightarrow$ triggers 24px crimson red border glow (`0xF800`) on screen, flashes Action Bar Button 4 (`[STOP / ABORT]`), immediately halting active host tasks.

### 2.4 Vivante GC Nano NPU Deep Learning Evolution Path
The Allwinner V553 embeds a Vivante GC Nano NPU (`npu@3050000`, providing ~0.5–1.0 TOPS compute):
- **Upstream Driver**: Supported in modern mainline Linux via the open-source `etnaviv` DRM driver stack.
- **Model Pipeline**: Quantize Google MediaPipe Hands or YOLO-Fastest gesture networks into INT8 format using TIM-VX / VeriSilicon compiler tools.
- **Zero-Copy Pipeline**: V4L2 captures NV12 frames directly into NPU-accessible DMA buffers $\rightarrow$ sub-10ms 21-landmark hand tracking with zero false positives.

---

## 3. Bluetooth Driver Technical Specification

### 3.1 Physical Interconnect
The onboard AltoBeam ATBM6165 module integrates dual-band Wi-Fi 6 and Bluetooth 5.4:
- **Wi-Fi Interface**: SDIO over MMC1 (`PE0..PE5`).
- **Bluetooth Interface**: Dedicated 4-wire high-speed UART **UART3** (`0x02500c00`, IRQ GIC SPI 5):
  - `PA16`: UART3 TXD
  - `PA17`: UART3 RXD
  - `PA18`: UART3 RTS (hardware flow-control request to send)
  - `PA19`: UART3 CTS (hardware flow-control clear to send)

### 3.2 Device Tree Configuration
```dts
&uart3 {
    pinctrl-names = "default";
    pinctrl-0 = <&uart3_pa_pins>;
    uart-has-rtscts;
    status = "okay";
};
```

### 3.3 BlueZ Protocol Stack Bringup
1. **Kernel Drivers**: Enable `CONFIG_BT=y`, `CONFIG_BT_HCIUART=y`, `CONFIG_BT_HCIUART_H4=y`.
2. **Initialization Command**:
   ```bash
   # Attach UART3 at initial 115200 baud, negotiating up to 1.5Mbps / 3Mbps with RTS/CTS
   hciattach -n -s 115200 /dev/ttyS3 any 1500000 flow
   ```
3. **Userspace Services**: Once `/dev/hci0` appears, launch `bluetoothd` to enable BLE scanning, pairing, and peripheral connectivity.

---

## 4. Microphone Driver Technical Specification

### 4.1 Audio Controller Architecture
The V553 SoC features two independent audio input pathways:
1. **Allwinner Digital Microphone Controller (DMIC)**:
   - Register base: `0x02031000`.
   - Clocks: `CLK_DMIC` (99), `CLK_BUS_DMIC` (100); Reset: `RST_BUS_DMIC` (46).
   - Pins: `PH0..PH4` or `PH9..PH10` (clock line CLK + data lines DATA0..DATA3).
   - Designed for onboard PDM digital MEMS microphones with superior EMI rejection.
2. **Internal Audio Codec**:
   - Register base: `0x02030000`.
   - Physical inputs: `MIC1P` / `MIC1N` analog differential lines.
   - Built-in low-noise microphone bias (MICBIAS) and programmable gain amplifier (PGA).

### 4.2 Upstream Linux 6.12 ALSA SoC Integration
Mainline Linux provides native support in:
- Driver source: `sound/soc/sunxi/sun50i-dmic.c`
- DT compatible: `compatible = "allwinner,sun50i-h6-dmic"`
- Capabilities: DMAEngine PCM capture, 8kHz–48kHz sampling rates, 16-bit / 24-bit PCM.

### 4.3 Validation Command
```bash
# Capture a 16kHz 16-bit mono voice sample via ALSA
arecord -D hw:0,0 -f S16_LE -r 16000 -c 1 /tmp/test_mic.wav
```

---

## 5. Subsystem Summary & Roadmap Table

| Subsystem | Hardware Endpoint | Driver Pipeline | Readiness | Next Action |
| :--- | :--- | :--- | :--- | :--- |
| **Gesture Recognition** | `/dev/video8` (TP9950) | `sunxi-vin` / V4L2 MPLANE | Algorithm Verified | Integrate with mainline Media Controller & evaluate NPU quantization |
| **Bluetooth** | `/dev/ttyS3` (PA16..PA19) | `sun8i-uart` + `hci_uart` | Architecture Clear | Enable UART3 in board DTS and run `hciattach` test |
| **Microphone** | `0x02031000` (DMIC) | `sound/soc/sunxi/sun50i-dmic.c` | In-Tree Driver | Add DMIC and simple-audio-card nodes in DTS and test recording |
