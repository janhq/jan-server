package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development tools",
	Long:  `Development tools for Jan Server Mono - setup and helpers.`,
}

var devSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup development environment",
	Long:  `Initialize development environment with dependencies and configuration.`,
	RunE:  runDevSetup,
}

func init() {
	devCmd.AddCommand(devSetupCmd)
}

func runDevSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Setting up development environment...")
	fmt.Println()

	// 1. Check Docker (optional - warn if not available)
	dockerAvailable := false
	fmt.Print("Checking Docker... ")
	if err := execCommand("docker", "--version"); err != nil {
		fmt.Println("(not available - Docker features will be limited)")
	} else {
		fmt.Println("OK")
		dockerAvailable = true
	}

	// 2. Check Docker Compose (optional - only if Docker is available)
	if dockerAvailable {
		fmt.Print("Checking Docker Compose... ")
		if err := execCommand("docker", "compose", "version"); err != nil {
			fmt.Println("(not available - some features may be limited)")
		} else {
			fmt.Println("OK")
		}
	}

	// 3. Check for .env file
	fmt.Print("Checking .env file... ")
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Println("not found")
		fmt.Println("Creating .env from template...")

		data, err := os.ReadFile(".env.template")
		if err != nil {
			return fmt.Errorf("failed to read .env.template: %w", err)
		}

		if err := os.WriteFile(".env", data, 0644); err != nil {
			return fmt.Errorf("failed to create .env: %w", err)
		}
		fmt.Println("Created .env file")
	} else {
		fmt.Println("OK")
	}

	// 4. Create necessary directories
	fmt.Print("Creating directories... ")
	dirs := []string{"docker", "backups", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	fmt.Println("OK")

	// 5. Create Docker networks (only if Docker is available)
	if dockerAvailable {
		fmt.Print("Creating Docker networks... ")
		networks := []string{"jan-network"}
		for _, network := range networks {
			// Check if network exists
			checkCmd := execCommandSilent("docker", "network", "inspect", network)
			if checkCmd != nil {
				// Network doesn't exist, create it
				if err := execCommandSilent("docker", "network", "create", network); err != nil {
					fmt.Printf("(skipped %s) ", network)
				}
			}
		}
		fmt.Println("OK")
	}

	fmt.Println()
	fmt.Println("Development environment setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review .env file and update settings")
	fmt.Println("  2. Start services: make docker-up")
	fmt.Println("  3. Check health: make health-check")
	fmt.Println()

	return nil
}
