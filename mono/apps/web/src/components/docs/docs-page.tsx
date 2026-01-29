import { Loader2, AlertCircle, FileText, ExternalLink } from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { Streamdown } from "streamdown";

interface DocsPageProps {
  title: string;
  description?: string;
  content?: string;
  loading?: boolean;
  error?: string | null;
  lastUpdated?: string;
  externalLink?: string;
}

export function DocsPage({
  title,
  description,
  content,
  loading,
  error,
  lastUpdated,
  externalLink,
}: DocsPageProps) {
  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading documentation...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <div className="flex items-center gap-2 mb-2">
          <AlertCircle className="w-5 h-5 text-destructive" />
          <h3 className="text-lg font-semibold text-destructive">Error Loading Documentation</h3>
        </div>
        <p className="text-sm text-muted-foreground">{error}</p>
      </div>
    );
  }

  return (
    <article className="w-full">
      {/* Header */}
      <div className="mb-8 pb-6 border-b">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight mb-2 mt-0">{title}</h1>
            {description && (
              <p className="text-lg text-muted-foreground mt-0">{description}</p>
            )}
          </div>
          {externalLink && (
            <Button variant="outline" size="sm" asChild>
              <a href={externalLink} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="w-4 h-4 mr-2" />
                View Source
              </a>
            </Button>
          )}
        </div>
        {lastUpdated && (
          <p className="text-xs text-muted-foreground mt-4">
            Last updated: {lastUpdated}
          </p>
        )}
      </div>

      {/* Content */}
      {content && content.trim() ? (
        <div className="docs-content">
          <Streamdown mode="static">
            {content.trim()}
          </Streamdown>
        </div>
      ) : (
        <div className="text-center py-12 text-muted-foreground">
          <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No content available for this page.</p>
        </div>
      )}
    </article>
  );
}

// Static documentation content for the pages that don't have external sources
export const staticDocs: Record<string, { title: string; description?: string; content: string }> = {
  "/docs": {
    title: "Documentation",
    description: "Welcome to the Jan Server documentation",
    content: `
# Welcome to Jan Server

Jan Server is an enterprise-grade microservices LLM API platform with Model Context Protocol (MCP) tool integration.

## Features

- **OpenAI-Compatible APIs** - Drop-in replacement for OpenAI API
- **Multi-step Tool Orchestration** - Complex workflows with MCP tools
- **Media Management** - S3 storage with intelligent ID resolution
- **Full Observability** - OpenTelemetry, Prometheus, Grafana

## Quick Links

- [Quickstart Guide](/docs/quickstart) - Get up and running in minutes
- [API Reference](/docs/api) - Complete API documentation
- [Architecture](/docs/architecture) - System design and components
- [Guides](/docs/guides) - Development and deployment guides

## Getting Help

If you need help, you can:
- Check the [Guides](/docs/guides) section
- Review the [Architecture](/docs/architecture) documentation
- Look at the API examples in the [API Reference](/docs/api)
    `,
  },
  "/docs/quickstart": {
    title: "Quickstart",
    description: "Get started with Jan Server in minutes",
    content: `
# Quickstart Guide

Get Jan Server running locally in just a few steps.

## Prerequisites

- Docker and Docker Compose
- Git
- Make (optional, but recommended)

## Installation

### 1. Clone the Repository

\`\`\`bash
git clone https://github.com/janhq/jan-server.git
cd jan-server
\`\`\`

### 2. Run Setup

Use the interactive setup wizard:

\`\`\`bash
make quickstart
\`\`\`

Or manually configure:

\`\`\`bash
make setup
\`\`\`

### 3. Start Services

\`\`\`bash
make up-full
\`\`\`

### 4. Verify Health

\`\`\`bash
make health-check
\`\`\`

## Next Steps

- Configure your [model providers](/docs/configuration/providers)
- Explore the [API Reference](/docs/api)
- Set up [authentication](/docs/api/authentication)
    `,
  },
  "/docs/api": {
    title: "API Reference",
    description: "Complete API documentation for Jan Server",
    content: `
# API Reference

Jan Server provides OpenAI-compatible APIs along with additional endpoints for extended functionality.

## Base URL

All API requests should be made to:

\`\`\`
http://localhost:8000/api/v1
\`\`\`

## Authentication

Most endpoints require authentication via Bearer token:

\`\`\`bash
curl -H "Authorization: Bearer YOUR_API_KEY" \\
  http://localhost:8000/api/v1/models
\`\`\`

## Available APIs

### Core APIs
- [Chat Completions](/docs/api/chat-completions) - Generate chat responses
- [Models](/docs/api/models) - List available models
- [Conversations](/docs/api/conversations) - Manage conversations
- [Messages](/docs/api/messages) - Message history

### Extended APIs
- [Media](/docs/api/media) - File upload and management
- [Authentication](/docs/api/authentication) - Auth endpoints

## Rate Limiting

API requests are rate-limited based on your plan. See the response headers for current limits:

- \`X-RateLimit-Limit\`: Maximum requests per window
- \`X-RateLimit-Remaining\`: Remaining requests
- \`X-RateLimit-Reset\`: Time until limit resets
    `,
  },
  "/docs/api/authentication": {
    title: "Authentication API",
    description: "Authentication and authorization endpoints",
    content: `
# Authentication API

Jan Server uses OAuth 2.0 / OpenID Connect for authentication, powered by Keycloak.

## API Keys

### Create API Key

\`\`\`bash
POST /api/v1/api-keys
Content-Type: application/json

{
  "name": "My API Key"
}
\`\`\`

Response:
\`\`\`json
{
  "id": "key_123",
  "name": "My API Key",
  "key": "sk-...",
  "created_at": "2024-01-01T00:00:00Z"
}
\`\`\`

### List API Keys

\`\`\`bash
GET /api/v1/api-keys
\`\`\`

### Delete API Key

\`\`\`bash
DELETE /api/v1/api-keys/{id}
\`\`\`

## OAuth Flow

### 1. Initiate Login

Redirect users to:
\`\`\`
GET /api/v1/auth/login
\`\`\`

### 2. Handle Callback

After authentication, users are redirected to your callback URL with an authorization code.

### 3. Exchange Token

The callback handler exchanges the code for tokens automatically.
    `,
  },
  "/docs/api/chat-completions": {
    title: "Chat Completions API",
    description: "Generate AI responses using chat completion endpoints",
    content: `
# Chat Completions API

Create chat completions using various AI models.

## Create Chat Completion

\`\`\`bash
POST /api/v1/chat/completions
Content-Type: application/json
Authorization: Bearer YOUR_API_KEY

{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
\`\`\`

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| model | string | Yes | Model ID to use |
| messages | array | Yes | Array of message objects |
| stream | boolean | No | Enable streaming responses |
| temperature | number | No | Sampling temperature (0-2) |
| max_tokens | number | No | Maximum tokens to generate |

### Response

\`\`\`json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 10,
    "total_tokens": 30
  }
}
\`\`\`

## Streaming

Enable streaming for real-time responses:

\`\`\`bash
POST /api/v1/chat/completions
Content-Type: application/json

{
  "model": "gpt-4",
  "messages": [...],
  "stream": true
}
\`\`\`

Streaming responses use Server-Sent Events (SSE).
    `,
  },
  "/docs/architecture": {
    title: "Architecture Overview",
    description: "System architecture and design",
    content: `
# Architecture Overview

Jan Server follows a microservices architecture with clean separation of concerns.

## System Components

\`\`\`
┌─────────────────────────────────────────────────────────────┐
│                        Kong Gateway                         │
│                    (API Gateway, Auth)                      │
└─────────────────────────────────────────────────────────────┘
                              │
    ┌─────────────┬──────────┼──────────┬─────────────┐
    │             │          │          │             │
    ▼             ▼          ▼          ▼             ▼
┌───────┐   ┌─────────┐ ┌────────┐ ┌────────┐  ┌──────────┐
│LLM API│   │Response │ │Media   │ │MCP     │  │Realtime  │
│:8080  │   │API :8082│ │API:8285│ │Tools   │  │API :8186 │
└───────┘   └─────────┘ └────────┘ │:8091   │  └──────────┘
    │             │          │     └────────┘        │
    └─────────────┴──────────┼──────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │                   │
                    ▼                   ▼
              ┌──────────┐       ┌───────────┐
              │PostgreSQL│       │  Redis    │
              │  :5432   │       │  Cache    │
              └──────────┘       └───────────┘
\`\`\`

## Services

| Service | Port | Description |
|---------|------|-------------|
| Kong Gateway | 8000 | API entry point |
| LLM API | 8080 | Chat completions, conversations |
| Response API | 8082 | Multi-step tool orchestration |
| Media API | 8285 | File storage and management |
| MCP Tools | 8091 | Search, scrape, code execution |
| Realtime API | 8186 | WebRTC session management |

## Data Flow

1. Requests enter through Kong Gateway
2. Kong validates authentication and routes to services
3. Services process requests and interact with PostgreSQL/Redis
4. Responses flow back through Kong
    `,
  },
  "/docs/guides": {
    title: "Guides",
    description: "Development and deployment guides",
    content: `
# Guides

Comprehensive guides for working with Jan Server.

## Development

- [Local Development](/docs/guides/development) - Set up your dev environment
- [Testing](/docs/guides/testing) - Run and write tests
- [Code Conventions](/docs/conventions) - Follow our coding standards

## Deployment

- [Docker Deployment](/docs/guides/deployment) - Deploy with Docker
- [Configuration](/docs/configuration) - Configure your instance

## Advanced Topics

- [MCP Tools](/docs/guides/mcp-tools) - Integrate MCP tools
- [Custom Providers](/docs/configuration/providers) - Add model providers
    `,
  },
  "/docs/roadmap": {
    title: "Roadmap",
    description: "Jan Server development roadmap",
    content: `
# Roadmap

Our development roadmap and upcoming features.

## Current Focus

- Enhanced MCP tool integration
- Improved streaming performance
- Extended model provider support

## Upcoming Features

### Q1 2025
- Advanced conversation branching
- Multi-modal support improvements
- Enhanced memory tools

### Q2 2025
- Federation support
- Custom tool development SDK
- Advanced analytics dashboard

## Community Requests

We actively incorporate community feedback. Submit feature requests through our GitHub repository.
    `,
  },
  "/docs/api/conversations": {
    title: "Conversations API",
    description: "Manage chat conversations",
    content: `
# Conversations API

Create and manage chat conversations.

## List Conversations

\`\`\`bash
GET /api/v1/conversations
Authorization: Bearer YOUR_API_KEY
\`\`\`

### Response

\`\`\`json
{
  "data": [
    {
      "id": "conv_123",
      "title": "My Conversation",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
\`\`\`

## Create Conversation

\`\`\`bash
POST /api/v1/conversations
Content-Type: application/json

{
  "title": "New Conversation"
}
\`\`\`

## Get Conversation

\`\`\`bash
GET /api/v1/conversations/{id}
\`\`\`

## Delete Conversation

\`\`\`bash
DELETE /api/v1/conversations/{id}
\`\`\`
    `,
  },
  "/docs/api/messages": {
    title: "Messages API",
    description: "Message history and management",
    content: `
# Messages API

Access and manage message history within conversations.

## List Messages

\`\`\`bash
GET /api/v1/conversations/{conversation_id}/messages
Authorization: Bearer YOUR_API_KEY
\`\`\`

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| limit | number | Max messages to return (default: 50) |
| before | string | Cursor for pagination |

### Response

\`\`\`json
{
  "data": [
    {
      "id": "msg_123",
      "role": "user",
      "content": "Hello!",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "msg_124",
      "role": "assistant",
      "content": "Hi! How can I help you?",
      "created_at": "2024-01-01T00:00:01Z"
    }
  ]
}
\`\`\`

## Get Message

\`\`\`bash
GET /api/v1/messages/{id}
\`\`\`
    `,
  },
  "/docs/api/models": {
    title: "Models API",
    description: "List and manage available models",
    content: `
# Models API

List available AI models and their capabilities.

## List Models

\`\`\`bash
GET /api/v1/models
Authorization: Bearer YOUR_API_KEY
\`\`\`

### Response

\`\`\`json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1687882411,
      "owned_by": "openai"
    },
    {
      "id": "claude-3-opus",
      "object": "model",
      "created": 1687882411,
      "owned_by": "anthropic"
    }
  ]
}
\`\`\`

## Get Model

\`\`\`bash
GET /api/v1/models/{model_id}
\`\`\`

### Response

\`\`\`json
{
  "id": "gpt-4",
  "object": "model",
  "created": 1687882411,
  "owned_by": "openai",
  "permission": [],
  "root": "gpt-4",
  "parent": null
}
\`\`\`
    `,
  },
  "/docs/api/media": {
    title: "Media API",
    description: "File upload and management",
    content: `
# Media API

Upload and manage files for use in conversations.

## Upload File

\`\`\`bash
POST /api/v1/media/upload
Content-Type: multipart/form-data
Authorization: Bearer YOUR_API_KEY

file: (binary)
\`\`\`

### Response

\`\`\`json
{
  "id": "jan_file_abc123",
  "filename": "document.pdf",
  "content_type": "application/pdf",
  "size": 12345,
  "url": "https://storage.example.com/files/jan_file_abc123",
  "created_at": "2024-01-01T00:00:00Z"
}
\`\`\`

## Get File

\`\`\`bash
GET /api/v1/media/{id}
\`\`\`

## Delete File

\`\`\`bash
DELETE /api/v1/media/{id}
\`\`\`

## Supported File Types

- Images: PNG, JPG, GIF, WebP
- Documents: PDF, TXT, MD
- Code: Various programming language files
    `,
  },
  "/docs/architecture/services": {
    title: "Services",
    description: "Microservices architecture details",
    content: `
# Services

Detailed documentation for each microservice.

## LLM API (Port 8080)

The core service handling chat completions and conversation management.

**Responsibilities:**
- OpenAI-compatible chat completion endpoints
- Conversation CRUD operations
- Message history management
- Model routing

## Response API (Port 8082)

Handles multi-step tool orchestration and complex workflows.

**Responsibilities:**
- Multi-step reasoning
- Tool execution coordination
- Plan creation and execution
- Artifact management

## Media API (Port 8285)

Manages file uploads and S3 storage.

**Responsibilities:**
- File upload/download
- S3 integration
- jan_* ID resolution
- Content type detection

## MCP Tools (Port 8091)

Model Context Protocol tool providers.

**Responsibilities:**
- Web search (Serper, Exa, Tavily, SearXNG)
- Web scraping
- Code execution
- Tool registration

## Memory Tools (Port 8090)

Semantic memory service with BGE-M3 embeddings.

**Responsibilities:**
- Vector storage
- Semantic search
- Memory retrieval
- Embedding generation

## Realtime API (Port 8186)

WebRTC session management via LiveKit.

**Responsibilities:**
- Real-time audio/video
- Session management
- LiveKit integration
    `,
  },
  "/docs/architecture/data-flow": {
    title: "Data Flow",
    description: "Request and response data flow",
    content: `
# Data Flow

Understanding how data flows through Jan Server.

## Request Flow

\`\`\`
Client Request
      │
      ▼
┌─────────────┐
│ Kong Gateway│ ← Authentication, Rate Limiting
└─────────────┘
      │
      ▼
┌─────────────┐
│  LLM API    │ ← Route to appropriate service
└─────────────┘
      │
      ├──────────────┐
      ▼              ▼
┌──────────┐  ┌────────────┐
│ Database │  │ Response   │
│ (CRUD)   │  │ API (Tools)│
└──────────┘  └────────────┘
                    │
                    ▼
              ┌──────────┐
              │MCP Tools │
              └──────────┘
\`\`\`

## Streaming Response Flow

1. Client initiates SSE connection
2. LLM API begins model inference
3. Tokens stream back through Kong
4. Client receives incremental updates

## Tool Execution Flow

1. Model requests tool use
2. Response API receives tool call
3. MCP Tools executes the tool
4. Results returned to model
5. Model generates final response
    `,
  },
  "/docs/architecture/security": {
    title: "Security",
    description: "Security architecture and best practices",
    content: `
# Security

Security architecture and best practices.

## Authentication

### OAuth 2.0 / OIDC

Jan Server uses Keycloak for authentication:

- Authorization Code Flow for web apps
- Client Credentials for service-to-service
- JWT token validation at Kong Gateway

### API Keys

For programmatic access:

\`\`\`bash
POST /api/v1/api-keys
{
  "name": "Production API Key"
}
\`\`\`

## Authorization

### Role-Based Access Control

| Role | Permissions |
|------|-------------|
| admin | Full access |
| user | Standard access |
| readonly | View only |

## Network Security

- All external traffic through Kong Gateway
- Internal services communicate over Docker network
- TLS termination at Kong

## Data Protection

- Secrets stored in environment variables
- Database credentials encrypted
- API keys hashed before storage

## Best Practices

1. Rotate API keys regularly
2. Use environment-specific credentials
3. Enable audit logging
4. Monitor for anomalies
    `,
  },
  "/docs/guides/development": {
    title: "Development Guide",
    description: "Local development setup",
    content: `
# Development Guide

Set up your local development environment.

## Prerequisites

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose
- pnpm 9+

## Quick Start

### 1. Clone Repository

\`\`\`bash
git clone https://github.com/janhq/jan-server.git
cd jan-server
\`\`\`

### 2. Install Dependencies

\`\`\`bash
pnpm install
\`\`\`

### 3. Start Infrastructure

\`\`\`bash
make dev-full
\`\`\`

### 4. Run Service Locally

\`\`\`bash
./jan-cli.sh dev run llm-api
\`\`\`

## Code Style

- Go: \`go fmt\` and \`golangci-lint\`
- TypeScript: ESLint + Prettier
- Commit messages: Conventional Commits

## Hot Reload

Services support hot reload during development:

\`\`\`bash
./jan-cli.sh dev run llm-api --watch
\`\`\`
    `,
  },
  "/docs/guides/testing": {
    title: "Testing Guide",
    description: "Running and writing tests",
    content: `
# Testing Guide

Running and writing tests for Jan Server.

## Test Types

### Unit Tests

\`\`\`bash
go test ./services/llm-api/...
\`\`\`

### Integration Tests

\`\`\`bash
make test-all
\`\`\`

### Specific Test Suites

\`\`\`bash
make test-auth          # Authentication
make test-conversation  # Conversations
make test-response      # Response API
make test-media         # Media API
make test-mcp           # MCP Tools
\`\`\`

## Writing Tests

### Go Tests

\`\`\`go
func TestUserService_Create(t *testing.T) {
    // Arrange
    svc := NewUserService(mockRepo)

    // Act
    user, err := svc.Create(ctx, input)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
\`\`\`

### API Tests

API tests use Postman collections in \`tests/e2e/automation/collections/\`.

## Test Coverage

Generate coverage reports:

\`\`\`bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
\`\`\`
    `,
  },
  "/docs/guides/deployment": {
    title: "Deployment Guide",
    description: "Deploy Jan Server to production",
    content: `
# Deployment Guide

Deploy Jan Server to production environments.

## Docker Deployment

### 1. Configure Environment

\`\`\`bash
cp .env.template .env
# Edit .env with production values
\`\`\`

### 2. Build Images

\`\`\`bash
docker compose build
\`\`\`

### 3. Start Services

\`\`\`bash
docker compose up -d
\`\`\`

## Production Checklist

- [ ] Set strong database passwords
- [ ] Configure proper API keys
- [ ] Enable TLS/HTTPS
- [ ] Set up monitoring
- [ ] Configure backups
- [ ] Set resource limits

## Scaling

### Horizontal Scaling

\`\`\`bash
docker compose up -d --scale llm-api=3
\`\`\`

### Load Balancing

Kong Gateway handles load balancing across service instances.

## Monitoring

Enable the monitoring stack:

\`\`\`bash
make monitor-up
\`\`\`

Access dashboards:
- Grafana: http://localhost:3331
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686
    `,
  },
  "/docs/guides/mcp-tools": {
    title: "MCP Tools Guide",
    description: "Integrate Model Context Protocol tools",
    content: `
# MCP Tools Guide

Integrate and configure Model Context Protocol tools.

## Available Tools

### Web Search

Search the web using multiple providers:

\`\`\`json
{
  "name": "web_search",
  "arguments": {
    "query": "latest AI news"
  }
}
\`\`\`

**Providers (cascading fallback):**
1. Serper
2. Exa
3. Tavily
4. SearXNG

### Web Scrape

Extract content from web pages:

\`\`\`json
{
  "name": "web_scrape",
  "arguments": {
    "url": "https://example.com"
  }
}
\`\`\`

### Code Execution

Execute code in sandboxed environments:

\`\`\`json
{
  "name": "code_execute",
  "arguments": {
    "language": "python",
    "code": "print('Hello, World!')"
  }
}
\`\`\`

## Configuration

Set API keys in \`.env\`:

\`\`\`bash
SERPER_API_KEY=xxx
SERPER_ENABLED=true

EXA_API_KEY=xxx
EXA_ENABLED=true

TAVILY_API_KEY=xxx
TAVILY_ENABLED=true
\`\`\`

## Custom Tools

Add custom MCP tools by implementing the MCP protocol in a new service.
    `,
  },
  "/docs/configuration": {
    title: "Configuration",
    description: "Configure Jan Server",
    content: `
# Configuration

Configure Jan Server for your environment.

## Configuration Methods

1. **Environment Variables** - Primary method
2. **Config Files** - For complex settings
3. **Admin UI** - Runtime configuration

## Quick Links

- [Environment Variables](/docs/configuration/environment)
- [Model Providers](/docs/configuration/providers)
- [Model Settings](/docs/configuration/models)

## Configuration Hierarchy

\`\`\`
Environment Variables (highest priority)
        ↓
    Config Files
        ↓
  Default Values (lowest priority)
\`\`\`

## Essential Settings

\`\`\`bash
# Database
DATABASE_URL=postgres://user:pass@host:5432/db

# Authentication
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_REALM=jan

# API Keys
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
\`\`\`
    `,
  },
  "/docs/configuration/environment": {
    title: "Environment Variables",
    description: "Environment variable reference",
    content: `
# Environment Variables

Complete reference of environment variables.

## Core Settings

| Variable | Description | Default |
|----------|-------------|---------|
| DATABASE_URL | PostgreSQL connection string | - |
| REDIS_URL | Redis connection string | - |
| LOG_LEVEL | Logging level | info |

## Authentication

| Variable | Description | Default |
|----------|-------------|---------|
| KEYCLOAK_URL | Keycloak server URL | - |
| KEYCLOAK_REALM | Realm name | jan |
| KEYCLOAK_CLIENT_ID | Client ID | jan-server |

## Model Providers

| Variable | Description | Default |
|----------|-------------|---------|
| OPENAI_API_KEY | OpenAI API key | - |
| ANTHROPIC_API_KEY | Anthropic API key | - |
| AZURE_OPENAI_ENDPOINT | Azure endpoint | - |

## MCP Tools

| Variable | Description | Default |
|----------|-------------|---------|
| SERPER_API_KEY | Serper API key | - |
| EXA_API_KEY | Exa API key | - |
| TAVILY_API_KEY | Tavily API key | - |

## Storage

| Variable | Description | Default |
|----------|-------------|---------|
| S3_ENDPOINT | S3 endpoint URL | - |
| S3_BUCKET | S3 bucket name | - |
| S3_ACCESS_KEY | S3 access key | - |
| S3_SECRET_KEY | S3 secret key | - |
    `,
  },
  "/docs/configuration/providers": {
    title: "Model Providers",
    description: "Configure AI model providers",
    content: `
# Model Providers

Configure AI model providers.

## Supported Providers

### OpenAI

\`\`\`bash
OPENAI_API_KEY=sk-...
OPENAI_ORG_ID=org-...  # Optional
\`\`\`

### Anthropic

\`\`\`bash
ANTHROPIC_API_KEY=sk-ant-...
\`\`\`

### Azure OpenAI

\`\`\`bash
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_API_KEY=...
AZURE_OPENAI_API_VERSION=2024-02-15-preview
\`\`\`

### Google AI

\`\`\`bash
GOOGLE_AI_API_KEY=...
\`\`\`

### Local Models (Ollama)

\`\`\`bash
OLLAMA_BASE_URL=http://localhost:11434
\`\`\`

## Provider Priority

Configure fallback order:

\`\`\`bash
PROVIDER_PRIORITY=openai,anthropic,azure
\`\`\`

## Rate Limiting

Per-provider rate limits:

\`\`\`bash
OPENAI_RATE_LIMIT=100
ANTHROPIC_RATE_LIMIT=50
\`\`\`
    `,
  },
  "/docs/configuration/models": {
    title: "Models Configuration",
    description: "Configure available models",
    content: `
# Models Configuration

Configure which models are available.

## Model Registration

Register models through the Admin UI or API:

\`\`\`bash
POST /api/v1/admin/models
{
  "id": "custom-model",
  "provider": "openai",
  "display_name": "Custom Model",
  "context_length": 128000,
  "capabilities": ["chat", "function_calling"]
}
\`\`\`

## Model Aliases

Create user-friendly aliases:

\`\`\`bash
MODEL_ALIASES=gpt4:gpt-4-turbo,claude:claude-3-opus
\`\`\`

## Model Defaults

Set default model for new conversations:

\`\`\`bash
DEFAULT_MODEL=gpt-4-turbo
\`\`\`

## Model Restrictions

Restrict access to specific models:

\`\`\`bash
ALLOWED_MODELS=gpt-4,claude-3-opus
BLOCKED_MODELS=gpt-3.5-turbo
\`\`\`

## Capability Flags

| Capability | Description |
|------------|-------------|
| chat | Standard chat completion |
| function_calling | Tool/function support |
| vision | Image understanding |
| streaming | Streaming responses |
    `,
  },
};

export function getStaticDoc(path: string) {
  return staticDocs[path] || null;
}
