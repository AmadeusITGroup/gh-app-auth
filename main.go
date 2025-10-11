package main

import (
	"fmt"
	"os"

	"github.com/AmadeusITGroup/gh-app-auth/cmd"
	"github.com/AmadeusITGroup/gh-app-auth/pkg/logger"
)

func main() {
	// Initialize diagnostic logging
	logger.Initialize()

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		logger.Close()
		os.Exit(1)
	}

	logger.Close()
}
