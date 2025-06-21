# dcode - Dialog Code

A Go wrapper for Claude Code that adds macOS permission dialog handling and seamless option passthrough.

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
