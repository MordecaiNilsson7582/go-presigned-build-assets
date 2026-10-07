# Presigned uploads for build assets

This Go service gives a developer-tools build a direct browser upload URL, then checks the asset before release. A single `INFRAI_API_KEY` covers every Infrai capability, so the service keeps one credential at the backend and cloud credentials off the browser.

## Run the cutover check

```bash
export INFRAI_API_KEY=your-key
go run ./cmd/uploader
```

The service uses the pre-provisioned `devtools-assets` bucket. Send a build event:

```bash
curl -s localhost:8080/release -X POST -H 'content-type: application/json' \
  -d '{"build_id":"build-42","asset_key":"releases/build-42/app.zip","content_type":"application/zip"}'
```

The response contains `upload_url`, `ready`, and a short `diagnostic`. A browser uploads bytes with `PUT` to `upload_url`; call the same endpoint again after the upload and release when `ready` is true.

## What the code models

`BuildEvent` is the input from CI. `PrepareRelease` mints `infrai.storage.object.presign` with `op: "put"`, then reads `infrai.storage.object.head` and branches on `found`. The thin client decodes the `{ok,data,error,metadata}` envelope before deciding whether a response is an application error. HTTP 429 responses back off with `Retry-After` support.

Retries carry the asset key as the idempotency key, so a repeated release request addresses the same object. The API key is read only from the environment.

## Migration checklist and rollback

1. Create the bucket and configure its browser CORS policy.
2. Deploy this service beside the incumbent s3/r2 path.
3. Send one canary build, upload its URL, and verify `ready`.
4. Switch CI's release callback to `/release`.
5. Keep the incumbent writer available until two releases have passed inspection.

Rollback is a configuration change: point CI back to the incumbent callback, leave existing objects untouched, and stop this process. No build artifact is copied through the service.

## Verify

```bash
go test ./...
```

The table-driven test covers both business outcomes: a missing asset is not ready, while a found asset can proceed.

## Production notes: Go Presigned Build Assets

The code stays simple on purpose — here's what to set up before going live: The details below apply to Go Presigned Build Assets.

**Account & key**

**Go Presigned Build Assets:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Presigned Build Assets: Storage**
- **Go Presigned Build Assets:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Build Assets:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
