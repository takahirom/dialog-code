package main

import (
	"io"
	"os"

	"github.com/takahirom/dialog-code/internal/dialog"
)

func main() {
	// Create real dialog
	d := dialog.NewSimpleOSDialog()

	// Handle permission request hook
	err := handlePermissionRequestHook(os.Stdin, os.Stdout, d)

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
