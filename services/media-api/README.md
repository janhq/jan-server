# media-api

`media-api` is the dedicated ingestion and resolution service for binary assets used by Jan Server. It accepts data URLs or remote URLs, pushes bytes to private S3-compatible storage, records metadata in Postgres, and returns short `jan_*` identifiers with presigned URLs for immediate access.

## Highlights

- Environment-driven config (`internal/config`) tailored for Menlo's S3 endpoint (`https://s3.menlo.ai`) and `platform-dev` bucket.
- PostgreSQL metadata store with schema managed by GORM.
- Automatic creation of the target database when using `postgres://` URLs.
- API-key protected routes (`X-Media-Service-Key`) plus optional observability hooks.
- Shared `utils/mediaid` package for consistent `jan_*` identifiers across services.
- Returns presigned URLs immediately upon upload for instant client access.

## Usage Flow

### Method 1: Direct Upload via API (Server-Proxied)

Client uploads an image directly through the media-api (via data URL or remote URL) and receives:
- `jan_id` - Persistent identifier for the media
- `presigned_url` - Short-lived URL for immediate access (default 5 min TTL)

```bash
curl -X POST http://localhost:8285/v1/media \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{"source":{"type":"remote_url","url":"https://placekitten.com/512/512"},"user_id":"user123"}'

# Response:
# {
#   "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
#   "mime": "image/jpeg",
#   "bytes": 45678,
#   "deduped": false,
#   "presigned_url": "https://s3.menlo.ai/platform-dev/images/jan_...?signature=..."
# }
```

**Use Case**: Simple uploads, remote URLs, or when client doesn't want to handle S3 directly.

---

### Method 2: Client-Side Direct Upload (Presigned URL)

Client requests a presigned upload URL, uploads directly to S3, then uses the `jan_id`:

#### Step 1: Request Presigned Upload URL

```bash
curl -X POST http://localhost:8285/v1/media/prepare-upload \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{"mime_type":"image/jpeg","user_id":"user123"}'

# Response:
# {
#   "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
#   "upload_url": "https://s3.menlo.ai/platform-dev/images/jan_...?X-Amz-Signature=...",
#   "mime_type": "image/jpeg",
#   "expires_in": 300
# }
```

#### Step 2: Client Uploads Directly to S3

```bash
curl -X PUT "https://s3.menlo.ai/platform-dev/images/jan_...?X-Amz-Signature=..." \
  -H "Content-Type: image/jpeg" \
  --data-binary @my-image.jpg
```

#### Step 3: Use jan_id in Completions

Client immediately uses the `jan_id` without waiting for server confirmation.

**Use Case**: Large files, faster uploads (bypass API), better for mobile/web apps.

---

### Using jan_id in LLM Completion Payload

Client injects the `jan_id` into the completion request using the format `data:image/<mime>;jan_<id>`:

```json
{
  "model": "gpt-4-vision",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "What's in this image?"},
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/jpeg;jan_01hqr8v9k2x3f4g5h6j7k8m9n0"
          }
        }
      ]
    }
  ]
}
```

---

### Backend Resolves jan_id to Fresh Presigned URL

Before forwarding to the LLM provider, the backend calls `/v1/media/resolve` to replace `jan_*` placeholders with fresh presigned URLs:

```bash
curl -X POST http://localhost:8285/v1/media/resolve \
  -H "X-Media-Service-Key: changeme-media-key" \
  -H "Content-Type: application/json" \
  -d '{"payload":{"messages":[{"content":[{"type":"image_url","image_url":{"url":"data:image/jpeg;jan_01hqr8v9k2x3f4g5h6j7k8m9n0"}}]}]}}'

# Response:
# {
#   "payload": {
#     "messages": [{
#       "content": [{
#         "type": "image_url",
#         "image_url": {
#           "url": "https://s3.menlo.ai/platform-dev/images/jan_...?signature=NEW_FRESH_SIG"
#         }
#       }]
#     }]
#   }
# }
```

## Environment variables

Populate the repo-level `.env` (via `make env-create`) and tweak the following keys:

| Variable | Description |
| --- | --- |
| `MEDIA_API_PORT` | HTTP listen port (default `8285`). |
| `MEDIA_DATABASE_URL` | Postgres DSN for metadata. |
| `MEDIA_SERVICE_KEY` | Shared secret required via `X-Media-Service-Key`. |
| `MEDIA_S3_ENDPOINT` | S3-compatible endpoint (`https://s3.menlo.ai`). |
| `MEDIA_S3_PUBLIC_ENDPOINT` | Optional public endpoint used when returning presigned URLs (e.g., `http://localhost:9000`). |
| `MEDIA_S3_ACCESS_KEY` / `MEDIA_S3_SECRET_KEY` | Credentials (`XXXXX` / `YYYY`). |
| `MEDIA_S3_BUCKET` | Target bucket (`platform-dev`). |
| `MEDIA_MAX_BYTES` | Max upload size (default 20 MB). |
| `MEDIA_S3_PRESIGN_TTL` | Lifespan of presigned URLs (default 5 min). |
| `MEDIA_RETENTION_DAYS` | Metadata retention window. |

> If the S3 bucket or credentials are omitted the service still starts, but media upload/resolve endpoints will respond with `media storage backend is not configured` until valid `MEDIA_S3_*` values are provided.

All env samples already contain the provided Menlo dev bucket configuration.

## Quick start

```bash
cd services/media-api
make run
curl -H "X-Media-Service-Key: changeme-media-key" \
  http://localhost:8285/healthz
```

## API surface

| Method & Path | Description |
| --- | --- |
| `POST /v1/media` | **Method 1**: Ingests data URL or remote URL, stores bytes privately, returns `{id, mime, bytes, deduped, presigned_url}`. |
| `POST /v1/media/prepare-upload` | **Method 2**: Generates presigned upload URL and reserves `jan_id`. Client uploads directly to S3. |
| `POST /v1/media/resolve` | Replaces `data:<mime>;jan_<id>` placeholders in arbitrary JSON with fresh presigned URLs. |
| `GET /v1/media/{id}` | Streams media bytes through the API or returns presigned URL (see `PROXY_DOWNLOAD` config). |

See `docs/swagger/swagger.yaml` for the full OpenAPI schema (regenerate with `make swagger`).

## Development scripts

- `make run` - start the service locally (loads `.env`).
- `make wire` - regenerate dependency injection graph after wiring changes.
- `make swagger` - refresh OpenAPI docs after editing handler annotations.
- `make tidy` - clean up go.mod / go.sum.

Need to integrate with `llm-api`? Wire the media client, dual-write uploads, resolve before calling the LLM, then enforce `jan_*` identifiers everywhere (see `docs/services.md` for the full flow).
