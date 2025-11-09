# Media API – Build & Migration Plan

## Phase 0 – Foundations
- Scaffold `services/media-api` mirroring existing Go layout (cmd/internal/etc.).
- Introduce `.env`/Helm vars: `MEDIA_API_PORT`, `MEDIA_API_URL`, `MEDIA_API_KEY`, `MEDIA_S3_BUCKET`, `MEDIA_GCS_BUCKET`, `MEDIA_SERVICE_KEY`.
- Create shared `pkg/mediaid` for generating/validating `jan_*` ULIDs.

## Phase 1 – Ingestion Pipeline
- `POST /v1/media` accepts `{ source: { type: "data_url" | "remote_url", ... } }`.
- Server fetch/validate payloads, enforce size limits, sniff MIME (allowlist jpeg/png/webp/gif/bmp/tiff), optional re-encode.
- Compute SHA-256 and dedupe; store bytes privately (S3 SSE-KMS or GCS CMEK) under `images/<id>.<ext>`.
- Persist DB row `media_objects {id, provider, storage_key, mime, bytes, sha256, created_by, created_at, retention}`.
- Emit audit logs/metrics (bytes uploaded, dedup hits, rejection reasons).

## Phase 2 – Serving & Resolution
- `POST /v1/media/resolve` scans arbitrary JSON for `data:<mime>;jan_<id>` and swaps with short-lived presigned URLs (or full data URLs behind flag).
- Optional `GET /v1/media/:id` proxy that enforces auth and streams from storage (no bucket URLs ever exposed).
- Add presign helper with TTL cache, lifecycle policies (auto-delete after N days unless pinned), and optional malware/DLP hook.

## Phase 3 – Operations & Security
- Harden IAM: split upload vs. serve roles with least privilege; no `ListBucket`.
- Enforce `content-length-range`, `Content-MD5`, user quotas, and request tracing.
- Integrate observability (structured logs + traces) so uploads/resolves correlate with `llm-api`.
- Add admin tooling (`jan media describe <id>`, purge, reissue URL).

## Migration with Existing llm-api
1. **Introduce media client**: add `MEDIA_API_URL`/`MEDIA_API_KEY` to `services/llm-api` config loader and implement an internal client (flagged).
2. **Dual-write ingest**: update UI/backend flows to call `media-api /v1/media`, storing only `jan_*` ids while retaining legacy URLs for replay.
3. **Resolve before provider call**: serialize outbound payloads, call `/resolve`, then forward to provider; add tracing/metrics for latency.
4. **Cutover & cleanup**: require `jan_*` references in validation, remove legacy URL paths, update docs/SDKs, and tighten provider capability metadata to match behavior.

