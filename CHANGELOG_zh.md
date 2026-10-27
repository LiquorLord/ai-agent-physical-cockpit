# 更新日志

本项目所有值得注意的变更都记录在此。格式遵循
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，并遵循
[语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.1] - 2026-10-27

### 修复

- 修复主线 Linux 6.12 下右侧触控按键无响应问题：在 `touch_cockpit` 中引入自适应双模坐标解算引擎，自动处理汇顶 GT911 芯片原生竖屏阵列（$600 \times 1600$）到横屏的 $90^\circ$ 几何旋转映射。

## [1.0.0] - 2026-10-22

### 新增

- AI-Agent 物理驾驶舱初始发布：100% 即插即用的 Linux 6.12 GUD USB 显示 + HID
  宏垫，带屏幕代理选择器。
