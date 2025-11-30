# PermissionRequest Hook - Requirements & TDD Checklist

## Overview

A Claude Code PermissionRequest hook that shows a dialog for tool execution approval.

---

## TDD Checklist

### Phase 1: Basic Allow/Deny Flow

- [x] **1.1** When user approves a Bash command in dialog, outputs allow JSON
- [x] **1.2** When user denies a Bash command in dialog, outputs deny JSON with interrupt:false
- [x] **1.3** When user denies with a message, the message is included in deny JSON

### Phase 2: Dialog Display

- [x] **2.1** Dialog shows the tool name (e.g., "Bash")
- [x] **2.2** Dialog shows the command content (e.g., "npm run build")
- [x] **2.3** Dialog shows "Allow" and "Deny" buttons

### Phase 3: Error Handling

- [x] **3.1** When stdin contains invalid JSON, exits with code 1
- [x] **3.2** When stdin is empty, exits with code 0 (default behavior)

### Phase 4: Various Tools

- [ ] **4.1** Works with Edit tool (shows file_path and changes)
- [ ] **4.2** Works with Write tool (shows file_path)
- [ ] **4.3** Works with unknown tools (shows raw tool_input)

---

## JSON Format Reference

### Input (stdin)
```json
{
  "hook_event_name": "PermissionRequest",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm run build"
  }
}
```

### Output - Allow
```json
{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}
```

### Output - Deny
```json
{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"User rejected","interrupt":false}}}
```

---

## Progress Log

| Test | Date | Commit |
|------|------|--------|