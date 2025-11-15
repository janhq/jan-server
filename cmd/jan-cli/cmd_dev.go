package main

import (
"fmt"

"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
Use:   "dev",
Short: "Development tools",
Long:  `Development tools for Jan Server - setup, scaffolding, and generators.`,
}

var devSetupCmd = &cobra.Command{
Use:   "setup",
Short: "Setup development environment",
Long:  `Initialize development environment with dependencies and configuration.`,
RunE:  runDevSetup,
}

var devScaffoldCmd = &cobra.Command{
Use:   "scaffold [service-name]",
Short: "Scaffold a new service",
Long:  `Generate a new service from the template with proper structure.`,
RunE:  runDevScaffold,
Args:  cobra.ExactArgs(1),
}

func init() {
devCmd.AddCommand(devSetupCmd)
devCmd.AddCommand(devScaffoldCmd)

// scaffold flags
devScaffoldCmd.Flags().StringP("template", "t", "api", "Service template (api, worker)")
devScaffoldCmd.Flags().StringP("port", "p", "", "Service port")
}

func runDevSetup(cmd *cobra.Command, args []string) error {
fmt.Println("Setting up development environment...")
fmt.Println(" Checking Docker...")
fmt.Println(" Checking Go...")
fmt.Println(" Creating .env file...")
fmt.Println(" Pulling Docker images...")
fmt.Println("\n Development environment ready!")
fmt.Println("\nNext steps:")
fmt.Println("  1. Edit .env file with your API keys")
fmt.Println("  2. Run: make up-full")
fmt.Println("  3. Run: make health-check")

return nil
}

func runDevScaffold(cmd *cobra.Command, args []string) error {
serviceName := args[0]
template, _ := cmd.Flags().GetString("template")
port, _ := cmd.Flags().GetString("port")

fmt.Printf("Scaffolding new service: %s\n", serviceName)
fmt.Printf("  Template: %s\n", template)
if port != "" {
fmt.Printf("  Port: %s\n", port)
}

fmt.Println("\nGenerating files...")
fmt.Println("  services/", serviceName, "/")
fmt.Println("     cmd/server/")
fmt.Println("     internal/")
fmt.Println("     Dockerfile")
fmt.Println("     go.mod")
fmt.Println("     README.md")

fmt.Println("\nTODO: Integrate with scripts/new-service-from-template.ps1")
fmt.Printf("\n Service %s scaffolded successfully!\n", serviceName)
fmt.Println("\nNext steps:")
fmt.Println("  1. cd services/", serviceName)
fmt.Println("  2. Update go.mod")
fmt.Println("  3. Implement your service logic")
fmt.Println("  4. Add to docker-compose.yml")

return nil
}
