package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "jan-cli",
	Short: "Jan Server Mono CLI - Command-line tool for Jan Server",
	Long: `jan-cli is the command-line interface for Jan Server Mono.

It provides tools for development, testing, and service operations.

Quick Start:
  jan-cli dev setup              # Setup development environment

Examples:
  # Run API tests
  jan-cli api-test run tests/e2e/automation/collections/auth.postman.json

  # Development tools
  jan-cli dev setup`,
	Version: version,
}

func init() {
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(apiTestCmd)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
