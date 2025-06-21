# dcode - Dialog Code

A Go wrapper for Claude Code that adds macOS permission dialog handling and seamless option passthrough.

![image](https://github.com/user-attachments/assets/85fdf549-a7a6-465c-a00d-e7e8df2a1ff4)

## ✨ Features

- **🔄 Seamless Passthrough** - All Claude Code options work exactly as expected
- **💬 Native macOS Dialogs** - Permission prompts appear as native dialog boxes

## 🚀 Quick Start

```bash
# Install from GitHub
go install github.com/takahirom/dialog-code/cmd/dcode@latest

# Or build locally
go build -o dcode ./cmd/dcode
or
bazel build //cmd/dcode:dcode && ./bazel-bin/cmd/dcode/dcode_/dcode

# Use just like claude
dcode
dcode --help
dcode --resume
```
