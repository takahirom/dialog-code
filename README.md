# dcode - Dialog Code

A Go wrapper for Claude Code that adds permission dialog handling and unknown option passthrough.

## Project Structure

This project has been refactored into a modular architecture with proper separation of concerns:

```
dcode/
├── MODULE.bazel                 # Bazel configuration (Bzlmod)
├── BUILD.bazel                  # Root build file
├── cmd/dcode/                   # Main binary
│   ├── main.go
│   └── BUILD.bazel
├── internal/
│   ├── types/                   # Type definitions & state management
│   │   ├── types.go
│   │   ├── types_test.go
│   │   └── BUILD.bazel
│   ├── choice/                  # Choice processing logic
│   │   ├── choice.go
│   │   ├── choice_test.go
│   │   └── BUILD.bazel
│   └── dialog/                  # Dialog UI handling
│       ├── dialog.go
│       └── BUILD.bazel
└── go.mod
```

## Features

### ✅ Working Features

1. **Unknown Option Passthrough**
   - `dcode --help` - Shows Claude Code help
   - `dcode --resume` - Shows session selection
   - All unknown options are passed through to the original `claude` command

2. **Permission Dialog System**
   - Detects Claude Code permission prompts
   - Collects available choices (Yes, Yes and don't ask again, No)
   - Shows macOS dialog with actual choices
   - Automatically responds based on user selection

3. **Auto-approve Mode**
   - `dcode -auto-approve` - Automatically approves all prompts
   - `dcode -strip-colors` - Removes ANSI color codes

## Build

### Using Go (Traditional)

```bash
# From project root
cd cmd/dcode
go build -o ../../dcode .

# Or build in place
go build -o dcode ./cmd/dcode
```

### Using Bazel (Recommended)

```bash
# Build the binary
bazel build //cmd/dcode:dcode

# The binary will be available at:
# ./bazel-bin/cmd/dcode/dcode_/dcode

# Copy to project root for easy access
cp ./bazel-bin/cmd/dcode/dcode_/dcode ./dcode
```

## Usage

### Basic Usage
```bash
# Interactive mode (recommended for permission dialogs)
dcode

# Pass through unknown options
dcode --resume
dcode --help
dcode --model sonnet

# Auto-approve mode
dcode -auto-approve
```

### Testing Permission Dialogs
1. Run `dcode` interactively
2. Type: `"Create a TEST_FILE file. I want to verify the permission check."`
3. Permission dialog should appear on macOS

## Testing

### Run Tests with Bazel

```bash
# Run all tests
bazel test //...

# Run specific package tests
bazel test //internal/types:types_test
bazel test //internal/choice:choice_test

# Run tests with verbose output
bazel test //... --test_output=all
```

### Run Tests with Go

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/types
go test ./internal/choice

# Run with verbose output
go test -v ./...
```

## Development

### Prerequisites

- **Go 1.24.4+** - Required for building with Go
- **Bazel 8.0+** - Required for Bazel builds (uses Bzlmod)
- **Claude Code** - The underlying claude binary must be installed

### Architecture

The project follows a clean modular architecture:

- **types**: Core data structures, interfaces, and state management
- **choice**: Logic for processing and selecting the best choice from options
- **dialog**: UI layer for displaying permission dialogs (macOS AppleScript)
- **cmd/dcode**: Main application entry point and I/O handling

### Adding New Features

1. Determine which package the feature belongs to
2. Add implementation to the appropriate package
3. Write tests in the corresponding `*_test.go` file
4. Update BUILD.bazel if adding new dependencies
5. Run tests: `bazel test //...`

## Debug

Debug logs are written to `debug_output.log` for troubleshooting.

## License

This project builds upon Claude Code and maintains compatibility with its usage patterns.