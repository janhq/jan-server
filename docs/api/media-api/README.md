# Media API Documentation

The Media API handles media ingestion, storage, and resolution with S3 integration and presigned URLs.

## Quick Start

### Base URL
- **Local**: http://localhost:8285
- **Via Gateway**: http://localhost:8000/api/media
- **Docker**: http://media-api:8285

## Key Features

- **jan_* ID System** - Persistent, globally unique media identifiers
- **S3 Integration** - Stores media in S3-compatible storage (Menlo S3)
- **Presigned URLs** - Immediate access to media with time-limited URLs (5-min TTL)
- **Deduplication** - Prevents duplicate storage via content hash
- **Multiple Input Methods** - Support for remote URLs, data URLs, and direct uploads
- **PostgreSQL Metadata** - Persistent metadata storage
- **API Key Authentication** - Service key-based security

## Service Ports & Configuration

| Component | Port | Environment Variable |
|-----------|------|---------------------|
| **HTTP Server** | 8285 | `MEDIA_API_PORT` |
| **Database** | 5432 | `MEDIA_DATABASE_URL` |
| **S3 Storage** | 443 | `MEDIA_S3_ENDPOINT` |

### Required Environment Variables

```bash
MEDIA_API_PORT=8285                                    # HTTP listen port
MEDIA_DATABASE_URL=postgres://media:password@api-db:5432/media_api?sslmode=disable
MEDIA_SERVICE_KEY=changeme-media-key                  # Service authentication key
MEDIA_API_KEY=changeme-api-key                        # API authentication key

# S3 Configuration (Menlo S3)
MEDIA_S3_ENDPOINT=https://s3.menlo.ai                # S3 endpoint
MEDIA_S3_REGION=us-west-2                            # S3 region
MEDIA_S3_BUCKET=platform-dev                         # S3 bucket
MEDIA_S3_ACCESS_KEY=XXXXX                            # S3 access key
MEDIA_S3_SECRET_KEY=YYYYY                            # S3 secret key
MEDIA_S3_USE_PATH_STYLE=true                         # Use path-style URLs
```

### Optional Configuration

```bash
MEDIA_S3_PUBLIC_ENDPOINT=                            # Public S3 endpoint
MEDIA_S3_PRESIGN_TTL=5m                              # Presigned URL TTL
MEDIA_MAX_BYTES=20971520                             # Max file size (20MB default)
MEDIA_PROXY_DOWNLOAD=true                            # Proxy downloads
MEDIA_RETENTION_DAYS=30                              # Media retention period
MEDIA_REMOTE_FETCH_TIMEOUT=15s                       # Remote fetch timeout
```

## Authentication

All endpoints require the `X-Media-Service-Key` header:

```bash
-H "X-Media-Service-Key: changeme-media-key"
```

## Main Endpoints

### Upload Media

**POST** `/v1/media`

Upload media directly or from remote URL.

```bash
# Upload from remote URL
curl -X POST http://localhost:8285/v1/media \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "remote_url",
      "url": "https://example.com/image.jpg"
    },
    "user_id": "user123"
  }'

# Upload from data URL (base64 image)
curl -X POST http://localhost:8285/v1/media \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "data_url",
      "data_url": "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
    },
    "user_id": "user123"
  }'
```

**Response:**
```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "mime": "image/jpeg",
  "bytes": 45678,
  "deduped": false,
  "presigned_url": "https://s3.menlo.ai/platform-dev/images/jan_...?X-Amz-Signature=..."
}
```

### Prepare Upload (Presigned URL)

**POST** `/v1/media/prepare-upload`

Get a presigned URL for client-side S3 upload.

```bash
curl -X POST http://localhost:8285/v1/media/prepare-upload \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "image/jpeg",
    "user_id": "user123"
  }'
```

**Response:**
```json
{
  "jan_id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "presigned_url": "https://s3.menlo.ai/platform-dev/images/jan_...?X-Amz-Signature=...",
  "presigned_post": {
    "url": "https://s3.menlo.ai",
    "fields": {
      "key": "images/jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
      "policy": "...",
      "x-amz-signature": "...",
      "x-amz-date": "..."
    }
  }
}
```

### Resolve Media IDs

**POST** `/v1/media/resolve`

Resolve `jan_*` IDs to presigned URLs.

```bash
curl -X POST http://localhost:8285/v1/media/resolve \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [
      "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
      "jan_01hqr8v9k2x3f4g5h6j7k8m9n1"
    ]
  }'
```

**Response:**
```json
{
  "media": [
    {
      "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
      "presigned_url": "https://s3.menlo.ai/platform-dev/images/jan_...?X-Amz-Signature=...",
      "expires_at": "2025-11-10T10:35:00Z"
    }
  ]
}
```

### Get Media

**GET** `/v1/media/{id}`

Retrieve media metadata and presigned URL.

```bash
curl -H "X-Media-Service-Key: changeme-media-key" \
  http://localhost:8285/v1/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0
```

**Response:**
```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "mime": "image/jpeg",
  "bytes": 45678,
  "created_at": "2025-11-10T10:30:00Z",
  "presigned_url": "https://s3.menlo.ai/...",
  "expires_at": "2025-11-10T10:35:00Z"
}
```

### Health Check

**GET** `/healthz`

```bash
curl http://localhost:8285/healthz
```

## Jan ID System

**Format**: `jan_` prefix + 26-character base32 identifier

### Characteristics
- **Globally Unique**: No collision across instances
- **Sortable**: Sequential generation ensures chronological ordering
- **Opaque**: No encoded information (privacy-preserving)
- **Example**: `jan_01hqr8v9k2x3f4g5h6j7k8m9n0`

### Usage in Other Services

Reference `jan_*` IDs in LLM API for media:

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b-vision",
    "messages": [{
      "role": "user",
      "content": [
        {"type": "text", "text": "What is this?"},
        {
          "type": "image_url",
          "image_url": {"url": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0"}
        }
      ]
    }]
  }'
```

## Deduplication

Media is deduplicated by content hash (SHA-256):

- **First Upload**: Stored in S3, new `jan_*` ID created
- **Duplicate Upload**: Returns existing `jan_*` ID, skips S3 storage
- **Response**: `"deduped": true` indicates existing media

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "deduped": true
}
```

## Presigned URL Management

### TTL Configuration
Default: 5 minutes (300 seconds)

```bash
MEDIA_S3_PRESIGN_TTL=5m          # 5 minutes
MEDIA_S3_PRESIGN_TTL=30m         # 30 minutes
MEDIA_S3_PRESIGN_TTL=1h          # 1 hour
```

### Expiration
- URLs are valid for specified TTL
- Each request to resolve/get generates new presigned URL
- Expired URLs are no longer valid

## Storage Flow

### 1. Remote URL Upload
```
Client → Media API (remote_url)
    ↓
Media API → Remote Server (fetch)
    ↓
Media API → S3 (upload)
    ↓
Media API ← S3 (confirmed)
    ↓
Client ← Media API (jan_id + presigned_url)
```

### 2. Client-Side Direct Upload
```
Media API → Client (presigned_url)
    ↓
Client → S3 (direct upload)
    ↓
Client ← S3 (confirmed)
    ↓
Client → Media API (confirm/associate)
    ↓
Media API ← Database (updated)
    ↓
Client ← Media API (jan_id)
```

## Error Handling

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Invalid request | Malformed parameters |
| 401 | Unauthorized | Missing/invalid service key |
| 404 | Not found | Media ID doesn't exist |
| 413 | Payload too large | Exceeds max file size |
| 500 | S3 error | Storage operation failed |

Example error:
```json
{
  "error": {
    "message": "File size exceeds maximum allowed",
    "type": "size_error",
    "code": "max_size_exceeded"
  }
}
```

## Related Services

- **LLM API** (Port 8080) - Media resolution
- **Response API** (Port 8082) - Tool outputs
- **Kong Gateway** (Port 8000) - API routing
- **PostgreSQL** - Metadata storage
- **Menlo S3** - Media storage

## See Also

- [LLM API Documentation](../llm-api/)
- [Architecture Overview](../../architecture/)
- [Development Guide](../../guides/development.md)
