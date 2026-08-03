package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/AmadeusITGroup/gh-app-auth/cmd"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/logger"
)

func main() {
	// Initialize diagnostic logging
	logger.Initialize()

	if err := cmd.Execute(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Close()
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		logger.Close()
		os.Exit(1)
	}

	logger.Close()
}
