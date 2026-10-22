# Contributing

Thanks for your interest in contributing to AGY Physical Cockpit!

# Getting Started

1. Fork the repository to your GitHub account.
2. Clone your forked repository:
   ```
   git clone https://github.com/yourusername/AGY-Physical-Cockpit.git
   ```

# Development Setup

- **Firmware**: Go applications use `GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build`.
- **C Applications**: Use `arm-linux-gnueabihf-gcc` for compilation.
- **Python Host Tools**: Ensure Python 3.6+ is installed and run the tools as specified in the README.

# Coding Conventions

- Go code should be formatted using `gofmt`.
- Shell scripts should pass `shellcheck`.
- Documentation: Every `.md` file under `docs/` must have a corresponding `_zh.md` translation for bilingual support (English/Zhongwen).

# Commit Messages

Follow conventional commits:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `chore`: Maintenance tasks

Example: `feat: add new macro functionality`

# Pull Request Process

1. Ensure your branch is up-to-date with the upstream repository.
2. Submit a pull request detailing the changes made and their purpose.

# Reporting Issues

Please report issues via GitHub Issues, providing as much detail as possible including steps to reproduce any bugs encountered.

# License Note

This project is licensed under the MIT License. Contributions are assumed to be released under this license unless otherwise stated.