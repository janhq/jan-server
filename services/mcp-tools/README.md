# MCP Tools Service

A standalone **Model Context Protocol (MCP)** service that provides AI models with access to external tools and capabilities.

## Features

- **MCP Protocol Support** - Full implementation of the Model Context Protocol
- **Web Search** - Pluggable engines (Serper, SearXNG, DuckDuckGo fallback) with offline and domain filters
- **Web Scraping** - Extract content from any webpage with structured metadata
- **File Search Tools** - Lightweight vector store (index + query) for MCP automations
- **Code Interpreter** - SandboxFusion-backed python_exec tool
- **Standalone Service** - Can run independently or with jan-server
- **Clean Architecture** - Domain/Infrastructure/Interfaces layers

## Architecture

`
services/mcp-tools/
├── domain/            # Business logic (transport-agnostic)
│   └── search/        # Search service interfaces and types
├── infrastructure/    # External systems integration
│   ├── config/        # Configuration management
│   ├── logger/        # Logging setup
│   └── search/        # Serper, SearXNG, fallback clients
├── interfaces/        # Delivery mechanisms
│   └── httpserver/
│       ├── middlewares/
│       └── routes/    # MCP route handlers
└── utils/
    └── mcp/          # MCP helper functions
``
services/mcp-tools/
├── domain/           # Business logic (transport-agnostic)
│   └── serper/       # Serper service interfaces and types
├── infrastructure/   # External systems integration
│   ├── config/       # Configuration management
│   ├── logger/       # Logging setup
│   └── serper/       # Serper API client implementation
├── interfaces/       # Delivery mechanisms
│   └── httpserver/   # HTTP/MCP server
│       ├── middlewares/
│       └── routes/   # MCP route handlers
└── utils/            # Utilities
    └── mcp/          # MCP helper functions
```

## Available Tools

### 1. google_search
Perform web searches via the configured engine (Serper, SearXNG, or DuckDuckGo fallback) and emit structured citations.

**Arguments:**
- `q` (required): Search query string
- `gl` (optional): Region code (ISO 3166-1 alpha-2, e.g., 'us')
- `hl` (optional): Language code (ISO 639-1, e.g., 'en')
- `location` (optional): Location for results
- `num` (optional): Number of results (default: 10)
- `tbs` (optional): Time-based filter ('qdr:h', 'qdr:d', 'qdr:w', 'qdr:m', 'qdr:y')
- `page` (optional): Page number (default: 1)
- `autocorrect` (optional): Enable autocorrect (default: true)
- `domain_allow_list` (optional): Array of domains to scope the query to (`["example.com","wikipedia.org"]`)
- `location_hint` (optional): Soft hint when upstream engines support region-aware ranking
- `offline_mode` (optional): Force cached/offline behaviour even if live engines are available

**Output:**
- JSON payload containing `results` blocks with `{ source_url, snippet, fetched_at, cache_status }`, plus a `citations` array and the raw upstream response for backward compatibility.

### 2. scrape
Scrape webpage content with metadata describing cache/fallback state.

**Arguments:**
- `url` (required): The URL to scrape
- `includeMarkdown` (optional): Return markdown format (default: false)

**Output:**
- JSON payload containing raw text, a `text_preview`, `cache_status`, and metadata describing whether the fallback fetcher was used.

### 3. file_search_index
Index arbitrary text into the lightweight vector store so that automations can cite custom documents.

**Arguments:**
- `document_id` (required): Stable identifier for the document
- `text` (required): Raw text body
- `metadata` (optional): Object that will be echoed back with search results
- `tags` (optional): Array of simple tags (e.g., `["support","guide"]`)

### 4. file_search_query
Query the vector store for the closest documents and receive citation-ready payloads.

**Arguments:**
- `query` (required): Natural language query
- `top_k` (optional): Number of hits to return (default 5, max 20)
- `document_ids` (optional): Restrict search to a subset of documents

### 5. python_exec
Execute trusted code inside SandboxFusion when a containerized interpreter is required.

**Arguments:**
- `code` (required): Script to execute
- `language` (optional): Defaults to python
- `session_id` (optional): Continue an existing SandboxFusion session
- `approved` (optional): Must be `true` when `SANDBOX_FUSION_REQUIRE_APPROVAL` is enabled

**Output:**
- JSON payload containing `stdout`, `stderr`, `duration_ms`, `session_id`, and any downloadable artifacts surfaced by SandboxFusion.

## Environment Variables

```env
HTTP_PORT=8091                    # HTTP server port
LOG_LEVEL=info                    # Log level (debug, info, warn, error)
LOG_FORMAT=json                   # Log format (json, console)
SERPER_API_KEY=your_api_key_here  # Serper API key (required)
SEARCH_ENGINE=serper              # serper or searxng
SEARXNG_URL=http://localhost:8086 # SearXNG base URL when SEARCH_ENGINE=searxng
SERPER_DOMAIN_FILTER=             # Optional CSV of domains to pin (e.g., example.com,wikipedia.org)
SERPER_LOCATION_HINT=             # Optional default location hint (e.g., California, United States)
SERPER_OFFLINE_MODE=false         # Force cached/offline search mode
VECTOR_STORE_URL=http://localhost:3015 # Base URL for the internal vector store service
SANDBOX_FUSION_URL=http://localhost:3010 # SandboxFusion container service
SANDBOX_FUSION_REQUIRE_APPROVAL=false   # Gate python_exec until manually approved
```

## Quick Start

### Local Development

```bash
cd services/mcp-tools

# Install dependencies
go mod tidy

# Run the service
go run .
```

### Docker

```bash
# Build
docker build -t mcp-tools:latest .

# Run
docker run -p 8091:8091 \
  -e SERPER_API_KEY=your_api_key \
  mcp-tools:latest
```

## Usage

### Health Check

```bash
curl http://localhost:8091/healthz
```

### MCP Request

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {
        "q": "Model Context Protocol",
        "num": 5
      }
    }
  }'
```

## Integration with LLM API

The MCP Tools service can be integrated with the llm-api service to provide tool-calling capabilities to LLM conversations.

### Add to docker-compose.yml

All MCP infrastructure services are defined in `docker/services-mcp.yml` and automatically included in the main `docker-compose.yml`.

```yaml
mcp-tools:
  build: ./services/mcp-tools
  restart: unless-stopped
  environment:
    HTTP_PORT: 8091
    SERPER_API_KEY: ${SERPER_API_KEY}
    LOG_LEVEL: info
    LOG_FORMAT: json
  ports:
    - "8091:8091"
  healthcheck:
    test: ["CMD", "wget", "--spider", "-q", "http://localhost:8091/healthz"]
    interval: 10s
    timeout: 5s
    retries: 5
```

## MCP Protocol

This service implements the [Model Context Protocol](https://modelcontextprotocol.io/), allowing AI models to:

1. **Discover Tools** - List available capabilities
2. **Call Tools** - Execute external functions
3. **Stream Results** - Receive real-time responses

Supported MCP methods:
- `initialize` - Handshake
- `tools/list` - List available tools
- `tools/call` - Execute a tool
- `ping` - Health check

## Development

### Project Structure Follows Platform Conventions

- **Clean Architecture** - Domain → Infrastructure → Interfaces
- **No HTTP in Domain** - Business logic is transport-agnostic
- **Dependency Injection** - All dependencies injected
- **Error Handling** - Structured error responses

### Adding New Tools

1. Define tool arguments in `interfaces/httpserver/routes/serper_mcp.go`
2. Add domain method in `domain/search/service.go`
3. Implement infrastructure in `infrastructure/search/client.go`
4. Register tool in `RegisterTools()` method

## Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## License

Part of the jan-server project.
