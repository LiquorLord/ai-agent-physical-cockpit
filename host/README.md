[English](README.md) | [简体中文](README_zh.md)

# Host-side Display Mode Hook (REQUIRED for fullscreen switching)

The cockpit has two display modes (`1200x600` dual-zone and `1600x600` fullscreen). The V553
board only **sends a keystroke**; the **host must change the video mode**. Without this hook the
5th button and the top-right hot zone appear to do nothing.

Contract: the board injects USB HID usage `0x73` (F24). Linux reports it as keycode **202**,
which has **no keysym** in the stock evdev XKB map, so `bindsym F24` never matches.
**Bind by keycode (`bindcode 202`), not by keysym.**

| Host | Script | Binding |
|------|--------|---------|
| X11 + i3 | `toggle_gud_mode_x11.sh` | `bindcode 202 exec --no-startup-id /usr/local/bin/toggle_gud_mode.sh` |
| Wayland + sway | `toggle_gud_mode_sway.sh` | `bindcode 202 exec /usr/local/bin/toggle_gud_mode.sh` |

Install:

```bash
sudo install -m755 host/toggle_gud_mode_x11.sh /usr/local/bin/toggle_gud_mode.sh   # or the sway one
# add the matching bindcode line to ~/.config/i3/config or ~/.config/sway/config, then reload
```

Notes:
- The X11 script defaults to `DISPLAY=:1` and places the GUD output `--below HDMI-1`; edit both for your layout.
- The board and the host each keep their own idea of the current mode. If one `F24` is lost they
  desynchronize; restart `touch_cockpit` on the board to reset it to dual-zone.
