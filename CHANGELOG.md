# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## [1.0.1] - 2026-10-27

### Fixed

- Fixed touch button input on mainline Linux 6.12 by adding runtime adaptive
  dual-mode coordinate mapping in `touch_cockpit` (handles raw Goodix GT911
  portrait $600 \times 1600$ matrix rotation to landscape).

## [1.0.0] - 2026-10-22

### Added

- Initial release of the AI-Agent Physical Cockpit: 100% plug-and-play Linux 6.12
  GUD USB display + HID macro pad with on-screen agent picker.
