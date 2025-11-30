package main

import (
	"io"
	"os"

	"github.com/takahirom/dialog-code/internal/dialog"
)

func main() {
	// Parse timeout from command line arguments
	timeout := parseTimeoutFlag(os.Args[1:])

	// Create real dialog with timeout
	d := dialog.NewSimpleOSDialog()
	d.SetTimeout(timeout)

	// Handle permission request hook
	err := handlePermissionRequestHook(os.Stdin, os.Stdout, d, timeout)

	// Handle io.EOF specially - exit 0 with no output
	if err == io.EOF {
		os.Exit(0)
	}

	// Exit with code 1 on error
	if err != nil {
		os.Exit(1)
	}

	// Exit with code 0 on success
	os.Exit(0)
}
