# Vector Store Service

A lightweight HTTP service that stores document embeddings locally and exposes two endpoints:

- `POST /documents` – index a document with `{ "document_id": "doc-1", "text": "..." }`
- `POST /query` – run a semantic search with `{ "text": "foo", "top_k": 3 }`

The service keeps the documents in memory, builds a simple normalized bag-of-words embedding, and returns cosine-similarity scores to keep the stack self-contained for MCP automation testing.

## Run locally

```bash
cd services/mcp-tools/tools/vector-store-service
go run .
# Service listens on :3015 by default (override with VECTOR_STORE_PORT)
```

## Docker build

```bash
docker build -t vector-store-service .
docker run -p 3015:3015 vector-store-service

```
