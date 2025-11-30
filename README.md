# dcode - Dialog Code

A macOS permission dialog hook for Claude Code using the PermissionRequest hook system.

<img width="1552" height="1232" alt="image" src="https://github.com/user-attachments/assets/51e4a423-7365-43ee-a93f-c2b319768a26" />

## Features

- **Native macOS Dialogs** - Permission prompts appear as native dialog boxes
- **Configurable Timeout** - Auto-denies if user doesn't respond (default: 60 seconds)
- **Claude Code Hook Integration** - Uses the official PermissionRequest hook system

## Installation

```bash
# Build with Bazel
bazel build //cmd/dcode:dcode

# The binary will be at:
# bazel-bin/cmd/dcode/dcode_/dcode
```

## Options

```bash
--timeout=N    Set dialog timeout in seconds (default: 60)
```

Example with custom timeout:
```bash
/path/to/dcode --timeout=30
```

## Setup

Add the hook to your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/bazel-bin/cmd/dcode/dcode_/dcode --timeout=30",
            "timeout": 120
          }
        ]
      }
    ]
  }
}
```

Note: The `timeout` in hooks config is Claude Code's hook timeout. The `--timeout=30` is dcode's dialog timeout.

## How It Works

1. Claude Code sends a PermissionRequest JSON to stdin
2. dcode shows a native macOS dialog with "Allow" and "Deny" buttons
3. User responds (or dialog times out after 60 seconds)
4. dcode outputs the decision as JSON to stdout

### Input Format

```json
{
  "hook_event_name": "PermissionRequest",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm run build"
  }
}
```

### Output Format

Allow:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow"
    }
  }
}
```

Deny:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "deny",
      "interrupt": false,
      "message": "User denied the request"
    }
  }
}
```

Timeout:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "deny",
      "interrupt": false,
      "message": "User did not respond within 60 seconds"
    }
  }
}
```

## Supported Tools

- **Bash** - Shows command to execute
- **Edit** - Shows file path being edited
- **Write** - Shows file path being written
- **Other tools** - Shows tool name

## Development

```bash
# Run tests
bazel test //...

# Build
bazel build //cmd/dcode:dcode
```
