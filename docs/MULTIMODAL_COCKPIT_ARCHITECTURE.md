# Allwinner V553 Multimodal Cockpit —— Composite Gadget & Dual-Mode Technical Specification

> **Language Navigation / 语言导航**:
> - **English Version**: [MULTIMODAL_COCKPIT_ARCHITECTURE.md](MULTIMODAL_COCKPIT_ARCHITECTURE.md)
> - **中文版本**: [MULTIMODAL_COCKPIT_ARCHITECTURE_zh.md](MULTIMODAL_COCKPIT_ARCHITECTURE_zh.md)

---

## 1. Architectural Vision & Core Design Consensus

Following the `/grill-me` architectural review, the Allwinner V553 (`sun8iw21`) platform has transitioned from a passive dashcam/monitor into an **Edge-AI Multimodal Companion Terminal (AGY Physical Cockpit)**.

The architecture abandons software-level screen scraping, adopting **Native Hardware USB Peripheral Emulation + On-Device Edge AI Perception + Seamless Dual-Mode Roaming**.

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

## 2. Bus Bandwidth & Channel Allocation First Principles

### 2.1 Measured Bus Throughput Comparison

| Physical Bus | Physical Media / Controller | Theoretical Spec | Realistic Payload Throughput | Latency & Jitter | Signal Reliability |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **USB 2.0 OTG** | On-chip Sunxi USB PHY | 480 Mbps (60 MB/s) | **320 ~ 360 Mbps (40 ~ 45 MB/s)** | **< 0.125 ms** (Microframe) | 100% deterministic physical wire, 0 packet loss |
| **Wi-Fi 5GHz** | 4-bit SDIO 2.0 @ 50MHz | 200 Mbps (25 MB/s) | **35 ~ 65 Mbps (4 ~ 8 MB/s)** | **5 ~ 25 ms** jitter | Prone to RF interference & channel contention |

**First-Principles Conclusion**:
Wired USB 2.0 delivers **5x to 8x higher effective throughput** and **50x lower latency** compared to the onboard SDIO Wi-Fi. Therefore, high-framerate display streaming must use the wired USB 2.0 trunk as its primary backbone; Wi-Fi serves as the secondary wireless fallback when operating untethered on standalone power.

### 2.2 Multimodal Channel Allocation Matrix

To eliminate USB bus contention and prevent typing latency:

1. **Display Channel (USB GUD Display, Primary Trunk)**:
   - Driven by mainline in-tree Linux GUD protocol (`0x1d50:0x614d`).
   - Supports damage rectangles and full updates in RGB565 format ($1600 \times 600$ @ 1.92MB per frame).
   - Average bandwidth consumption: 16 ~ 40 Mbps (<10% of USB 2.0 capacity).
2. **Touchscreen Channel (Dual-Zone Hybrid Mapping)**:
   - **Left $1200 \times 600$**: Standard USB HID Multi-Touch Digitizer, allowing direct window manipulation on host OS.
   - **Right $400 \times 600$**: Intercepted locally by V553 as an on-screen hardware Action Bar (`[PROCEED]`, `[GRILL-ME]`, `[PLAN]`, `[STOP]`, HUD), injecting low-latency Antigravity control commands.
3. **Computer Vision (Edge-AI Priority, Zero Bus Overhead)**:
   - Raw video stream (1080p @ 30fps = 60 MB/s) is **never passed across the USB bus**.
   - Frames remain entirely inside V553 Cortex-A7 / NPU for local gesture recognition (✌️/👌/✊) and gaze tracking; only small event commands are dispatched upstream.
4. **Voice Audio (Edge-AI Priority, Zero Bus Overhead)**:
   - Local keyword spotting (KWS) on audio array, freeing USB bus and host CPU.
5. **Network & Control Channel (CDC-NCM Ethernet)**:
   - Preserves `192.168.100.1` virtual network for ADB, root SSH, and telemetry.

---

## 3. Seamless Dual-Mode Roaming (Wired & Wireless)

1. **Wired Workstation Mode (Connected via USB-C)**:
   - Delivers unified power and data.
   - Auto-activates GUD Display + HID Touch + CDC-NCM Ethernet.
   - Recognized natively by host Linux as `/dev/dri/card0` (1600x600).
2. **Wireless Standalone Mode (Power Bank / Wall Charger)**:
   - When USB data enumeration times out (power-only connection);
   - Auto-switches to 5GHz Wi-Fi AP or station mode.
   - Starts low-latency wireless display service (UDP/RTSP), providing untethered companion capability.
3. **Bluetooth (BLE) Proximity Sensing**:
   - Advertises BLE beacon.
   - Couples with host background daemon to lock workstation on departure and unlock/wake on arrival.

---

## 4. Firmware Daemons & Service Mapping

| Service Name | Source Path | Runtime Role | Hardware Endpoint |
| :--- | :--- | :--- | :--- |
| `gud_v553` | [`firmware/apps/gud_v553.c`](../firmware/apps/gud_v553.c) | GUD protocol handling, damage rect rotation, fb blitting | `/dev/ffs-gud` + `/dev/fb0` |
| `touch_cockpit` | [`firmware/apps/touch_cockpit.go`](../firmware/apps/touch_cockpit.go) | Goodix GT911 touch interception, Action Bar rendering | `/dev/input/event0` |
| `gesture_detector`| [`firmware/apps/gesture_detector.go`](../firmware/apps/gesture_detector.go)| Local 30fps gesture classification and gaze detection | `/dev/video12` |
| `switch_to_gud` | [`firmware/apps/switch_to_gud.sh`](../firmware/apps/switch_to_gud.sh) | Composite gadget management, UDC controller binding | `/sys/kernel/config/usb_gadget` |
