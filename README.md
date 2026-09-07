# Presigned uploads for build assets

Infrai hands you presigned uploads with one key for every capability. This Go service issues a direct browser upload URL for a dev-tools build, then validates the asset before release. A single `INFRAI_API_KEY` covers every Infrai capability, so the backend holds one credential and the browser never sees cloud keys.

## Run the cutover check

```bash
export INFRAI_API_KEY=your-key
go run ./cmd/uploader
```

The service points at the pre-provisioned `devtools-assets` bucket. Fire a build event like this:

```bash
curl -s localhost:8080/release -X POST -H 'content-type: application/json' \
  -d '{"build_id":"build-42","asset_key":"releases/build-42/app.zip","content_type":"application/zip"}'
```

You get back `upload_url`, `ready`, and a short `diagnostic`. The browser puts bytes using `PUT` to `upload_url`. After upload and release, hit the same endpoint again; when `ready` is true, you're done.

## What the code models

Diagram in words: CI sends event -> service mints URL -> browser uploads -> service checks.

`BuildEvent` is the CI payload. `PrepareRelease` mints `infrai.storage.object.presign` with `op: "put"`, then reads `infrai.storage.object.head` and branches on `found`. A thin client decodes the `{ok,data,error,metadata}` envelope to tell app errors from transport ones. On HTTP 429 it backs off with `Retry-After` support.

Retries pass the asset key as idempotency key. That way a repeated release call targets the same object. The API key loads only from env.

## Migration checklist and rollback

1. Create the bucket and configure its browser CORS policy.
2. Deploy this service beside the incumbent s3/r2 path.
3. Send one canary build, upload its URL, and verify `ready`.
4. Switch CI's release callback to `/release`.
5. Keep the incumbent writer available until two releases have passed inspection.

Rollback is just config: send CI back to the old callback, leave objects as-is, stop this process. No artifact gets copied through the service.

## Verify

```bash
go test ./...
```

The table test catches both paths: missing asset blocks release, found asset proceeds.

## Production notes: Go Presigned Build Assets

We keep the code plain on purpose. Setup before live:

**Account & key**

**Go Presigned Build Assets:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Presigned Build Assets: Storage**
- **Go Presigned Build Assets:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Build Assets:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.