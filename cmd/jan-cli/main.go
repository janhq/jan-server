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
	Short: "Jan Server CLI - Unified command-line tool for Jan Server",
	Long: `jan-cli is the official command-line interface for Jan Server.

It provides tools for configuration management, service operations,
database management, deployment, and development workflows.

Examples:
  # Configuration management
  jan-cli config validate
  jan-cli config export --format env
  
  # Service operations
  jan-cli service list
  jan-cli service logs llm-api
  
  # Development tools
  jan-cli dev setup
  jan-cli dev scaffold my-service`,
	Version: version,
}

func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(devCmd)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config-dir", "config", "Configuration directory")
}
