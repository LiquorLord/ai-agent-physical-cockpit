[English](README.md) | [简体中文](README_zh.md)

# 主机端显示模式钩子（全屏切换必须配合）

驱动器有两种显示模式（`1200x600` 双屏、`1600x600` 全屏）。V553 板子只负责**发送一个按键**，
**切换视频模式必须由主机完成**。没有这个钩子，第 5 个按钮和右上角热区看起来就像没有反应。

约定：板子注入 USB HID 用法码 `0x73`（F24）。Linux 把它报告为键码 **202**，
而标准 evdev XKB 映射里该键码**没有 keysym**，所以 `bindsym F24` 永远匹配不上。
**必须按键码绑定（`bindcode 202`），不能按键名绑定。**

| 主机 | 脚本 | 绑定 |
|------|------|------|
| X11 + i3 | `toggle_gud_mode_x11.sh` | `bindcode 202 exec --no-startup-id /usr/local/bin/toggle_gud_mode.sh` |
| Wayland + sway | `toggle_gud_mode_sway.sh` | `bindcode 202 exec /usr/local/bin/toggle_gud_mode.sh` |

安装：

```bash
sudo install -m755 host/toggle_gud_mode_x11.sh /usr/local/bin/toggle_gud_mode.sh   # sway 用另一个脚本
# 在 ~/.config/i3/config 或 ~/.config/sway/config 加入对应的 bindcode 行，然后 reload
```

注意：
- X11 脚本默认 `DISPLAY=:1`，并把 GUD 输出放在 `HDMI-1` 下方（`--below HDMI-1`），请按你的布局修改。
- 板子和主机各自记录当前模式；如果某次 `F24` 丢失两边会失步，在板子上重启 `touch_cockpit` 可复位为双屏。
