[1.0.0] - 2026-10-20
### Added
- Finalized 100% plug-and-play Linux 6.12 GUD USB display + HID release (v1.0)
- Enabled Bluetooth (UART3) and DMIC audio sound card in DTS and kernel config
- Decoupled device into standalone USB Display + HID macro pad with on-screen agent picker and SD card JSON customization
- Added multi-agent profile engine (AGY/CLAUDE/DSH/SHELL) and unified CLI profile switcher
- Integrated DRM display pipeline, panel driver, dual-zone userspace stack, and multimodal roadmap
- Added Linux 6.12 baseline, board DTS, upstream patches, and built zImage/DTB

### Changed
- Updated kanbans with Phase 1 completion and Linux 6.12 root shell bringup

### Fixed
- Resolved multi-touch protocol A pressure tracking and hardened non-tty PTY bridge
- Aligned GT911 coordinates 1:1 to native 1600x600 landscape
- Adapted touch_cockpit to Linux MT Protocol A and enabled raw event logging
- Overhauled init_usb_monitor with persistent debug logging and factory UDC bind
- Implemented robust 10-attempt UDC bind polling to guarantee D+ pullup activation on cold boot

### Removed
- Cleanly decoupled and archived visual gesture engine behind feature flag

---

## Pre-release history

### Added
- Migrated agy CLI to dual-zone screen and deployed 4-button tactile console
- Implemented telemetry HUD, unified executive CLI (./cockpit), and host udev integration
- Implemented dual-zone display streaming and zero-typing PTY master
- Implemented closed fist ✊ STOP gesture recognition with red border glow
- Synchronized 24px feedback glow and Action Bar flash with gesture recognition
- Switched to user-facing 1080p camera (/dev/video8) with YCbCr skin gesture detection
- Implemented V4L2 multi-planar streaming and downsampled classifier
- Added touch_cockpit action bar, gesture_detector V4L2 classifier, and updated boot orchestration
- Established standalone bilingual project kanban and end-of-day handover state
- Added 3-in-1 composite USB gadget (ADB+NCM+HID) and hardware page flip ioctl in fb_receiver
- Implemented USB Composite Gadget (CDC-NCM + HID) and send_hid_key injector
- Restructured project directory, separated bilingual documentation, and established repository baseline

### Changed
- Launched dedicated allwinner-build Incus container and configured ARM toolchain

### Fixed
- Resolved multi-touch protocol A pressure tracking and hardened non-tty PTY bridge
- Aligned GT911 coordinates 1:1 to native 1600x600 landscape
- Adapted touch_cockpit to Linux MT Protocol A and enabled raw event logging
- Overhauled init_usb_monitor with persistent debug logging and factory UDC bind
- Implemented robust 10-attempt UDC bind polling to guarantee D+ pullup activation on cold boot

### Removed
- Cleanly decoupled and archived visual gesture engine behind feature flag