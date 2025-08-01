# dcode - Dialog Code

A Go wrapper for Claude Code that adds macOS permission dialog handling and seamless option passthrough.

**Vibe coded** - This project was created with AI assistance.

![image](https://github.com/user-attachments/assets/b217ab55-0586-40ec-bc09-82d37e9b3b19)

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
Waits N seconds for user intervention before automatically rejecting.

```bash
dcode --auto-reject-wait=10  # Wait 10 seconds for user input
```

**Features**:
- If user presses any choice key (1, 2, 3) or Enter during wait period, cancels auto-reject
- If no user input is detected, automatically rejects after specified time
- Provides safety window for user intervention while ensuring eventual rejection

**Use case**: Semi-automated environments where you want to give users a chance to intervene but ensure commands don't hang indefinitely.
