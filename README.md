# dcode - Dialog Code

A Go wrapper for Claude Code that adds macOS permission dialog handling and seamless option passthrough.

<img width="1552" height="1232" alt="image" src="https://github.com/user-attachments/assets/51e4a423-7365-43ee-a93f-c2b319768a26" />


## ✨ Features

- **🔄 Seamless Passthrough** - All Claude Code options work exactly as expected
- **💬 Native macOS Dialogs** - Permission prompts appear as native dialog boxes
- **🛡️ Auto-Reject Modes** - Automatically reject unauthorized commands for unattended operation

## 🚀 Quick Start

```bash
# Install from GitHub
go install github.com/takahirom/dialog-code/cmd/dcode@latest

# Use just like claude
dcode
dcode --help
dcode --resume
dcode --debug  # Enable debug logging (creates debug_output.log)
```

## 🛡️ Auto-Reject Options

For unattended operation or enhanced security, dcode provides auto-reject modes:

### `--auto-reject`
Immediately rejects all permission prompts without user interaction.

```bash
dcode --auto-reject
```

**Use case**: Fully automated environments where no user interaction is desired.

### `--auto-reject-wait=N`
Shows a dialog with timeout information and waits N seconds for user intervention before automatically rejecting.

```bash
dcode --auto-reject-wait=10  # Wait 10 seconds for user input
```

**Features**:
- **📱 Interactive Dialog**: Shows native macOS dialog indicating auto-reject timeout
- **⏱️ Timeout Protection**: Automatically rejects after specified time if no user response
- **🎯 User Override**: User can choose any option during wait period to override auto-reject
- **🔒 Graceful Fallback**: If dialog fails or times out, safely defaults to rejection

**Use case**: Semi-automated environments where you want to give users a visual prompt and chance to intervene but ensure commands don't hang indefinitely.
