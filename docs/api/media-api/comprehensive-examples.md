# Media API Comprehensive Examples

Complete working examples for Media API ingestion, multipart upload, `jan_*` IDs, and serving,
with JavaScript and cURL.

> The Media API has a small, fixed set of routes. There is **no** presigned-upload, resolve, or
> presign endpoint. All ingest endpoints return `{ id, mime, bytes, deduped, url }`.

## Table of Contents

- [Authentication](#authentication)
- [Ingest from Remote URL](#ingest-from-remote-url)
- [Ingest from Base64/Data URL](#ingest-from-base64data-url)
- [Multipart Upload](#multipart-upload)
- [Get Media Bytes and Metadata](#get-media-bytes-and-metadata)
- [Public Serving](#public-serving)
- [Integration with LLM API](#integration-with-llm-api)
- [Error Handling](#error-handling)

---

## Authentication

All protected Media API calls require authentication via the Kong gateway. Through Kong the
`/media` route uses `strip_path`, so `/media/media` maps to the service path `/media`.

**JavaScript:**

```javascript
const authResponse = await fetch("http://localhost:8000/llm/auth/guest-login", {
  method: "POST",
});
const { access_token: token } = await authResponse.json();
const headers = { Authorization: `Bearer ${token}` };
```

**cURL:**

```bash
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')
export TOKEN
```

---

## Ingest from Remote URL

Ingest an image from a remote URL. The Media API fetches and stores it.

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/media/media", {
  method: "POST",
  headers: { ...headers, "Content-Type": "application/json" },
  body: JSON.stringify({
    source: { type: "remote_url", url: "https://example.com/images/photo.jpg" },
    filename: "photo.jpg",
    user_id: "user_123",
  }),
});

const result = await response.json();
console.log(`Jan ID: ${result.id}`);
console.log(`MIME type: ${result.mime}`);
console.log(`Size: ${result.bytes} bytes`);
console.log(`URL: ${result.url}`);
```

**cURL:**

```bash
curl -X POST http://localhost:8000/media/media \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source": { "type": "remote_url", "url": "https://example.com/images/photo.jpg" },
    "user_id": "user_123"
  }' | jq
```

**Response:**

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "mime": "image/jpeg",
  "bytes": 45678,
  "deduped": false,
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0"
}
```

---

## Ingest from Base64/Data URL

Ingest an image from a base64-encoded data URL (useful for canvas captures or generated images).

**JavaScript:**

```javascript
const file = document.getElementById("fileInput").files[0];
const reader = new FileReader();

reader.onload = async (e) => {
  const dataUrl = e.target.result;

  const response = await fetch("http://localhost:8000/media/media", {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({
      source: { type: "data_url", data_url: dataUrl },
      user_id: "user_456",
    }),
  });

  const result = await response.json();
  console.log(`Jan ID: ${result.id}`);
};

reader.readAsDataURL(file);
```

**cURL:**

```bash
IMAGE_B64=$(base64 -w 0 image.png)
DATA_URL="data:image/png;base64,$IMAGE_B64"

curl -X POST http://localhost:8000/media/media \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"source\": { \"type\": \"data_url\", \"data_url\": \"$DATA_URL\" },
    \"user_id\": \"user_456\"
  }" | jq
```

---

## Multipart Upload

Upload a raw file directly. The `file` form field is required; `user_id` is optional. This works
for both the `s3` and `local` storage backends and returns the same response shape as
`/v1/media`.

**JavaScript:**

```javascript
const file = document.getElementById("fileInput").files[0];
const form = new FormData();
form.append("file", file);
form.append("user_id", "user_789");

const response = await fetch("http://localhost:8000/media/media/upload", {
  method: "POST",
  headers, // do NOT set Content-Type; the browser sets the multipart boundary
  body: form,
});

const result = await response.json();
console.log(`Jan ID: ${result.id}, URL: ${result.url}`);
```

**cURL:**

```bash
curl -X POST http://localhost:8000/media/media/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/image.png" \
  -F "user_id=user_789" | jq
```

**Response:**

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n1",
  "mime": "image/png",
  "bytes": 91234,
  "deduped": false,
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n1"
}
```

---

## Get Media Bytes and Metadata

### Get Media Bytes

**GET** `/v1/media/{id}` streams the content. When `MEDIA_PROXY_DOWNLOAD=false`, it returns
`{"url": "..."}` with a direct link instead of streaming.

**cURL:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/media/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0 --output image.jpg
```

### Get Media Metadata

**GET** `/v1/media/{id}/metadata`

**JavaScript:**

```javascript
const janId = "jan_01hqr8v9k2x3f4g5h6j7k8m9n0";

const response = await fetch(
  `http://localhost:8000/media/media/${janId}/metadata`,
  { headers },
);

const meta = await response.json();
console.log(`ID: ${meta.id}`);
console.log(`Content-Type: ${meta.content_type}`);
console.log(`Size: ${meta.size} bytes`);
console.log(`URL: ${meta.url}`);
```

**cURL:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/media/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0/metadata | jq
```

**Response:**

```json
{
  "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "url": "http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "content_type": "image/jpeg",
  "filename": "photo.jpg",
  "size": 45678
}
```

---

## Public Serving

**GET** `/api/media/{id}` serves the bytes with no authentication (rate-limited by Kong). Use it
directly in `<img src>` or any public context.

```html
<img src="http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0" alt="example" />
```

```bash
curl http://localhost:8000/api/media/jan_01hqr8v9k2x3f4g5h6j7k8m9n0 --output image.jpg
```

---

## Integration with LLM API

Use `jan_*` IDs in chat completions for vision-capable models.

### Complete Flow: Ingest -> Chat

**cURL:**

```bash
# Step 1: Ingest the image
JAN_ID=$(curl -s -X POST http://localhost:8000/media/media \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source": { "type": "remote_url", "url": "https://example.com/screenshot.png" },
    "user_id": "user_123"
  }' | jq -r '.id')

echo "Ingested: $JAN_ID"

# Step 2: Reference the jan_* ID in a chat completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"jan-v1-4b\",
    \"messages\": [{
      \"role\": \"user\",
      \"content\": [
        {\"type\": \"text\", \"text\": \"Describe this image\"},
        {\"type\": \"image_url\", \"image_url\": {\"url\": \"$JAN_ID\"}}
      ]
    }]
  }" | jq '.choices[0].message.content'
```

> Use any vision-capable model you have configured. The LLM API resolves the `jan_*` reference
> via `MEDIA_RESOLVE_URL`.

---

## Error Handling

All errors use the shared envelope (`{ "error": { "code", "error", "message" } }` — `code` is the
trace UUID from the platform error).

- **400 Invalid request** — malformed body or unsupported `source.type`.
- **404 Not found** — the `jan_*` ID does not exist.
- **413 Payload too large** — the content exceeds `MEDIA_MAX_BYTES`.
- **500 Storage error** — the storage backend failed.

**Example (413):**

```json
{
  "code": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
  "error": "payload too large",
  "message": "file size exceeds MEDIA_MAX_BYTES"
}
```

---

## Configuration Reference

### Environment Variables

| Variable                     | Default  | Description                                  |
| ---------------------------- | -------- | -------------------------------------------- |
| `MEDIA_API_PORT`             | 8285     | HTTP listen port                             |
| `MEDIA_STORAGE_BACKEND`      | s3       | Storage backend (`s3` or `local`)            |
| `MEDIA_MAX_BYTES`            | 52428800 | Max content size in bytes (default 50 MB)    |
| `MEDIA_PROXY_DOWNLOAD`       | true     | Stream bytes through the API vs return a URL |
| `MEDIA_RETENTION_DAYS`       | 30       | Media retention period                       |
| `MEDIA_REMOTE_FETCH_TIMEOUT` | 15s      | Timeout for fetching remote URLs             |
| `MEDIA_S3_PRESIGN_TTL`       | 168h     | S3 URL lifetime (when S3 URLs are enabled)   |

### Jan ID Format

- **Prefix:** `jan_`
- **Body:** a lowercased 26-character ULID (Crockford base32)
- **Total length:** 30 characters
- **Example:** `jan_01hqr8v9k2x3f4g5h6j7k8m9n0`
- **Properties:** globally unique, time-sortable, opaque

### Deduplication

Media is deduplicated by content hash:

- The same content returns the same `jan_*` ID.
- The ingest response includes `"deduped": true` when an existing object was reused.

---

## Related Documentation

- [Media API Reference](README.md) - Full endpoint documentation
- [Decision Guide: Upload Methods](../decision-guides.md#media-upload-methods) - Choose the best upload approach
- [Decision Guide: Jan ID System](../decision-guides.md#jan-id-system) - Understanding media identifiers
- [LLM API](../llm-api/) - Using media with vision models
- [Examples Index](../examples/README.md) - Cross-service examples
