# Media API Documentation

The Media API handles image uploads and storage.

## Quick Start

Examples: [API examples index](../examples/README.md) includes uploads, jan\_\* IDs, and OCR/preview flows.

### URLs

- **Direct access**: http://localhost:8285
- **Through gateway**: http://localhost:8000/media (Kong prefixes `/media` before forwarding)
- **Inside Docker**: http://media-api:8285

## What You Can Do

- **Ingest images** - From remote URLs, base64 data URLs, or multipart upload. See [Upload Method Guide](../decision-guides.md#media-upload-methods) to choose the best approach.
- **Get jan\_\* IDs** - Unique identifiers for each image. See [Jan ID System Guide](../decision-guides.md#jan-id-system) to understand how they work.
- **Serve media** - Fetch bytes by ID (authenticated) or embed via the public `/api/media/{id}` route.
- **Prevent duplicates** - The same content uploaded twice returns the same `jan_*` ID.
- **Store in S3 or locally** - Backend selected by `MEDIA_STORAGE_BACKEND`.

## Service Ports & Configuration

| Component                          | Port | Key Environment Variables                                                                                                                 |
| ---------------------------------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **HTTP Server**                    | 8285 | `MEDIA_API_PORT`                                                                                                                          |
| **Database (PostgreSQL)**          | 5432 | `DB_POSTGRESQL_WRITE_DSN`, `DB_POSTGRESQL_READ1_DSN` (optional replica)                                                                   |
| **Object Storage (S3-compatible)** | 443  | `MEDIA_STORAGE_BACKEND` (`s3` or `local`), `MEDIA_S3_ENDPOINT`, `MEDIA_S3_BUCKET`, `MEDIA_S3_ACCESS_KEY_ID`, `MEDIA_S3_SECRET_ACCESS_KEY` |

### Required Environment Variables

```bash
# Core service + database
MEDIA_API_PORT=8285
DB_POSTGRESQL_WRITE_DSN=postgres://media:password@api-db:5432/media_api?sslmode=disable
# Optional read replica
DB_POSTGRESQL_READ1_DSN=postgres://media_ro:password@api-db-ro:5432/media_api?sslmode=disable

# Auth (enable when fronted by Kong)
AUTH_ENABLED=true
AUTH_ISSUER=http://localhost:8085/realms/jan
ACCOUNT=account
AUTH_JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs

# Storage backend selection
MEDIA_STORAGE_BACKEND=s3    # or "local"

# S3 configuration (required when MEDIA_STORAGE_BACKEND=s3)
MEDIA_S3_BUCKET=platform-dev
MEDIA_S3_REGION=us-west-2
MEDIA_S3_ENDPOINT=https://s3.menlo.ai
MEDIA_S3_ACCESS_KEY_ID=XXXXX
MEDIA_S3_SECRET_ACCESS_KEY=YYYYY
MEDIA_S3_USE_PATH_STYLE=true
```

### Optional Configuration

```bash
# Public endpoint for download links (falls back to MEDIA_S3_ENDPOINT when empty)
MEDIA_S3_PUBLIC_ENDPOINT=https://cdn.example.com
# Presigned URL lifetime
MEDIA_S3_PRESIGN_TTL=168h
# Upload limits + retention
MEDIA_MAX_BYTES=20971520      # 20 MB
MEDIA_RETENTION_DAYS=30
MEDIA_REMOTE_FETCH_TIMEOUT=15s
# Download behavior
MEDIA_PROXY_DOWNLOAD=true     # stream bytes through the API instead of redirecting

# Local filesystem backend overrides (when MEDIA_STORAGE_BACKEND=local)
MEDIA_LOCAL_STORAGE_PATH=./media-data
MEDIA_LOCAL_STORAGE_BASE_URL=http://localhost:8285/v1/files
```

## Authentication

All endpoints require authentication through the Kong gateway.

**For complete authentication documentation, see [Authentication Guide](../README.md#authentication)**

**Quick example:**

```bash
# Get guest token
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')

# Use in requests (Kong strips /media, so /media/media -> service /media)
curl -H "Authorization: Bearer $TOKEN" \
 http://localhost:8000/media/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0/metadata
```

**Key points:**

- Use Kong gateway (port 8000) for all client requests: `http://localhost:8000/media/...`
- Both Bearer tokens and API keys (`X-API-Key`) work through Kong
- Direct service access (port 8285) requires valid JWT token

Direct calls to port 8285 still honor JWT validation when `AUTH_ENABLED=true` on the service. Use the gateway whenever possible so rate-limiting/cors policies apply consistently.

## Endpoints

The Media API exposes a small, fixed set of routes. There is **no** presigned-upload, resolve,
presign, or bulk-delete endpoint. (`/v1/files`, `/v1/files/upload`, `/v1/files/{id}`, and
`/v1/files/{id}/metadata` exist as aliases of the `/v1/media*` routes below.)

| Service path              | Method | Description                                                  |
| ------------------------- | ------ | ------------------------------------------------------------ |
| `/v1/media`               | POST   | Ingest a data URL or remote URL; returns the `jan_*` ID      |
| `/v1/media/upload`        | POST   | Multipart file upload; returns the `jan_*` ID                |
| `/v1/media/{id}`          | GET    | Stream the media bytes (or a direct URL if proxying is off)  |
| `/v1/media/{id}/metadata` | GET    | Get media metadata                                           |
| `/api/media/{id}`         | GET    | Public read-only serving via Kong (no auth; used in `img src`) |

> Via Kong the protected routes are reached under `/media` (`strip_path`), e.g.
> `POST http://localhost:8000/media/media`. The public route is `GET http://localhost:8000/api/media/{id}`.

### Ingest Media

**POST** `/v1/media`

Ingest media from a remote URL or a base64 data URL. The request body is an `IngestRequest`
with a `source` whose `type` is `remote_url` or `data_url`.

```bash
# From a remote URL
curl -X POST http://localhost:8000/media/media \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
   "source": { "type": "remote_url", "url": "https://example.com/image.jpg" },
   "filename": "image.jpg",
   "user_id": "user123"
 }'

# From a data URL (base64)
curl -X POST http://localhost:8000/media/media \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
   "source": { "type": "data_url", "data_url": "data:image/jpeg;base64,/9j/4AAQSkZJRg..." },
   "user_id": "user123"
 }'
```

**Response** (same shape for `/v1/media` and `/v1/media/upload`):

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "mime": "image/jpeg",
  "bytes": 45678,
  "deduped": false,
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0"
}
```

The `url` is a direct, embeddable link served through the public `/api/media/{id}` route.

### Multipart Upload

**POST** `/v1/media/upload`

Upload a raw file (any storage backend). The `file` form field is required; `user_id` is optional.

```bash
curl -X POST http://localhost:8000/media/media/upload \
 -H "Authorization: Bearer <token>" \
 -F "file=@/path/to/image.png" \
 -F "user_id=user123"
```

Returns the same `{id, mime, bytes, deduped, url}` body as `/v1/media`.

### Get Media Bytes

**GET** `/v1/media/{id}`

Streams the media content. If `MEDIA_PROXY_DOWNLOAD=false`, returns `{"url": "..."}` with a
direct link instead of streaming.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/media/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0 --output image.jpg
```

### Get Media Metadata

**GET** `/v1/media/{id}/metadata`

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/media/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0/metadata
```

**Response:**

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "content_type": "image/jpeg",
  "filename": "image.jpg",
  "size": 45678
}
```

### Public Serving

**GET** `/api/media/{id}` (no authentication)

Serves the media bytes directly through Kong. Use this URL in `<img src>` or other public
contexts.

```bash
curl http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0 --output image.jpg
```

### Health Check

**GET** `/healthz`

```bash
# Via gateway
curl http://localhost:8000/media/healthz

# Direct service port
curl http://localhost:8285/healthz
```

## Jan ID System

**Format**: `jan_` prefix + a lowercased 26-character ULID (Crockford base32).

### Characteristics

- **Globally Unique**: ULID-based, no collision across instances
- **Sortable**: ULIDs are time-ordered, so IDs sort chronologically
- **Opaque**: Treat as an identifier; do not parse meaning from it
- **Example**: `jan_01hqr8v9k2x3f4g5h6j7k8m9n0` (the part after `jan_` is exactly 26 chars)

### Usage in Other Services

Reference `jan_*` IDs in the LLM API for media:

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
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
  "mime": "image/jpeg",
  "bytes": 45678,
  "deduped": true,
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0"
}
```

## Storage Backends

The backend is chosen by `MEDIA_STORAGE_BACKEND` (`s3` or `local`).

- **S3** (`s3`): objects are stored in the configured bucket. When `MEDIA_S3_URL_ENABLED=true`,
  the service may hand back S3 URLs whose lifetime is governed by `MEDIA_S3_PRESIGN_TTL`
  (default `168h`). Otherwise bytes are served through the API / public `/api/media/{id}` route
  (controlled by `MEDIA_PROXY_DOWNLOAD`).
- **Local** (`local`): bytes are stored under `MEDIA_LOCAL_STORAGE_PATH` and served by ID.

### Upload & Fetch Flow

```
# Ingest (remote URL or data URL)
Client -> POST /v1/media -> Media API fetches/decodes -> stores in backend
Client <- { id: jan_*, url: /api/media/jan_* }

# Multipart upload
Client -> POST /v1/media/upload (file) -> Media API stores in backend
Client <- { id: jan_*, url: /api/media/jan_* }

# Fetch later
Client -> GET /v1/media/{id}            # authenticated stream
Anyone -> GET /api/media/{id}           # public stream (e.g. <img src>)
```

## Error Handling

| Status | Error             | Cause                        |
| ------ | ----------------- | ---------------------------- |
| 400    | Invalid request   | Malformed parameters         |
| 401    | Unauthorized      | Missing/invalid bearer token |
| 404    | Not found         | Media ID doesn't exist       |
| 413    | Payload too large | Exceeds max file size        |
| 500    | S3 error          | Storage operation failed     |

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
