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
	Short: "Interactive setup and run Jan Server",
	Long:  `Interactively configure environment variables and start Jan Server with all services.`,
	RunE:  runSetupAndRun,
}

func init() {
	setupAndRunCmd.Flags().Bool("skip-prompts", false, "Skip interactive prompts and use existing .env")
	setupAndRunCmd.Flags().Bool("with-memory-tools", false, "Enable memory tools profile and defaults during setup")
	setupAndRunCmd.Flags().Bool("with-realtime-api", false, "Enable realtime API profile during setup")
	setupAndRunCmd.Flags().Bool("with-aio", false, "Enable AIO Sandbox profile during setup")
	setupAndRunCmd.Flags().Bool("skip-realtime", false, "Skip realtime API setup (disable realtime profile)")
	setupAndRunCmd.Flags().Bool("skip-memory", false, "Skip memory tools setup (disable memory profile)")
	setupAndRunCmd.Flags().Bool("skip-aio", false, "Skip AIO Sandbox setup (disable aio profile)")
}

func runSetupAndRun(cmd *cobra.Command, args []string) error {
	skipPrompts, _ := cmd.Flags().GetBool("skip-prompts")
	enableMemory, _ := cmd.Flags().GetBool("with-memory-tools")
	enableRealtime, _ := cmd.Flags().GetBool("with-realtime-api")
	enableAIO, _ := cmd.Flags().GetBool("with-aio")
	skipRealtime, _ := cmd.Flags().GetBool("skip-realtime")
	skipMemory, _ := cmd.Flags().GetBool("skip-memory")
	skipAIO, _ := cmd.Flags().GetBool("skip-aio")

	fmt.Println("🚀 Jan Server Setup and Run")
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
			fmt.Println("✓ Found existing .env file")
			fmt.Print("Do you want to update it? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			// Default is No for updating existing config
			if response != "y" && response != "yes" {
				fmt.Println("Using existing .env file...")
			} else {
				if err := promptForEnvVars(envPath, enableMemory, enableAIO, skipRealtime, skipMemory, skipAIO); err != nil {
					return fmt.Errorf("failed to update .env: %w", err)
				}
			}
		} else {
			fmt.Println("📝 Creating .env file...")
			// Copy template
			if err := copyEnvTemplate(envPath); err != nil {
				return fmt.Errorf("failed to copy .env template: %w", err)
			}

			if err := promptForEnvVars(envPath, enableMemory, enableAIO, skipRealtime, skipMemory, skipAIO); err != nil {
				return fmt.Errorf("failed to configure .env: %w", err)
			}
		}
	} else if !envExists {
		// Skip prompts but no .env exists - copy template
		fmt.Println("📝 Creating .env from template...")
		if err := copyEnvTemplate(envPath); err != nil {
			return fmt.Errorf("failed to copy .env template: %w", err)
		}
	}

	if skipPrompts && enableMemory {
		if err := applyMemoryDefaults(envPath); err != nil {
			return fmt.Errorf("failed to enable memory tools defaults: %w", err)
		}
	}

	if skipPrompts && enableRealtime {
		if err := applyRealtimeDefaults(envPath); err != nil {
			return fmt.Errorf("failed to enable realtime API defaults: %w", err)
		}
	}

	if skipPrompts && enableAIO {
		if err := applyAIODefaults(envPath); err != nil {
			return fmt.Errorf("failed to enable AIO Sandbox defaults: %w", err)
		}
	}

	if skipRealtime {
		if err := disableRealtime(envPath); err != nil {
			return fmt.Errorf("failed to disable realtime API: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("⚙️  Running setup...")
	fmt.Println()

	// Run dev setup
	if err := execCommand("make", "setup"); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// Ask about monitoring setup
	if !skipPrompts {
		fmt.Println()
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("📊 Monitoring Stack Setup (Optional)")
		fmt.Println()
		fmt.Println("Would you like to set up the monitoring stack?")
		fmt.Println("This includes:")
		fmt.Println("  • Prometheus (metrics)")
		fmt.Println("  • Grafana (dashboards)")
		fmt.Println("  • Jaeger (distributed tracing)")
		fmt.Println("  • OpenTelemetry Collector")
		fmt.Println()
		fmt.Println()
		fmt.Print("Set up monitoring? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		monitorResponse, _ := reader.ReadString('\n')
		monitorResponse = strings.TrimSpace(strings.ToLower(monitorResponse))

		// Default is No for monitoring (optional feature)
		if monitorResponse == "y" || monitorResponse == "yes" {
			fmt.Println()
			fmt.Println("🔧 Installing monitoring dependencies...")

			// Enable tracing in .env
			if err := updateEnvVariable(envPath, "OTEL_ENABLED", "true"); err != nil {
				fmt.Println("⚠️  Warning: Failed to enable OTEL_ENABLED in .env")
			} else {
				fmt.Println("✓ Enabled telemetry collection (OTEL_ENABLED=true)")
			}

			if err := execCommand("make", "monitor-up"); err != nil {
				fmt.Println("⚠️  Warning: Failed to start monitoring stack")
				fmt.Println("You can set it up later with: jan-cli monitor setup")
			} else {
				fmt.Println("✓ Monitoring stack started successfully!")
				fmt.Println()
				fmt.Println("Access monitoring dashboards:")
				fmt.Println("  • Grafana:    http://localhost:3331 (admin/admin)")
				fmt.Println("  • Prometheus: http://localhost:9090")
				fmt.Println("  • Jaeger:     http://localhost:16686")
			}
		} else {
			fmt.Println("⏭️  Skipping monitoring setup")
			fmt.Println("You can set it up later with: jan-cli monitor setup")
		}
	}

	// Ask about starting platform app
	startPlatform := false
	startWeb := true
	if !skipPrompts {
		fmt.Println()
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("🌐 Platform Web App Setup (Optional)")
		fmt.Println()
		fmt.Println("Would you like to start the Platform web app?")
		fmt.Println("This provides:")
		fmt.Println("  • Interactive API documentation")
		fmt.Println("  • Admin dashboard")
		fmt.Println("  • Web UI for Jan Server")
		fmt.Println()
		fmt.Print("Start Platform app? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		platformResponse, _ := reader.ReadString('\n')
		platformResponse = strings.TrimSpace(strings.ToLower(platformResponse))

		if platformResponse == "y" || platformResponse == "yes" {
			startPlatform = true
			fmt.Println("✓ Platform app will be started")

			// Update COMPOSE_PROFILES to include platform
			if err := addPlatformProfile(envPath); err != nil {
				fmt.Println("⚠️  Warning: Failed to add platform profile to .env")
			}
		} else {
			fmt.Println("⏭️  Skipping platform app")
			fmt.Println("You can start it later by adding 'platform' to COMPOSE_PROFILES in .env")
		}
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("🐳 Starting Docker services...")
	fmt.Println("This may take 1-2 minutes on first run...")
	fmt.Println()

	// Start services
	if err := execCommand("make", "up-full"); err != nil {
		// Docker compose up -d returns non-zero if services are already running
		// Check if it's actually an error or just a warning
		fmt.Println()
		fmt.Println("Note: Some services may already be running")
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("✅ Jan Server is starting!")
	fmt.Println()
	fmt.Println("Waiting for services to be ready (30 seconds)...")

	// Wait for services to start - cross-platform
	if isWindows() {
		execCommandSilent("powershell", "-Command", "Start-Sleep -Seconds 30")
	} else {
		execCommandSilent("sleep", "30")
	}

	fmt.Println()
	fmt.Println("Access your services:")
	fmt.Println("  • API Gateway:      http://localhost:8000")
	fmt.Println("  • API Docs:         http://localhost:8000/api/swagger/index.html")
	fmt.Println("  • LLM API:          http://localhost:8080")
	fmt.Println("  • Keycloak:         http://localhost:8085 (admin/admin)")

	// Only show vLLM if using local provider
	if os.Getenv("_USING_LOCAL_VLLM") == "true" {
		fmt.Println("  • vLLM (Local):     http://localhost:8101")
	}

	// Show Realtime API if enabled
	if os.Getenv("REALTIME_API_ENABLED") == "true" {
		fmt.Println("  • Realtime API:     http://localhost:8186")
	}

	// Show AIO Sandbox if enabled
	if os.Getenv("AIO_ENABLED") == "true" {
		fmt.Println("  • AIO Sandbox:      http://localhost:8180")
		fmt.Println("  • AIO Docs:         http://localhost:8180/v1/docs")
	}

	// Show Platform if it was started
	if startPlatform {
		fmt.Println("  • Platform Web App: http://localhost:3000")
	}
	if startWeb {
		fmt.Println("  - Web App:          http://localhost:3001")
	}

	fmt.Println()
	fmt.Println("Get started:")
	fmt.Println("  1. Get a token:     curl -X POST http://localhost:8000/llm/auth/guest-login")
	fmt.Println("  2. Health check:    make health-check")
	fmt.Println("  3. View logs:       make logs-llm-api")
	fmt.Println("  4. Stop services:   make down")
	fmt.Println()

	return nil
}

func copyEnvTemplate(destPath string) error {
	templatePath := ".env.template"

	// Read template
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	return nil
}

func promptForEnvVars(envPath string, defaultEnableMemory bool, enableAIO bool, skipRealtime bool, skipMemory bool, skipAIO bool) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("=== Configuration Wizard ===")
	fmt.Println()

	// Read current .env
	if _, err := os.ReadFile(envPath); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	updates := make(map[string]string)

	// 1. LLM Provider Configuration
	fmt.Println("📦 LLM Provider Setup")
	fmt.Println("Choose your LLM provider:")
	fmt.Println("  1. Local vLLM (requires GPU, uses HuggingFace token)")
	fmt.Println("  2. Remote API endpoint (OpenAI-compatible)")
	fmt.Print("Enter choice [1/2] (default: 1): ")

	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)
	if providerChoice == "" {
		providerChoice = "1"
	}

	// Track which services to enable
	useLocalVLLM := false
	profiles := []string{"infra", "api", "mcp", "web"} // Always include core services + web app
	updates["JAN_PROVIDER_CONFIGS"] = "true"

	if providerChoice == "1" {
		// Local vLLM setup
		fmt.Println()
		fmt.Print("HF_TOKEN (get from https://huggingface.co/settings/tokens): ")
		hfToken, _ := reader.ReadString('\n')
		hfToken = strings.TrimSpace(hfToken)
		if hfToken != "" {
			updates["HF_TOKEN"] = hfToken
		}

		fmt.Print("VLLM_INTERNAL_KEY (press Enter to keep current or default): ")
		internalKey, _ := reader.ReadString('\n')
		internalKey = strings.TrimSpace(internalKey)
		if internalKey != "" {
			updates["VLLM_INTERNAL_KEY"] = internalKey
		}

		updates["VLLM_ENABLED"] = "true"
		updates["REMOTE_LLM_ENABLED"] = "false"
		updates["VLLM_PROVIDER_URL"] = "http://vllm-jan-gpu:8001/v1"
		updates["VLLM_TOOL_SUPPORT"] = "true"
		profiles = append(profiles, "full") // Add vLLM
		useLocalVLLM = true

		fmt.Println("✓ Will use local vLLM with model: Qwen/Qwen2.5-0.5B-Instruct")
	} else {
		// Remote provider setup
		fmt.Println()
		fmt.Print("Remote API URL (e.g., https://api.openai.com/v1): ")
		remoteURL, _ := reader.ReadString('\n')
		remoteURL = strings.TrimSpace(remoteURL)

		fmt.Print("API Key (press Enter if no key required): ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		updates["VLLM_ENABLED"] = "false"
		updates["REMOTE_LLM_ENABLED"] = "true"
		updates["REMOTE_LLM_PROVIDER_URL"] = remoteURL
		updates["REMOTE_API_KEY"] = apiKey

		if remoteURL != "" {
			updates["HF_TOKEN"] = "not_required_for_remote_provider"
			// Note: infra, api, mcp already in profiles
			fmt.Println("✓ Will use remote provider:", remoteURL)
		}
	}

	// 2. MCP Search Tool Configuration
	fmt.Println()
	fmt.Println("🔍 MCP Search Tool Setup")
	fmt.Println("Choose search provider for MCP tools:")
	fmt.Println("  1. Serper (requires API key, Google search)")
	fmt.Println("  2. SearXNG (local, no API key needed)")
	fmt.Println("  3. None (disable MCP search, but Vector Store still available)")
	fmt.Print("Enter choice [1/2/3] (default: 1): ")

	searchChoice, _ := reader.ReadString('\n')
	searchChoice = strings.TrimSpace(searchChoice)
	if searchChoice == "" {
		searchChoice = "1"
	}

	// MCP profile is already included in the profiles list

	switch searchChoice {
	case "1":
		fmt.Println()
		fmt.Print("SERPER_API_KEY (get from https://serper.dev): ")
		serperKey, _ := reader.ReadString('\n')
		serperKey = strings.TrimSpace(serperKey)
		if serperKey != "" {
			updates["SERPER_API_KEY"] = serperKey
			updates["SEARCH_ENGINE"] = "serper"
			fmt.Println("✓ Will use Serper for search")
		} else {
			fmt.Println("⚠️  No API key provided, falling back to SearXNG")
			updates["SEARCH_ENGINE"] = "searxng"
		}
	case "2":
		updates["SEARCH_ENGINE"] = "searxng"
		updates["SERPER_API_KEY"] = "not_required_for_searxng"
		fmt.Println("✓ Will use SearXNG (local) for search")
	case "3":
		updates["SEARCH_ENGINE"] = "none"
		updates["SERPER_API_KEY"] = "mcp_search_disabled"
		fmt.Println("✓ MCP search disabled (Vector Store still available)")
	}

	// 3. Memory Tools Configuration
	var enableMemory bool
	var externalEmbedding bool
	var useRedis bool

	if skipMemory {
		// Skip memory prompt - disable memory tools
		enableMemory = false
		fmt.Println()
		fmt.Println("⏭️  Skipping memory tools setup (disabled by default)")
	} else {
		fmt.Println()
		fmt.Println("🧠 Memory Tools Setup")
		fmt.Println("Enable memory tools for long-term context and retrieval.")
		memoryPromptDefault := "Y/n"
		if !defaultEnableMemory {
			memoryPromptDefault = "y/N"
		}
		fmt.Printf("Enable memory tools? (%s): ", memoryPromptDefault)

		memoryChoice, _ := reader.ReadString('\n')
		memoryChoice = strings.TrimSpace(strings.ToLower(memoryChoice))

		// Default based on defaultEnableMemory flag (Y/n or y/N)
		enableMemory = defaultEnableMemory
		if memoryChoice != "" {
			enableMemory = memoryChoice != "n" && memoryChoice != "no"
		}

		if enableMemory {
			externalEmbedding, useRedis = configureMemoryOptions(reader, updates)
		}
	}
	applyMemorySettings(updates, &profiles, enableMemory, externalEmbedding, useRedis)

	// 4. Media API Configuration
	fmt.Println()
	fmt.Println("🖼️  Media API Setup")

	// Media API with local storage only works when using local vLLM
	if !useLocalVLLM {
		fmt.Println("Note: Media API with local storage requires local vLLM deployment")
		fmt.Println("      Only S3 storage is available with remote API providers")
		fmt.Print("Enable Media API with S3 storage? (y/N): ")

		mediaChoice, _ := reader.ReadString('\n')
		mediaChoice = strings.TrimSpace(strings.ToLower(mediaChoice))

		// Default is No for S3 with remote provider (requires credentials)
		if mediaChoice == "y" || mediaChoice == "yes" {
			updates["MEDIA_API_ENABLED"] = "true"
			updates["MEDIA_STORAGE_BACKEND"] = "s3"

			fmt.Println()
			fmt.Println("S3-compatible storage configuration:")
			fmt.Println("(Press Enter to use default Menlo AI settings)")

			fmt.Print("S3 Endpoint URL (default: https://s3.menlo.ai): ")
			s3Endpoint, _ := reader.ReadString('\n')
			s3Endpoint = strings.TrimSpace(s3Endpoint)
			if s3Endpoint == "" {
				s3Endpoint = "https://s3.menlo.ai"
			}
			updates["MEDIA_S3_ENDPOINT"] = s3Endpoint

			fmt.Print("S3 Bucket name (default: platform-dev): ")
			s3Bucket, _ := reader.ReadString('\n')
			s3Bucket = strings.TrimSpace(s3Bucket)
			if s3Bucket == "" {
				s3Bucket = "platform-dev"
			}
			updates["MEDIA_S3_BUCKET"] = s3Bucket

			fmt.Print("S3 Access Key ID (default: 7N33WPTUI1KN99MFILQS): ")
			s3AccessKey, _ := reader.ReadString('\n')
			s3AccessKey = strings.TrimSpace(s3AccessKey)
			if s3AccessKey == "" {
				s3AccessKey = "7N33WPTUI1KN99MFILQS"
			}
			updates["MEDIA_S3_ACCESS_KEY_ID"] = s3AccessKey

			fmt.Print("S3 Secret Access Key (default: ppxQsHpnfDSewYZD065aGjQeEQ0nTFA7c2aHNPz5): ")
			s3SecretKey, _ := reader.ReadString('\n')
			s3SecretKey = strings.TrimSpace(s3SecretKey)
			if s3SecretKey == "" {
				s3SecretKey = "ppxQsHpnfDSewYZD065aGjQeEQ0nTFA7c2aHNPz5"
			}
			updates["MEDIA_S3_SECRET_ACCESS_KEY"] = s3SecretKey

			fmt.Print("S3 Region (default: us-west-2): ")
			s3Region, _ := reader.ReadString('\n')
			s3Region = strings.TrimSpace(s3Region)
			if s3Region == "" {
				s3Region = "us-west-2"
			}
			updates["MEDIA_S3_REGION"] = s3Region
			promptForPublicS3URL(reader, updates, s3Endpoint, s3Bucket)

			// Set media API URLs
			updates["MEDIA_API_URL"] = "http://media-api:8285"
			updates["MEDIA_RESOLVE_URL"] = "http://media-api:8285/v1/media/resolve"

			fmt.Println("✓ Media API enabled with S3 storage")
		} else {
			updates["MEDIA_API_ENABLED"] = "false"
			fmt.Println("✓ Media API disabled")
		}
	} else {
		// Local vLLM - offer both storage options
		fmt.Print("Enable Media API? (Y/n): ")

		mediaChoice, _ := reader.ReadString('\n')
		mediaChoice = strings.TrimSpace(strings.ToLower(mediaChoice))

		// Default is Yes for Media API with local vLLM
		if mediaChoice == "n" || mediaChoice == "no" {
			updates["MEDIA_API_ENABLED"] = "false"
			fmt.Println("✓ Media API disabled")
		} else {
			updates["MEDIA_API_ENABLED"] = "true"

			// Ask for storage backend
			fmt.Println()
			fmt.Println("Choose Media storage backend:")
			fmt.Println("  1. Local file system (default, stores files locally)")
			fmt.Println("  2. S3-compatible storage (requires credentials)")
			fmt.Print("Enter choice [1/2] (default: 1): ")

			storageChoice, _ := reader.ReadString('\n')
			storageChoice = strings.TrimSpace(storageChoice)
			if storageChoice == "" {
				storageChoice = "1"
			}

			if storageChoice == "2" {
				// S3 Configuration
				updates["MEDIA_STORAGE_BACKEND"] = "s3"

				fmt.Println()
				fmt.Println("S3-compatible storage configuration:")
				fmt.Println("(Press Enter to use default Menlo AI settings)")

				fmt.Print("S3 Endpoint URL (default: https://s3.menlo.ai): ")
				s3Endpoint, _ := reader.ReadString('\n')
				s3Endpoint = strings.TrimSpace(s3Endpoint)
				if s3Endpoint == "" {
					s3Endpoint = "https://s3.menlo.ai"
				}
				updates["MEDIA_S3_ENDPOINT"] = s3Endpoint

				fmt.Print("S3 Bucket name (default: platform-dev): ")
				s3Bucket, _ := reader.ReadString('\n')
				s3Bucket = strings.TrimSpace(s3Bucket)
				if s3Bucket == "" {
					s3Bucket = "platform-dev"
				}
				updates["MEDIA_S3_BUCKET"] = s3Bucket

				fmt.Print("S3 Access Key ID (default: 7N33WPTUI1KN99MFILQS): ")
				s3AccessKey, _ := reader.ReadString('\n')
				s3AccessKey = strings.TrimSpace(s3AccessKey)
				if s3AccessKey == "" {
					s3AccessKey = "7N33WPTUI1KN99MFILQS"
				}
				updates["MEDIA_S3_ACCESS_KEY_ID"] = s3AccessKey

				fmt.Print("S3 Secret Access Key (default: ppxQsHpnfDSewYZD065aGjQeEQ0nTFA7c2aHNPz5): ")
				s3SecretKey, _ := reader.ReadString('\n')
				s3SecretKey = strings.TrimSpace(s3SecretKey)
				if s3SecretKey == "" {
					s3SecretKey = "ppxQsHpnfDSewYZD065aGjQeEQ0nTFA7c2aHNPz5"
				}
				updates["MEDIA_S3_SECRET_ACCESS_KEY"] = s3SecretKey

				fmt.Print("S3 Region (default: us-west-2): ")
				s3Region, _ := reader.ReadString('\n')
				s3Region = strings.TrimSpace(s3Region)
				if s3Region == "" {
					s3Region = "us-west-2"
				}
				updates["MEDIA_S3_REGION"] = s3Region
				promptForPublicS3URL(reader, updates, s3Endpoint, s3Bucket)

				fmt.Println("✓ Media API enabled with S3 storage")
			} else {
				// Local file system storage (default)
				updates["MEDIA_STORAGE_BACKEND"] = "local"
				updates["MEDIA_LOCAL_STORAGE_PATH"] = "./media-data"
				updates["MEDIA_LOCAL_STORAGE_BASE_URL"] = "http://localhost:8285/v1/files"
				fmt.Println("✓ Media API enabled with local file system storage")
			}

			// Set media API URLs (common for both backends)
			updates["MEDIA_API_URL"] = "http://media-api:8285"
			updates["MEDIA_RESOLVE_URL"] = "http://media-api:8285/v1/media/resolve"
		}
	}

	// 5. Realtime API Configuration
	if !skipRealtime {
		fmt.Println()
		fmt.Println("🎙️  Realtime API Setup")
		fmt.Println("Enable realtime API for real-time audio/video communication via LiveKit.")
		fmt.Print("Enable Realtime API? (y/N): ")

		realtimeChoice, _ := reader.ReadString('\n')
		realtimeChoice = strings.TrimSpace(strings.ToLower(realtimeChoice))

		// Default is No for Realtime API (requires LiveKit credentials)
		if realtimeChoice == "y" || realtimeChoice == "yes" {
			updates["REALTIME_API_ENABLED"] = "true"
			profiles = append(profiles, "realtime")

			fmt.Println()
			fmt.Println("LiveKit Configuration (required for Realtime API):")
			fmt.Println("Get credentials from https://cloud.livekit.io or your self-hosted LiveKit server")
			fmt.Println()

			fmt.Print("LIVEKIT_WS_URL (e.g., wss://your-app.livekit.cloud): ")
			livekitURL, _ := reader.ReadString('\n')
			livekitURL = strings.TrimSpace(livekitURL)
			if livekitURL != "" {
				updates["LIVEKIT_WS_URL"] = livekitURL
			}

			fmt.Print("LIVEKIT_API_KEY: ")
			livekitKey, _ := reader.ReadString('\n')
			livekitKey = strings.TrimSpace(livekitKey)
			if livekitKey != "" {
				updates["LIVEKIT_API_KEY"] = livekitKey
			}

			fmt.Print("LIVEKIT_API_SECRET: ")
			livekitSecret, _ := reader.ReadString('\n')
			livekitSecret = strings.TrimSpace(livekitSecret)
			if livekitSecret != "" {
				updates["LIVEKIT_API_SECRET"] = livekitSecret
			}

			fmt.Println("✓ Realtime API enabled (profile: realtime)")
		} else {
			updates["REALTIME_API_ENABLED"] = "false"
			fmt.Println("✓ Realtime API disabled (enable later by editing .env)")
		}
	} else {
		updates["REALTIME_API_ENABLED"] = "false"
		fmt.Println("✓ Realtime API disabled (skipped via --skip-realtime flag)")
	}

	// 6. AIO Sandbox Configuration
	if skipAIO {
		updates["AIO_ENABLED"] = "false"
		fmt.Println()
		fmt.Println("⏭️  Skipping AIO Sandbox setup (disabled via --skip-aio flag)")
	} else if enableAIO {
		// Enable AIO via flag without prompting
		updates["AIO_ENABLED"] = "true"
		updates["AIO_URL"] = "http://aio-sandbox:8080"
		updates["AIO_TIMEOUT"] = "120s"
		profiles = append(profiles, "aio")
		fmt.Println()
		fmt.Println("✓ AIO Sandbox enabled (via --with-aio flag)")
	} else {
		fmt.Println()
		fmt.Println("🤖 AIO Sandbox Setup")
		fmt.Println("AIO Sandbox provides browser automation, shell/terminal, file operations, and code execution.")
		fmt.Println("Note: AIO requires ~4GB RAM and takes ~60s to start.")
		fmt.Print("Enable AIO Sandbox? (y/N): ")

		aioChoice, _ := reader.ReadString('\n')
		aioChoice = strings.TrimSpace(strings.ToLower(aioChoice))

		// Default is No for AIO (resource-intensive)
		if aioChoice == "y" || aioChoice == "yes" {
			updates["AIO_ENABLED"] = "true"
			updates["AIO_URL"] = "http://aio-sandbox:8080"
			updates["AIO_TIMEOUT"] = "120s"
			profiles = append(profiles, "aio")
			fmt.Println("✓ AIO Sandbox enabled (profile: aio)")
			fmt.Println("  Tools: aio_shell_exec, aio_file_read, aio_file_write, aio_file_list,")
			fmt.Println("         aio_browser_info, aio_code_execute, aio_markitdown_convert")
		} else {
			updates["AIO_ENABLED"] = "false"
			fmt.Println("✓ AIO Sandbox disabled (enable later with: make up-aio)")
		}
	}

	// Apply all updates
	fmt.Println()

	// Ensure Keycloak URLs are properly set for browser access
	if _, exists := updates["KEYCLOAK_PUBLIC_URL"]; !exists {
		updates["KEYCLOAK_PUBLIC_URL"] = "http://localhost:8085"
	}
	if _, exists := updates["KEYCLOAK_ADMIN_URL"]; !exists {
		updates["KEYCLOAK_ADMIN_URL"] = "http://localhost:8085"
	}
	if _, exists := updates["KEYCLOAK_BASE_URL"]; !exists {
		updates["KEYCLOAK_BASE_URL"] = "http://keycloak:8085"
	}
	if _, exists := updates["ISSUER"]; !exists {
		updates["ISSUER"] = "http://localhost:8085/realms/jan"
	}

	// Set COMPOSE_PROFILES based on enabled services
	if len(profiles) > 0 {
		updates["COMPOSE_PROFILES"] = strings.Join(profiles, ",")
	}

	// Store provider choice for return value (used later for conditional output)
	if useLocalVLLM {
		updates["_USING_LOCAL_VLLM"] = "true"
	}

	if len(updates) > 0 {
		if err := applyEnvUpdates(envPath, updates); err != nil {
			return err
		}

		fmt.Println("✓ Configuration saved to .env")
	} else {
		fmt.Println("✓ No changes made")
	}

	// Check if using local vLLM (look in updates or re-read from env)
	data, _ := os.ReadFile(envPath)
	if strings.Contains(string(data), "COMPOSE_PROFILES=full") {
		os.Setenv("_USING_LOCAL_VLLM", "true")
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

func applyMemoryDefaults(envPath string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	profiles := parseProfiles(strings.Split(string(data), "\n"))
	updates := make(map[string]string)
	setMemoryDefaults(updates, &profiles, false, false)
	updates["MCP_ENABLE_MEMORY_RETRIEVE"] = "true"
	if len(profiles) > 0 {
		updates["COMPOSE_PROFILES"] = strings.Join(profiles, ",")
	}

	return applyEnvUpdates(envPath, updates)
}

func applyRealtimeDefaults(envPath string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	profiles := parseProfiles(strings.Split(string(data), "\n"))
	updates := make(map[string]string)

	// Add realtime profile if not present
	hasRealtime := false
	for _, profile := range profiles {
		if profile == "realtime" {
			hasRealtime = true
			break
		}
	}
	if !hasRealtime {
		profiles = append(profiles, "realtime")
	}

	updates["REALTIME_API_ENABLED"] = "true"
	if len(profiles) > 0 {
		updates["COMPOSE_PROFILES"] = strings.Join(profiles, ",")
	}

	return applyEnvUpdates(envPath, updates)
}

func disableRealtime(envPath string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	profiles := parseProfiles(strings.Split(string(data), "\n"))
	updates := make(map[string]string)

	// Remove realtime profile if present
	newProfiles := []string{}
	for _, profile := range profiles {
		if profile != "realtime" {
			newProfiles = append(newProfiles, profile)
		}
	}

	updates["REALTIME_API_ENABLED"] = "false"
	if len(newProfiles) > 0 {
		updates["COMPOSE_PROFILES"] = strings.Join(newProfiles, ",")
	}

	fmt.Println("✓ Realtime API disabled (profile removed)")
	return applyEnvUpdates(envPath, updates)
}

func applyAIODefaults(envPath string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	profiles := parseProfiles(strings.Split(string(data), "\n"))
	updates := make(map[string]string)

	// Add aio profile if not present
	hasAIO := false
	for _, profile := range profiles {
		if profile == "aio" {
			hasAIO = true
			break
		}
	}
	if !hasAIO {
		profiles = append(profiles, "aio")
	}

	updates["AIO_ENABLED"] = "true"
	updates["AIO_URL"] = "http://aio-sandbox:8080"
	updates["AIO_TIMEOUT"] = "120s"
	if len(profiles) > 0 {
		updates["COMPOSE_PROFILES"] = strings.Join(profiles, ",")
	}

	fmt.Println("✓ AIO Sandbox enabled (profile: aio)")
	return applyEnvUpdates(envPath, updates)
}

func applyMemorySettings(updates map[string]string, profiles *[]string, enable bool, externalEmbedding bool, useRedis bool) {
	if enable {
		setMemoryDefaults(updates, profiles, externalEmbedding, useRedis)
		updates["MCP_ENABLE_MEMORY_RETRIEVE"] = "true"
		fmt.Println("Memory tools enabled (profile: memory)")
	} else {
		updates["MEMORY_TOOLS_ENABLED"] = "false"
		updates["MCP_ENABLE_MEMORY_RETRIEVE"] = "false"
		updates["PROMPT_ORCHESTRATION_MEMORY"] = "false"
		fmt.Println("Memory tools disabled (enable later by editing .env)")
	}
}

func setMemoryDefaults(updates map[string]string, profiles *[]string, externalEmbedding bool, useRedis bool) {
	if profiles != nil {
		hasMemory := false
		hasMock := false
		hasRedis := false
		for _, profile := range *profiles {
			if profile == "memory" {
				hasMemory = true
			}
			if profile == "memory-mock" {
				hasMock = true
			}
			if profile == "memory-redis" {
				hasRedis = true
			}
		}
		if !hasMemory {
			*profiles = append(*profiles, "memory")
		}
		if !externalEmbedding && !hasMock {
			*profiles = append(*profiles, "memory-mock")
		}
		if useRedis && !hasRedis {
			*profiles = append(*profiles, "memory-redis")
		}
	}

	if _, exists := updates["MEMORY_TOOLS_PORT"]; !exists {
		updates["MEMORY_TOOLS_PORT"] = "8090"
	}

	if !externalEmbedding && updates["EMBEDDING_SERVICE_URL"] == "" {
		updates["EMBEDDING_SERVICE_URL"] = "http://bge-m3:8091"
	}

	updates["MEMORY_TOOLS_ENABLED"] = "true"
	updates["EMBEDDING_CACHE_TYPE"] = "memory"
	updates["PROMPT_ORCHESTRATION_MEMORY"] = "true"
}

func configureMemoryOptions(reader *bufio.Reader, updates map[string]string) (bool, bool) {
	fmt.Println()
	fmt.Println("Memory Embedding Service")
	fmt.Println("Use the built-in BGE-M3 mock (default) or point to your own embedding endpoint.")
	fmt.Print("Custom embedding service URL (leave blank for http://bge-m3:8091): ")
	customURL, _ := reader.ReadString('\n')
	customURL = strings.TrimSpace(customURL)
	external := false
	if customURL != "" {
		updates["EMBEDDING_SERVICE_URL"] = customURL
		external = true
	} else if _, exists := updates["EMBEDDING_SERVICE_URL"]; !exists {
		updates["EMBEDDING_SERVICE_URL"] = "http://bge-m3:8091"
	}

	fmt.Println()
	fmt.Println("Embedding Cache")
	fmt.Println("Choose Redis for shared cache or in-memory for simplicity.")
	fmt.Print("Use Redis cache? (y/N): ")
	cacheChoice, _ := reader.ReadString('\n')
	cacheChoice = strings.TrimSpace(strings.ToLower(cacheChoice))
	// Default is No for Redis (in-memory is simpler for getting started)
	useRedis := false
	if cacheChoice == "y" || cacheChoice == "yes" {
		updates["EMBEDDING_CACHE_TYPE"] = "redis"
		fmt.Print("Redis URL (default: redis://redis-memory:6379/3): ")
		redisURL, _ := reader.ReadString('\n')
		redisURL = strings.TrimSpace(redisURL)
		if redisURL == "" {
			redisURL = "redis://redis-memory:6379/3"
		}
		updates["EMBEDDING_CACHE_REDIS_URL"] = redisURL
		useRedis = true
	} else {
		updates["EMBEDDING_CACHE_TYPE"] = "memory"
	}

	return external, useRedis
}

func promptForPublicS3URL(reader *bufio.Reader, updates map[string]string, s3Endpoint string, s3Bucket string) {
	fmt.Println()
	fmt.Println("Public S3 URLs (optional)")
	fmt.Println("Remote LLMs cannot read local media URLs.")
	fmt.Print("Return public S3 URLs in upload responses? (y/N): ")

	publicChoice, _ := reader.ReadString('\n')
	publicChoice = strings.TrimSpace(strings.ToLower(publicChoice))

	if publicChoice == "y" || publicChoice == "yes" {
		updates["MEDIA_S3_URL_ENABLED"] = "true"

		base := strings.TrimSuffix(strings.TrimSpace(s3Endpoint), "/")
		defaultPublic := base
		if s3Bucket != "" {
			defaultPublic = fmt.Sprintf("%s/%s", base, strings.TrimPrefix(s3Bucket, "/"))
		}

		fmt.Printf("Public S3 base URL (default: %s): ", defaultPublic)
		publicEndpoint, _ := reader.ReadString('\n')
		publicEndpoint = strings.TrimSpace(publicEndpoint)
		if publicEndpoint == "" {
			publicEndpoint = defaultPublic
		}
		updates["MEDIA_S3_PUBLIC_ENDPOINT"] = publicEndpoint
		fmt.Println("✓ Public S3 URLs enabled for media uploads")
	} else {
		updates["MEDIA_S3_URL_ENABLED"] = "false"
	}
}

func updateEnvVariable(envPath, key, value string) error {
	// Read current .env
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	found := false

	// Update existing line or add new one
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	// If not found, append
	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	return nil
}

func parseProfiles(lines []string) []string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "COMPOSE_PROFILES=") {
			value := strings.TrimPrefix(trimmed, "COMPOSE_PROFILES=")
			if value != "" {
				return strings.Split(value, ",")
			}
		}
	}
	return []string{"infra", "api", "mcp", "web"}
}

func addPlatformProfile(envPath string) error {
	// Read current .env
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	// Find and update COMPOSE_PROFILES line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "COMPOSE_PROFILES=") {
			value := strings.TrimPrefix(trimmed, "COMPOSE_PROFILES=")
			profiles := strings.Split(value, ",")

			// Check if platform is already in the list
			hasPlatform := false
			for _, p := range profiles {
				if strings.TrimSpace(p) == "platform" {
					hasPlatform = true
					break
				}
			}

			// Add platform if not present
			if !hasPlatform {
				profiles = append(profiles, "platform")
				lines[i] = "COMPOSE_PROFILES=" + strings.Join(profiles, ",")
			}
			break
		}
	}

	// Write back
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	return nil
}
