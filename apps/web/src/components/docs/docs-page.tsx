import { Loader2, AlertCircle, FileText, ExternalLink } from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { Streamdown } from "streamdown";

// Import all markdown files as raw strings at build time
const markdownFiles = import.meta.glob('/docs/**/*.md', { as: 'raw', eager: true });

export interface DocMetadata {
  title: string;
  description?: string;
  lastUpdated?: string;
}

export interface DocContent {
  metadata: DocMetadata;
  content: string;
}

export interface DocsPageProps {
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

// Static documentation content for pages that don't have markdown equivalents
const staticDocs: Record<string, { title: string; description?: string; content: string }> = {
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

// Path mapping from URL to markdown file
const markdownPathMap: Record<string, string> = {
  "/docs": "/docs/README.md",
  "/docs/quickstart": "/docs/quickstart.md",
  "/docs/api": "/docs/api/README.md",
  "/docs/api/authentication": "/docs/api/README.md",
  "/docs/api/conversations": "/docs/api/llm-api/README.md",
  "/docs/api/messages": "/docs/api/llm-api/README.md",
  "/docs/api/media": "/docs/api/media-api/README.md",
  "/docs/architecture": "/docs/architecture/README.md",
  "/docs/architecture/services": "/docs/architecture/services.md",
  "/docs/architecture/data-flow": "/docs/architecture/data-flow.md",
  "/docs/architecture/security": "/docs/architecture/security.md",
  "/docs/guides": "/docs/guides/README.md",
  "/docs/guides/development": "/docs/guides/development.md",
  "/docs/guides/testing": "/docs/guides/testing.md",
  "/docs/guides/deployment": "/docs/guides/deployment.md",
  "/docs/guides/mcp-tools": "/docs/guides/mcp-admin-interface.md",
  "/docs/configuration": "/docs/configuration/README.md",
  "/docs/configuration/environment": "/docs/configuration/env-var-mapping.md",
  "/docs/configuration/providers": "/docs/configuration/README.md",
};

export function getStaticDoc(path: string): { title: string; description?: string; content: string } | null {
  // First check if we have a markdown file for this path
  const markdownPath = markdownPathMap[path];
  if (markdownPath && markdownFiles[markdownPath]) {
    const rawContent = markdownFiles[markdownPath];
    // Extract title from first H1
    const titleMatch = rawContent.match(/^#\s+(.+)$/m);
    const title = titleMatch ? titleMatch[1] : path.split('/').pop() || 'Documentation';
    return {
      title,
      content: rawContent,
    };
  }

  // Fall back to static docs
  return staticDocs[path] || null;
}

// Get doc metadata without content (for navigation)
export function getDocMetadata(path: string): DocMetadata | null {
  const doc = getStaticDoc(path);
  if (!doc) return null;

  // Extract last updated from markdown
  const markdownPath = markdownPathMap[path];
  if (markdownPath && markdownFiles[markdownPath]) {
    const lastUpdatedMatch = markdownFiles[markdownPath].match(/\*\*Last updated[:\s]*([^*]+)\*\*/i);
    return {
      title: doc.title,
      lastUpdated: lastUpdatedMatch ? lastUpdatedMatch[1].trim() : undefined,
    };
  }

  return {
    title: doc.title,
    description: doc.description,
  };
}