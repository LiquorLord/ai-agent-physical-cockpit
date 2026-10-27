# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-10-27

### Fixed

- Fixed touch button input on mainline Linux 6.12 by adding runtime adaptive
  dual-mode coordinate mapping in `touch_cockpit` (handles raw Goodix GT911
  portrait $600 \times 1600$ matrix rotation to landscape).

## [1.0.0] - 2026-10-22

### Added

- Initial release of the AI-Agent Physical Cockpit: 100% plug-and-play Linux 6.12
  GUD USB display + HID macro pad with on-screen agent picker.
