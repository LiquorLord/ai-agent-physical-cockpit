# 贡献指南

感谢您对AI-Agent 物理驾驶舱项目的兴趣！

# 开始贡献

1. 将仓库fork到您的GitHub账户。
2. 克隆您的分支仓库：
   ```
   git clone https://github.com/yourusername/AI-Agent-Physical-Cockpit.git
   ```

# 开发环境设置

- **固件**：Go应用程序使用`GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build`。
- **C应用程序**：使用`arm-linux-gnueabihf-gcc`进行编译。
- **Python主机工具**：确保安装了Python 3.6+，并按照README中的说明运行这些工具。

# 编码规范

- Go代码应使用`gofmt`格式化。
- Shell脚本应通过`shellcheck`检查。
- 文档：在`docs/`目录下的每个`.md`文件都必须有一个对应的`_zh.md`翻译，以支持双语（英语/中文）。

# 提交信息

遵循常规提交规范：
- `feat`: 新功能
- `fix`: 修复错误
- `docs`: 文档更改
- `chore`: 维护任务

示例：`feat: 添加新宏功能`

# 拉取请求流程

1. 确保您的分支与上游仓库保持同步。
2. 提交一个拉取请求，详细说明所做的更改及其目的。

# 报告问题

请通过GitHub Issues报告问题，并尽可能提供详细的步骤以重现遇到的任何错误。

# 许可证声明

本项目采用MIT许可证。除非另有说明，否则贡献将被视为在该许可证下发布。