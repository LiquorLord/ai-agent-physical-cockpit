# Legacy Linux 4.9 Cockpit Golden Archive / Linux 4.9 座舱黄金封板归档

## 1. Overview / 概述
This archive preserves the complete, verified, and operational production milestone of the **Allwinner V553 (`sun8iw21p1`) Smart Cockpit** running the patched Linux 4.9.191 kernel prior to modern mainline Linux (6.x) migration.
本归档保存了全志 V553 智能座舱在迁移至现代主线内核（6.x）前，在 Linux 4.9.191 内核下完整调通且已通过硬件验证的黄金成果。

## 2. Verified Capabilities / 已验证核心特性
- **SPI NOR Flash**: 100% factory backup (`backup_factory_16MB.bin`) and unbrick FEL flashing pipeline.
- **USB Composite Gadget**: OpenMoko GUD USB Display (1200x600 @ `/dev/dri/cardX`) + Single-Endpoint USB HID Hardware Keyboard (`/dev/input/eventX`).
- **Physical Keyboard Injection**: Goodix GT911 touch events mapped to hardware scancodes (`PROCEED`, `GRILL-ME`, `PLAN`, `STOP`).
- **Display Engine**: BOE 10.26" 1600x600 MIPI-DSI 4-lane display locked with custom bootsplash and dual-zone telemetry HUD.
- **Kernel ABI**: CMA struct alignment resolved (`CONFIG_DMA_CMA` disabled) ensuring zero kernel crash.

## 3. Directory Contents / 归档清单
- `kernel/`:
  - `zImage`: Linux 4.9.191 kernel image (hash verified).
  - `sunxi.dtb`: Device tree binary with display, touch, and storage nodes.
  - `config_v4.9_cockpit`: Production kernel compilation configuration.
  - `0001-v553-musb-f_hid-single-endpoint.patch`: MUSB pure IN interrupt endpoint patch.
- `userland/`:
  - `touch_cockpit`: ARMv7 Go static interaction daemon.
  - `gud_v553`: GUD direct-framebuffer streaming daemon.
  - `show_splash`: Low-level DE2.0 bootsplash display loader.
  - `switch_to_gud.sh`: Fast composite gadget activation script.
  - `init_usb_monitor.sh`: MicroSD dual-boot arbitration script.
  - `S12usb_gadget`: Init gadget service script.
- `system_images/`:
  - `backup_factory_16MB.bin`: Factory SPI NOR Flash 16MB image.
- `MANIFEST.sha256`: Cryptographic checksum manifest.

## 4. Rollback Instruction / 一键恢复回滚指引
To restore the device back to this Linux 4.9 state:
如需将设备恢复至此 4.9 稳定态：
```bash
# 1. Mount SD card boot partition
mount /dev/mmcblk0p1 /boot

# 2. Copy archived kernel and DTB
cp archive/legacy_v4.9_cockpit/kernel/zImage /boot/zImage
cp archive/legacy_v4.9_cockpit/kernel/sunxi.dtb /boot/sunxi.dtb
sync && reboot
```
