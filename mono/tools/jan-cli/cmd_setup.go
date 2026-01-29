package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var setupAndRunCmd = &cobra.Command{
	Use:   "setup-and-run",
	Short: "Interactive setup and run Jan Server Mono",
	Long:  `Interactively configure environment variables and start Jan Server Mono with all services.`,
	RunE:  runSetupAndRun,
}

func init() {
	setupAndRunCmd.Flags().Bool("skip-prompts", false, "Skip interactive prompts and use existing .env")
}

func runSetupAndRun(cmd *cobra.Command, args []string) error {
	skipPrompts, _ := cmd.Flags().GetBool("skip-prompts")

	fmt.Println("Jan Server Mono - Setup and Run")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	// Check if .env exists
	envPath := ".env"
	envExists := false
	if _, err := os.Stat(envPath); err == nil {
		envExists = true
	}

	if !skipPrompts {
		// Create or update .env file
		if envExists {
			fmt.Println("Found existing .env file")
			fmt.Print("Do you want to update it? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				if err := promptForEnvVars(envPath); err != nil {
					return fmt.Errorf("failed to update .env: %w", err)
				}
			} else {
				fmt.Println("Using existing .env file...")
			}
		} else {
			fmt.Println("Creating .env file...")
			if err := copyEnvTemplate(envPath); err != nil {
				return fmt.Errorf("failed to copy .env template: %w", err)
			}

			if err := promptForEnvVars(envPath); err != nil {
				return fmt.Errorf("failed to configure .env: %w", err)
			}
		}
	} else if !envExists {
		fmt.Println("Creating .env from template...")
		if err := copyEnvTemplate(envPath); err != nil {
			return fmt.Errorf("failed to copy .env template: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Starting Docker services...")
	fmt.Println("This may take a few minutes on first run...")
	fmt.Println()

	// Start services
	if err := execCommand("docker", "compose", "up", "-d"); err != nil {
		fmt.Println()
		fmt.Println("Note: Some services may already be running")
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Jan Server Mono is starting!")
	fmt.Println()
	fmt.Println("Waiting for services to be ready...")

	// Wait for services to start
	if isWindows() {
		execCommandSilent("powershell", "-Command", "Start-Sleep -Seconds 15")
	} else {
		execCommandSilent("sleep", "15")
	}

	fmt.Println()
	fmt.Println("Access your services:")
	fmt.Println("  Backend API:    http://localhost:8080")
	fmt.Println("  Web UI:         http://localhost:3001")
	fmt.Println("  MinIO Console:  http://localhost:9001 (minioadmin/minioadmin)")
	fmt.Println()
	fmt.Println("Get started:")
	fmt.Println("  1. Register:    curl -X POST http://localhost:8080/v1/auth/local/register \\")
	fmt.Println("                    -H 'Content-Type: application/json' \\")
	fmt.Println("                    -d '{\"email\":\"test@example.com\",\"password\":\"password123\",\"name\":\"Test\"}'")
	fmt.Println()
	fmt.Println("  2. Login:       curl -X POST http://localhost:8080/v1/auth/local/login \\")
	fmt.Println("                    -H 'Content-Type: application/json' \\")
	fmt.Println("                    -d '{\"email\":\"test@example.com\",\"password\":\"password123\"}'")
	fmt.Println()
	fmt.Println("  3. Health:      make health-check")
	fmt.Println("  4. Logs:        make docker-logs")
	fmt.Println("  5. Stop:        make docker-down")
	fmt.Println()

	return nil
}

func copyEnvTemplate(destPath string) error {
	templatePath := ".env.template"

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	return nil
}

func promptForEnvVars(envPath string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("=== Configuration Wizard ===")
	fmt.Println()

	updates := make(map[string]string)

	// 1. JWT Secret
	fmt.Println("Security Configuration")
	fmt.Print("JWT Secret (press Enter for auto-generated): ")
	jwtSecret, _ := reader.ReadString('\n')
	jwtSecret = strings.TrimSpace(jwtSecret)
	if jwtSecret == "" {
		jwtSecret = generateRandomString(32)
		fmt.Println("  Generated JWT secret")
	}
	updates["LOCAL_JWT_SECRET"] = jwtSecret

	// 2. LLM Provider Configuration
	fmt.Println()
	fmt.Println("LLM Provider Setup")
	fmt.Println("Configure a default LLM provider (you can add more via admin API later)")
	fmt.Print("Provider API URL (e.g., https://api.openai.com/v1, press Enter to skip): ")
	providerURL, _ := reader.ReadString('\n')
	providerURL = strings.TrimSpace(providerURL)

	if providerURL != "" {
		fmt.Print("API Key (press Enter if no key required): ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		updates["DEFAULT_PROVIDER_URL"] = providerURL
		if apiKey != "" {
			updates["DEFAULT_PROVIDER_API_KEY"] = apiKey
		}
		fmt.Println("  Provider configured")
	} else {
		fmt.Println("  Skipping provider setup (configure later via admin API)")
	}

	// 3. Memory Service
	fmt.Println()
	fmt.Println("Memory Service (optional)")
	fmt.Println("Enable memory for long-term context and retrieval")
	fmt.Print("Enable memory service? (y/N): ")
	memoryChoice, _ := reader.ReadString('\n')
	memoryChoice = strings.TrimSpace(strings.ToLower(memoryChoice))

	if memoryChoice == "y" || memoryChoice == "yes" {
		updates["MEMORY_ENABLED"] = "true"
		fmt.Print("Memory service URL (default: http://memory-tools:8090): ")
		memoryURL, _ := reader.ReadString('\n')
		memoryURL = strings.TrimSpace(memoryURL)
		if memoryURL == "" {
			memoryURL = "http://memory-tools:8090"
		}
		updates["MEMORY_SERVICE_URL"] = memoryURL
		fmt.Println("  Memory service enabled")
	} else {
		updates["MEMORY_ENABLED"] = "false"
		fmt.Println("  Memory service disabled")
	}

	// Apply all updates
	fmt.Println()
	if len(updates) > 0 {
		if err := applyEnvUpdates(envPath, updates); err != nil {
			return err
		}
		fmt.Println("Configuration saved to .env")
	}

	return nil
}

func applyEnvUpdates(envPath string, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	pending := make(map[string]string, len(updates))
	for key, value := range updates {
		pending[key] = value
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		for key, value := range pending {
			if strings.HasPrefix(trimmed, key+"=") {
				lines[i] = fmt.Sprintf("%s=%s", key, value)
				delete(pending, key)
			}
		}
	}

	for key, value := range pending {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	return nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
