# 全志 V553 现代主线 Linux 6.12 屏幕驱动移植与根因修复权威技术指南

> **语言导航 / Language Navigation**:
> - **中文版本**: [V553_MAINLINE_DISPLAY_PORTING_GUIDE_zh.md](V553_MAINLINE_DISPLAY_PORTING_GUIDE_zh.md)
> - **English Version**: [V553_MAINLINE_DISPLAY_PORTING_GUIDE.md](V553_MAINLINE_DISPLAY_PORTING_GUIDE.md)

---

## 1. 概述与问题背景

在将全志 V553（`sun8iw21p1`）智能座舱终端从陈旧的厂商 BSP Linux 4.9.191 内核移植到现代主线 Linux 6.12 LTS 时，遇到了最典型的硬件调试难题：
**设备通电开机约 8~10 秒后，屏幕背光成功点亮，呈现类似古老 CRT 显示器的微弱圆角光晕，但屏幕没有任何图形输出（白屏/无信号状态）。**

经过对 Linux 4.9 原厂内核（DISP2 架构）与 Linux 6.12 主线内核（DRM/KMS 架构）全链路源码的第一性原理深度比对与逆向工程分析，定位并攻克了导致该故障的 **五大核心根因**。本指南系统性地记录这一技术攻坚全过程。

---

## 2. 4.9 DISP2 与 6.12 DRM 显示管线架构对比

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Linux 4.9 厂商内核 (DISP2)                      │
│                                                                        │
│   Display Engine 2 ──► TCON0 (tcon0_cfg_mode_tri) ──► DSI (LPTX DCS)   │
│   (300MHz 时钟)        (SUNXI_DSI_PASSIVE_BUG: /16)  │ (非突发同步脉冲)│
│                                                      ▼                 │
│   /dev/fb0 (16bpp RGB565) ◄───────────────────────── BOE 10.26" 屏幕   │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│                        Linux 6.12 现代内核 (DRM/KMS)                   │
│                                                                        │
│   DE2 (mixer0) ──► tcon_top ──► tcon_lcd0 ──► DSI Transmitter ──► Panel│
│   (CRTC Atomic)    (时钟树路由) (0x0000ffff   (SYNC_PULSE +      (DCS在│
│                                 全量清中断)     TCON DRQ 流控)     LP态)│
│                                                      │                 │
│   /dev/fb0 (32bpp XRGB8888) ◄────────────────────────┴─────────────────┘
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 五大核心致命根因与第一性原理深度剖析

### 3.1 根因一：全志硅片缺陷 `SUNXI_DSI_PASSIVE_BUG` 与空隙计算错误

- **硬件事实与物理现象**：
  在全志 V553（`sun8iw21`）及同代的 `sun50iw10` 芯片中，存在名为 `SUNXI_DSI_PASSIVE_BUG` 的硅片硬件特性：**TCON0 内部的 DSI FIFO 取数逻辑固定以 4-lane 速度从内存读取数据，不受外部物理 DSI 通道数（2-lane）影响**。
- **4.9 原厂处理（`de_lcd_sun50iw10.c:899`）**：
  ```c
  #if defined(SUNXI_DSI_PASSIVE_BUG)
      /* 强制将除数锁定为 (tcon_div * 4)，而不是物理 lane 数 (2) */
      lcd_dev[sel]->tcon0_cpu_tri0.bits.block_space =
          panel->lcd_ht * dsi_pixel_bits / (tcon_div * 4) - panel->lcd_x - 40;
  #endif
  ```
  代入参数：$648 \times 24 / (4 \times 4) - 600 - 40 = 972 - 640 = \mathbf{332}$。
- **6.12 主线原版错误（`sun4i_tcon.c:395`）**：
  主线上游公式除以了物理通道数 `(tcon_div * lanes)`：
  代入物理 2-lane：$648 \times 24 / (4 \times 2) - 600 - 40 = 1944 - 640 = \mathbf{1304}$！
  - **严重后果**：TCON0 在两行之间的空隙等待周期被放大了整整 4 倍（1304 周期），超出了消隐区的物理容限，导致 TCON FIFO 严重欠载饥饿，行扫描信号瞬间中断。

### 3.2 根因二：消隐区 Porch 语义差异与 TCON DRQ 门限（`/* Maaaaaagic */` 谜题）

- **参数语义的代沟**：
  在全志 4.9 DTS 中，`lcd_hbp = <0x14>`（20）**并非纯后走（Back Porch），而是包含脉冲宽度 `lcd_hspw = <8>` 的总和**。
  - 真正纯后走（Pure HBP）：$20 - 8 = \mathbf{12}$。
  - 真正纯前走（Pure HFP）：$648 - 600 - 20 = \mathbf{28}$。
- **主线驱动阈值判定（`sun6i_mipi_dsi.c:440`）**：
  主线开发者在此留下了著名的 `/* Maaaaaagic */` 注释：
  ```c
  } else if ((mode->hsync_start - mode->hdisplay) > 20) {
      /* Maaaaaagic */
      u16 drq = (mode->hsync_start - mode->hdisplay) - 20;
      drq *= mipi_dsi_pixel_format_to_bpp(device->format) / 32;
      val = (SUN6I_DSI_TCON_DRQ_ENABLE_MODE | SUN6I_DSI_TCON_DRQ_SET(drq));
  }
  ```
  此前主线参数误将前走设定为 20，计算 $(620 - 600) = 20$，导致 `> 20` 判定为 **False**！`TCON_DRQ` 寄存器被写为 0，**硬件数据请求（DRQ）流控完全未开启**！
  - **修正方案**：将前走设为精准的 $28$，$(628 - 600) = 28 > 20$ 成立，计算 `drq = (28 - 20) * 24 / 32 = 6`，硬件 DRQ 成功激活。

### 3.3 根因三：MIPI-DSI 视频模式与负延时下溢（Burst 突发模式陷阱）

- **模式冲突**：
  京东方此屏必须工作在带有同步脉冲的非突发视频模式（`MIPI_DSI_MODE_VIDEO_SYNC_PULSE`）。此前驱动配置了 `MIPI_DSI_MODE_VIDEO_BURST`。
- **负延时下溢崩溃**：
  在主线 `sun6i_mipi_dsi.c:462` 中计算突发延时：
  $$\text{delay} = \frac{(h_{\text{total}} - h_{\text{display}}) \times 150}{\text{clk} \times 8 / 1000} - 50 = \frac{48 \times 150}{424} - 50 = 16 - 50 = -34$$
  由于变量为 `u16 delay`，$-34$ 发生算术下溢变为 **$65502$（`0x0FDE`）**，写死在 `SUN6I_DSI_INST_LOOP_NUM_REG(0)` 中，导致 DSI 指令发射循环彻底冻结；且 Burst 模式设置了 `HSA_HSE_DIS` 和 `HBP_DIS`，剥离了硬件同步脉冲。

### 3.4 根因四：DCS 寄存器低功耗时序约束（LPTX before HSC）

- **全志物理层时序规则**：
  全志 DSI 控制器向面板发送 25 条 DCS 寄存器配置序列时，物理层通道必须处于低功耗（LPTX）状态。
- **主线倒置缺陷**：
  此前在 `sun6i_dsi_encoder_enable()` 中过早调用 `sun6i_dsi_start(dsi, DSI_START_HSC)` 拉起了高速时钟道，导致 DSI 控制器提前进入 High Speed（HS）状态，所有的 `mipi_dsi_dcs_write()` 指令被物理层拒绝传输，面板未初始化，仅有背光加电。

### 3.5 根因五：TCON 中断雪崩与 Framebuffer 32bpp 色深错位

1. **GINT0 中断雪崩**：
   主线 `sun4i_tcon.c:786` 仅对 4 个掩码位写 0，状态寄存器中未清零标志（`0xcc000601` 中的 bit 9 与 bit 0）导致 150kHz 高频中断死循环，抢占全部 CPU 并阻断 DRM Atomic Commit。
2. **Framebuffer 颜色深度错位**：
   主线 `sun4i_drv.c:114` 硬编码调用 `drm_fbdev_dma_setup(drm, 32)` 创建 32 位真彩色 `/dev/fb0`（3.84MB），而旧版用户态程序按 16 位（1.92MB）映射与写入，高 16 位为 0（全黑透明），导致图形不可见。

---

## 4. 落地修复实现方案

### 4.1 面板驱动：`panel-boe-wtl1026boe2l.c`
- **非突发同步脉冲模式**：
  ```c
  dsi->mode_flags = MIPI_DSI_MODE_VIDEO |
                    MIPI_DSI_MODE_VIDEO_SYNC_PULSE |
                    MIPI_DSI_MODE_LPM;
  ```
- **精确原厂消隐区时序**：
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
- **挂接面板旋转属性**：在结构体中集成 `enum drm_panel_orientation` 并实现 `.get_orientation` 回调。

### 4.2 TCON 控制器：`sun4i_tcon.c`
- **引入 `SUNXI_DSI_PASSIVE_BUG` 补偿**：
  ```c
  block_space = mode->htotal * bpp / (tcon_div * 4);
  block_space -= mode->hdisplay + 40;
  ```
- **300MHz DE 时钟对齐 `start_delay`**：
  ```c
  start_delay = 10 * mode->crtc_htotal * 300 / (mode->crtc_clock / 1000) / 8;
  ```
- **全量清除中断低 16 位状态标志**：
  ```c
  regmap_update_bits(tcon->regs, SUN4I_TCON_GINT0_REG, 0x0000ffff, 0);
  ```

### 4.3 DSI 控制器：`sun6i_mipi_dsi.c`
- **恢复正确的低功耗传输顺序**：
  ```c
  if (dsi->panel)
      drm_panel_prepare(dsi->panel); /* 发送 25 条 DCS 指令 (LPTX) */
  if (dsi->panel)
      drm_panel_enable(dsi->panel);
  sun6i_dsi_start(dsi, DSI_START_HSC); /* 启动高速时钟道 */
  udelay(1000);
  sun6i_dsi_start(dsi, DSI_START_HSD); /* 启动高速数据道 */
  ```

### 4.4 用户态全栈：32bpp XRGB8888 动态适配
- **`touch_cockpit.go` 与 `gud_v553.c`**：
  通过 `ioctl(fb, FBIOGET_VSCREENINFO)` 动态探测显存深度。当检测到 `BitsPerPixel == 32` 时，映射完整 3,840,000 字节，并将 RGB565 颜色通过查找表/位移转换为 `0xff000000 | (r << 16) | (g << 8) | b` 硬件级写入。

---

## 5. 验证与交付物校验

所有更新已统一编译打包入 `output/` (git-ignored build artifacts)：

| 文件名 | 大小 | 说明 |
| :--- | :--- | :--- |
| `zImage` | 5.8MB | Linux 6.12 LTS 生产内核（集成全部修复） |
| `sunxi.dtb` | 18KB | 权威设备树（集成消隐区修正与 `rotation = <90>`） |
| `touch_cockpit` | 2.3MB | 32bpp 自适应宏按键 HUD（支持玻璃触控切模） |
| `gud_v553` | 18KB | 32bpp 自适应 GUD USB 显卡硬件级 Blitter |
| `MANIFEST.sha256` | 473B | 不可篡改哈希签名校验清单 |

执行 `./host-tools/update_kernel_sd.sh` 即可一键完成全栈写入与安全交付。
