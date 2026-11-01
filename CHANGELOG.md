# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Dual display modes: Dual-Zone `1200x600@60Hz` (left desktop, right 400x600 touch Action Bar)
  and Fullscreen `1600x600@57.11Hz` (whole panel touch-active). A 5th button under the four
  macro buttons enters Fullscreen; a transparent top-right hot zone (`X>=1450, Y<=100`) returns.
- `gud_v553` follows the host-selected mode dynamically and renders the full panel when 1600 wide.
- `host/` hook scripts for X11+i3 and Wayland+sway. **The host hook is required**: the board only
  injects HID `F24` (`0x73`); the host must switch the video mode. See `host/README.md`.

### Fixed

- HID report descriptor max usage raised from `0x65` to `0x73`; the host previously dropped `F24`.
- The hook must bind keycode 202 (`bindcode 202`); `F24` has no keysym in the default XKB map.
- `scripts/uart_cmd.py` aborts stale quoted input with Ctrl+C before each command.

## [1.0.2] - 2026-10-28

### Fixed

- Maintained a persistent, mutex-protected `/dev/hidg0` file descriptor in
  `touch_cockpit` instead of reopening and tearing down the gadget character
  device on every keystroke, preventing `f_hid_release` from resetting endpoint
  FIFOs and dropping trailing packets.
- Introduced a 200ms pre-settle delay before Enter scancode injection and extended
  key hold duration to 50ms with 30ms post-release delay. This ensures terminal
  event loops, multiplexers (tmux), and interactive CLI frameworks (e.g., `agy`,
  Ink/React CLI, readline) finish processing preceding character inputs and
  settle into an idle input state before executing Enter.
- Added explicit support for Carriage Return (`\r`) in HID key injection and
  guaranteed automatic trailing newline append for `proceed` action buttons.
- Configured direct high-speed prebuilt golden SD production image mirrors (official GitHub release + domestic high-speed proxy mirror) with SHA256 checksums in README, allowing end users to start using the cockpit immediately without compiling from source.

## [1.0.1] - 2026-10-27

### Fixed

- Fixed touch button input on mainline Linux 6.12 by adding runtime adaptive
  dual-mode coordinate mapping in `touch_cockpit` (handles raw Goodix GT911
  portrait $600 \times 1600$ matrix rotation to landscape).

## [1.0.0] - 2026-10-22

### Added

- Initial release of the AI-Agent Physical Cockpit: 100% plug-and-play Linux 6.12
  GUD USB display + HID macro pad with on-screen agent picker.
