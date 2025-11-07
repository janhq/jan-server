# MCP Provider Integration

The `mcp-tools` service now supports bridging to external MCP (Model Context Protocol) servers, allowing you to aggregate tools from multiple MCP providers into a single unified endpoint.

## Architecture

```
┌─────────────────┐
│   AI Model/     │
│   LLM Client    │
└────────┬────────┘
         │ MCP Request
         ▼
┌─────────────────────────────────────┐
│      mcp-tools (Bridge Service)     │
│  ┌───────────────────────────────┐  │
│  │  Internal Tools (Serper)      │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  External Provider Bridges    │  │
│  │  - Code Sandbox MCP           │  │
│  │  - Playwright MCP             │  │
│  │  - SearXNG (future)           │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
         │ Proxied MCP Calls
         ▼
┌──────────────────────────────────────┐
│   External MCP Servers (separate)    │
│  ┌────────────────────────────────┐  │
│  │  code-sandbox-mcp:3000/mcp     │  │
│  │  (Execute code in sandboxes)   │  │
│  └────────────────────────────────┘  │
│  ┌────────────────────────────────┐  │
│  │  playwright-mcp:3000/mcp           │  │
│  │  (Browser automation)          │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

## Quick Start

### 1. Start the Full MCP Stack

```bash
# Start all MCP services + mcp-tools bridge
make mcp-with-tools

# Or start MCP services only (without bridge)
make mcp-full
```

This will start:
- **SearXNG** (http://localhost:8086) - Meta search engine
- **Vector Store MCP** (http://localhost:3015) - Lightweight embedding service for file search
- **SandboxFusion** (http://localhost:3010) - Python code interpreter
- **Code Sandbox MCP** (http://localhost:3002) - Code execution in sandboxes
- **Playwright MCP** (http://localhost:3003) - Browser automation
- **mcp-tools Bridge** (http://localhost:8091/v1/mcp) - Unified MCP endpoint

### 2. Query Available Tools

```bash
# List all tools (internal + external)
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

Expected response includes:
- `google_search` - Internal Serper tool
- `scrape` - Internal Serper tool
- `code-sandbox_*` - Tools from Code Sandbox MCP
- `playwright_*` - Tools from Playwright MCP

### 3. Call an External Tool

Example: Call a Playwright tool through the bridge

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "playwright_screenshot",
      "arguments": {
        "url": "https://example.com",
        "fullPage": true
      }
    },
    "id": 2
  }'
```

## Configuration

### MCP Provider Config File

Location: `services/mcp-tools/configs/mcp-providers.yml`

### Environment Variables

Add to your `.env` file:

```env
# Enable/disable specific MCP providers
SEARXNG_URL=http://searxng:8080

# Debug logging for MCP providers
MCP_PROVIDER_DEBUG=false
```

## Docker Compose Configuration

### Shared Network

Both `docker-compose.yml` and `docker/services-mcp.yml` use the `mcp-network` bridge to enable communication:

```yaml
# docker/services-mcp.yml
networks:
  mcp-network:
    driver: bridge

# docker-compose.yml
networks:
  mcp-network:
    external: true
    name: jan-server_mcp-network
```

### mcp-tools Service

The `mcp-tools` service is configured to:
1. Mount the config directory: `./services/mcp-tools/configs:/app/configs:ro`
2. Connect to both default and mcp-network
3. Reference MCP providers by Docker service names

```yaml
mcp-tools:
  build: ./services/mcp-tools
  environment:
    CODE_SANDBOX_URL: http://code-sandbox-mcp:3000/mcp
    PLAYWRIGHT_URL: http://playwright-mcp:3000/mcp
  networks:
    - default
    - mcp-network
  volumes:
    - ./services/mcp-tools/configs:/app/configs:ro
```

## How It Works

### Initialization Flow

1. **mcp-tools starts** → Loads `configs/mcp-providers.yml`
2. **For each enabled provider** → Creates a Bridge instance
3. **Bridge initialization** → Sends `initialize` MCP request to provider
4. **Fetch tool list** → Calls `tools/list` on each provider
5. **Register proxy tools** → Adds prefixed tools (e.g., `playwright_screenshot`) to main MCP server

### Tool Call Flow

1. **Client calls mcp-tools** → `POST /v1/mcp` with `tools/call` method
2. **mcp-tools identifies provider** → Based on tool name prefix
3. **Forward to provider** → Bridge sends MCP `tools/call` request to external server
4. **Provider executes** → Code Sandbox runs code, Playwright automates browser, etc.
5. **Return result** → Bridge forwards response back to client

### Tool Naming Convention

External tools are prefixed with their provider name:
- `code-sandbox_write_file_sandbox` - Write files into a sandbox workspace
- `code-sandbox_sandbox_exec` - Execute Python/shell commands inside the sandbox
- `playwright_navigate` - Navigate browser via Playwright MCP
- `playwright_screenshot` - Take screenshot via Playwright MCP

Internal tools keep their original names:
- `google_search` - Serper search (internal)
- `scrape` - Serper scraper (internal)

## Adding New MCP Providers

### 1. Add Provider to docker/services-mcp.yml

```yaml
services:
  my-new-mcp:
    image: my-org/my-mcp-server:latest
    ports:
      - "3004:3000"
    networks:
      - mcp-network
    profiles: ["mcp", "mcp-full"]
```

### 2. Add Provider Config

Edit `services/mcp-tools/configs/mcp-providers.yml`:

```yaml
providers:
  - name: my-new-provider
    description: Description of what this provider does
    enabled: ${MY_PROVIDER_ENABLED:-true}
    endpoint: ${MY_PROVIDER_URL:-http://my-new-mcp:3000}
    type: mcp-http
    proxy_mode: true
    timeout: 30s
```

### 3. Add Environment Variables

In `docker-compose.yml` under `mcp-tools` service:

```yaml
environment:
  MY_PROVIDER_ENABLED: ${MY_PROVIDER_ENABLED:-true}
  MY_PROVIDER_URL: ${MY_PROVIDER_URL:-http://my-new-mcp:3000}
```

### 4. Restart Services

```bash
make mcp-down-all
make mcp-with-tools
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make mcp-full` | Start MCP services only (SearXNG, Code Sandbox, Playwright) |
| `make mcp-down` | Stop MCP services |
| `make mcp-with-tools` | Start MCP services + mcp-tools bridge |
| `make mcp-down-all` | Stop all MCP-related services |

## Troubleshooting

### Check Provider Health

```bash
# Check if mcp-tools can reach providers
docker compose logs mcp-tools

# Expected output:
# "MCP provider initialized successfully" for each provider
```

### Test Individual Provider

```bash
# Test Code Sandbox MCP directly
curl -X POST http://localhost:3002 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Test Playwright MCP directly
curl -X POST http://localhost:3003 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Network Issues

```bash
# Verify mcp-tools can resolve provider hostnames
docker exec -it jan-server-mcp-tools-1 ping -c 2 code-sandbox-mcp
docker exec -it jan-server-mcp-tools-1 ping -c 2 playwright-mcp
```

### Provider Not Showing Tools

1. Check if provider is enabled in `mcp-providers.yml`
2. Verify endpoint URL is correct
3. Check logs: `docker compose logs code-sandbox-mcp`
4. Ensure network connectivity between services

## MCP Protocol Support

The bridge supports these MCP protocol methods:

| Method | Description | Status |
|--------|-------------|--------|
| `initialize` | Initialize MCP session | ✅ Supported |
| `tools/list` | List available tools | ✅ Supported |
| `tools/call` | Execute a tool | ✅ Supported |
| `ping` | Health check | ✅ Supported |
| `prompts/list` | List prompts | 🚧 TODO |
| `prompts/call` | Execute prompt | 🚧 TODO |
| `resources/list` | List resources | 🚧 TODO |
| `resources/read` | Read resource | 🚧 TODO |

## References

- **MCP Protocol Spec**: https://modelcontextprotocol.io/
- **Code Sandbox MCP**: https://github.com/philschmid/code-sandbox-mcp
- **Playwright MCP**: https://github.com/microsoft/playwright-mcp
- **SearXNG**: https://github.com/searxng/searxng
- **mcp-go SDK**: https://github.com/mark3labs/mcp-go

## Example: Full Workflow

```bash
# 1. Start everything
make mcp-with-tools

# 2. Wait for services to initialize (30-60 seconds)
sleep 60

# 3. List all available tools
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | jq

# 4. Call a Playwright tool (example)
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{
      "name":"playwright_navigate",
      "arguments":{"url":"https://github.com"}
    },
    "id":2
  }' | jq

# 5. Call a Code Sandbox tool (example)
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{
      "name":"code-sandbox_sandbox_exec",
      "arguments":{
        "container_id":"<container id from code-sandbox_sandbox_initialize>",
        "commands":["python -c \"print('Hello from Code Sandbox!')\""]
      }
    },
    "id":3
  }' | jq

# 6. Cleanup
make mcp-down-all
```
