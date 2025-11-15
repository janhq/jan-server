package main

import (
"fmt"

"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
Use:   "service",
Short: "Service operations",
Long:  `Manage Jan Server services - list, start, stop, logs, and status.`,
}

var serviceListCmd = &cobra.Command{
Use:   "list",
Short: "List all services",
Long:  `List all available Jan Server services and their status.`,
RunE:  runServiceList,
}

var serviceLogsCmd = &cobra.Command{
Use:   "logs [service]",
Short: "Show service logs",
Long:  `Display logs for a specific service.`,
RunE:  runServiceLogs,
Args:  cobra.MinimumNArgs(1),
}

var serviceStatusCmd = &cobra.Command{
Use:   "status [service]",
Short: "Show service status",
Long:  `Display status information for a service.`,
RunE:  runServiceStatus,
}

func init() {
serviceCmd.AddCommand(serviceListCmd)
serviceCmd.AddCommand(serviceLogsCmd)
serviceCmd.AddCommand(serviceStatusCmd)

// logs flags
serviceLogsCmd.Flags().IntP("tail", "n", 100, "Number of lines to show")
serviceLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
}

func runServiceList(cmd *cobra.Command, args []string) error {
fmt.Println("Available services:")
services := []struct {
Name string
Port string
Desc string
}{
{"llm-api", "8080", "LLM API - OpenAI-compatible chat completions"},
{"media-api", "8285", "Media API - File upload and management"},
{"response-api", "8082", "Response API - Multi-step orchestration"},
{"mcp-tools", "8091", "MCP Tools - Model Context Protocol tools"},
}

for _, svc := range services {
fmt.Printf("  %-15s :%s  %s\n", svc.Name, svc.Port, svc.Desc)
}

return nil
}

func runServiceLogs(cmd *cobra.Command, args []string) error {
service := args[0]
tail, _ := cmd.Flags().GetInt("tail")
follow, _ := cmd.Flags().GetBool("follow")

fmt.Printf("Showing logs for %s (tail=%d, follow=%v)\n", service, tail, follow)
fmt.Println("TODO: Integrate with docker compose logs or kubectl logs")

return nil
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
service := ""
if len(args) > 0 {
service = args[0]
}

if service == "" {
fmt.Println("Service status overview:")
fmt.Println("TODO: Check health endpoints for all services")
} else {
fmt.Printf("Status for %s:\n", service)
fmt.Println("TODO: Check health endpoint and show details")
}

return nil
}
