# Allwinner V553 Modern Mainline Linux 6.12 Display Pipeline Porting & Root-Cause Resolution Guide

> **Language Navigation / 语言导航**:
> - **English Version**: [V553_MAINLINE_DISPLAY_PORTING_GUIDE.md](V553_MAINLINE_DISPLAY_PORTING_GUIDE.md)
> - **中文版本**: [V553_MAINLINE_DISPLAY_PORTING_GUIDE_zh.md](V553_MAINLINE_DISPLAY_PORTING_GUIDE_zh.md)

---

## 1. Overview & Problem Statement

When porting the Allwinner V553 (`sun8iw21p1`) smart cockpit terminal from the legacy vendor BSP Linux 4.9.191 kernel to modern mainline Linux 6.12 LTS, a quintessential embedded hardware bringup defect was observed:
**Approximately 8–10 seconds after power-on, the screen backlight illuminated with a CRT-like rounded glow, but the panel presented no graphical scanout whatsoever (blank/white uninitialized state).**

Through a rigorous, first-principles comparative audit of the legacy Linux 4.9 BSP (DISP2 architecture) against modern mainline Linux 6.12 (DRM/KMS architecture), **five critical root causes** were isolated and resolved. This guide systematically documents the technical findings and engineering solutions.

---

## 2. Display Pipeline Architecture Comparison: 4.9 DISP2 vs 6.12 DRM/KMS

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Linux 4.9 Vendor Kernel (DISP2)                 │
│                                                                        │
│   Display Engine 2 ──► TCON0 (tcon0_cfg_mode_tri) ──► DSI (LPTX DCS)   │
│   (300MHz Clock)       (SUNXI_DSI_PASSIVE_BUG: /16)  │ (Sync Pulse)    │
│                                                      ▼                 │
│   /dev/fb0 (16bpp RGB565) ◄───────────────────────── BOE 10.26" Screen │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│                        Linux 6.12 Modern Kernel (DRM/KMS)              │
│                                                                        │
│   DE2 (mixer0) ──► tcon_top ──► tcon_lcd0 ──► DSI Transmitter ──► Panel│
│   (CRTC Atomic)    (Clock Tree) (0x0000ffff   (SYNC_PULSE +      (DCS in│
│                                 Full IRQ Ack)   TCON DRQ Flow)     LP) │
│                                                      │                 │
│   /dev/fb0 (32bpp XRGB8888) ◄────────────────────────┴─────────────────┘
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. The Five Core Root Causes & First-Principles Dissection

### 3.1 Root Cause #1: Silicon Quirk `SUNXI_DSI_PASSIVE_BUG` & Inter-Line Gap Distortion

- **Physical Silicon Behavior**:
  On Allwinner V553 (`sun8iw21`) and `sun50iw10` chips, a silicon hardware quirk designated `SUNXI_DSI_PASSIVE_BUG` exists: **the TCON0 internal DSI FIFO fetch engine reads data from memory at a fixed 4-lane rate, regardless of the active physical DSI lane count (2-lane)**.
- **Legacy 4.9 Handling (`de_lcd_sun50iw10.c:899`)**:
  ```c
  #if defined(SUNXI_DSI_PASSIVE_BUG)
      /* Hardcodes divisor to (tcon_div * 4) rather than physical lane count (2) */
      lcd_dev[sel]->tcon0_cpu_tri0.bits.block_space =
          panel->lcd_ht * dsi_pixel_bits / (tcon_div * 4) - panel->lcd_x - 40;
  #endif
  ```
  Substituting parameters: $648 \times 24 / (4 \times 4) - 600 - 40 = 972 - 640 = \mathbf{332}$.
- **Mainline 6.12 Upstream Defect (`sun4i_tcon.c:395`)**:
  Upstream mainline divided by `(tcon_div * lanes)`:
  Substituting physical 2-lane: $648 \times 24 / (4 \times 2) - 600 - 40 = 1944 - 640 = \mathbf{1304}$!
  - **Severe Consequence**: The inter-line gap wait cycles were inflated by 4x (1304 cycles), far exceeding the horizontal blanking period. The TCON FIFO starved immediately, terminating line scanout.

### 3.2 Root Cause #2: Porch Semantics Discrepancy & TCON DRQ Activation (`/* Maaaaaagic */`)

- **Parameter Semantic Divergence**:
  In Allwinner 4.9 DTS, `lcd_hbp = <0x14>` (20) **subsumes the sync pulse width `lcd_hspw = <8>`**.
  - True Pure Back Porch (HBP): $20 - 8 = \mathbf{12}$.
  - True Pure Front Porch (HFP): $648 - 600 - 20 = \mathbf{28}$.
- **Mainline Driver Threshold Gate (`sun6i_mipi_dsi.c:440`)**:
  Upstream mainline contains the following logic:
  ```c
  } else if ((mode->hsync_start - mode->hdisplay) > 20) {
      /* Maaaaaagic */
      u16 drq = (mode->hsync_start - mode->hdisplay) - 20;
      drq *= mipi_dsi_pixel_format_to_bpp(device->format) / 32;
      val = (SUN6I_DSI_TCON_DRQ_ENABLE_MODE | SUN6I_DSI_TCON_DRQ_SET(drq));
  }
  ```
  When front porch was erroneously set to 20, $(620 - 600) = 20$, which evaluated `> 20` to **False**. The `TCON_DRQ` register was written with 0 (**hardware DRQ data streaming was completely disabled**)!
  - **Resolution**: Setting front porch to exact $28$ evaluates $(628 - 600) = 28 > 20$ to **True**, calculating `drq = (28 - 20) * 24 / 32 = 6` and activating hardware DRQ flow control.

### 3.3 Root Cause #3: MIPI-DSI Video Mode & Negative Delay Underflow (Burst Trap)

- **Mode Incompatibility**:
  The BOE panel strictly requires Non-Burst Video Mode with Sync Pulses (`MIPI_DSI_MODE_VIDEO_SYNC_PULSE`). The driver was previously configured with `MIPI_DSI_MODE_VIDEO_BURST`.
- **Arithmetic Underflow Stall**:
  In `sun6i_mipi_dsi.c:462`, burst delay was calculated as:
  $$\text{delay} = \frac{(h_{\text{total}} - h_{\text{display}}) \times 150}{\text{clk} \times 8 / 1000} - 50 = \frac{48 \times 150}{424} - 50 = 16 - 50 = -34$$
  In `u16 delay`, $-34$ underflowed to **$65502$ (`0x0FDE`)**, written into `SUN6I_DSI_INST_LOOP_NUM_REG(0)` and stalling DSI instruction scanout; furthermore, burst mode set `HSA_HSE_DIS` and `HBP_DIS`, stripping hardware horizontal sync pulses.

### 3.4 Root Cause #4: DCS Low-Power Transmission Timing Constraint (LPTX before HSC)

- **Allwinner Physical Layer Protocol**:
  Allwinner DSI controllers must transmit DCS initialization commands while the physical D-PHY is in Low Power (LPTX) mode.
- **Mainline Ordering Inversion**:
  In `sun6i_dsi_encoder_enable()`, `sun6i_dsi_start(dsi, DSI_START_HSC)` was invoked before `drm_panel_prepare()`. This forced the controller into High Speed (HS) clock mode prematurely, causing all 25 DCS commands to be rejected by the PHY. The panel never initialized and remained blank.

### 3.5 Root Cause #5: TCON IRQ Storm & Framebuffer Color Depth Mismatch

1. **GINT0 IRQ Storm**:
   Upstream `sun4i_tcon.c:786` only cleared 4 mask bits. Unacknowledged status bits in `0xcc000601` (bits 9 and 0) created an endless 150kHz interrupt storm, consuming 100% CPU and timing out DRM atomic commits.
2. **Framebuffer Depth Mismatch**:
   Upstream `sun4i_drv.c:114` hardcodes `drm_fbdev_dma_setup(drm, 32)` creating 32bpp XRGB8888 (3.84MB). Userspace previously mapped only 16bpp (1.92MB) and wrote 16-bit words, leaving Alpha and Red as 0 (transparent black).

---

## 4. Implementation Details

### 4.1 Panel Driver: `panel-boe-wtl1026boe2l.c`
- **Non-Burst Sync Pulse Mode**:
  ```c
  dsi->mode_flags = MIPI_DSI_MODE_VIDEO |
                    MIPI_DSI_MODE_VIDEO_SYNC_PULSE |
                    MIPI_DSI_MODE_LPM;
  ```
- **Exact Vendor Blanking Geometry**:
  ```c
  static const struct drm_display_mode wtl1026boe2l_default_mode = {
      .clock = 53000,
      .hdisplay = 600,
      .hsync_start = 600 + 28,      /* HFP = 28 */
      .hsync_end = 600 + 28 + 8,    /* HSPW = 8 */
      .htotal = 648,                 /* HBP = 648 - 636 = 12 */
      .vdisplay = 1600,
      .vsync_start = 1600 + 18,     /* VFP = 18 */
      .vsync_end = 1600 + 18 + 2,   /* VSPW = 2 */
      .vtotal = 1634,                /* VBP = 1634 - 1620 = 14 */
  };
  ```
- **Panel Orientation Hook**: Added `enum drm_panel_orientation` and wired `.get_orientation` callback.

### 4.2 TCON Controller: `sun4i_tcon.c`
- **`SUNXI_DSI_PASSIVE_BUG` Compensation**:
  ```c
  block_space = mode->htotal * bpp / (tcon_div * 4);
  block_space -= mode->hdisplay + 40;
  ```
- **300MHz DE Clock Aligned `start_delay`**:
  ```c
  start_delay = 10 * mode->crtc_htotal * 300 / (mode->crtc_clock / 1000) / 8;
  ```
- **Full Lower 16-Bit Interrupt Flag Clearance**:
  ```c
  regmap_update_bits(tcon->regs, SUN4I_TCON_GINT0_REG, 0x0000ffff, 0);
  ```

### 4.3 DSI Controller: `sun6i_mipi_dsi.c`
- **Restored Proper Low-Power Transmission Sequence**:
  ```c
  if (dsi->panel)
      drm_panel_prepare(dsi->panel); /* Sends 25 DCS commands in LPTX */
  if (dsi->panel)
      drm_panel_enable(dsi->panel);
  sun6i_dsi_start(dsi, DSI_START_HSC); /* Start High-Speed Clock Lane */
  udelay(1000);
  sun6i_dsi_start(dsi, DSI_START_HSD); /* Start High-Speed Data Lane */
  ```

### 4.4 Userspace Dual-Mode 32bpp XRGB8888 Adaptation
- **`touch_cockpit.go` and `gud_v553.c`**:
  Queried `ioctl(fb, FBIOGET_VSCREENINFO)` at runtime. When `BitsPerPixel == 32`, mapped the full 3,840,000 bytes and converted RGB565 pixels to `0xff000000 | (r << 16) | (g << 8) | b`.

---

## 5. Delivery Artifacts & Verification

All compiled release assets are staged in `output/` (git-ignored build artifacts):

| File | Size | Description |
| :--- | :--- | :--- |
| `zImage` | 5.8MB | Linux 6.12 LTS kernel with all display pipeline fixes |
| `sunxi.dtb` | 18KB | Board device tree with corrected blanking & `rotation = <90>` |
| `touch_cockpit` | 2.3MB | 32bpp adaptive touch macro pad HUD |
| `gud_v553` | 18KB | 32bpp adaptive GUD USB Display blitter |
| `MANIFEST.sha256` | 473B | Cryptographic SHA256 release checksums |

Running `./host-tools/update_kernel_sd.sh` flashes the entire stack to the MicroSD card in under 5 seconds.
