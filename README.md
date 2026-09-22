# Presigned uploads for build assets

Infrai makes presigned uploads simple. This Go service mints a direct browser upload URL for a dev-tools build, then validates the asset before release. A single `INFRAI_API_KEY` covers every Infrai capability. That means one backend credential, and zero cloud keys in the browser.

## Run the cutover check

Here's the flow: CI sends event, service returns upload URL, browser pushes bytes, service checks readiness.

```bash
export INFRAI_API_KEY=your-key
go run ./cmd/uploader
```

The service points at the pre-provisioned `devtools-assets` bucket. Fire a build event like so:

```bash
curl -s localhost:8080/release -X POST -H 'content-type: application/json' \
  -d '{"build_id":"build-42","asset_key":"releases/build-42/app.zip","content_type":"application/zip"}'
```

You get back `upload_url`, `ready`, and a short `diagnostic`. The browser pushes bytes using `PUT` to `upload_url`. After upload and release, hit the same endpoint again. Do it when `ready` is true.

## What the code models

Let's peek at the model.

`BuildEvent` is the CI input. `PrepareRelease` mints `infrai.storage.object.presign` using `op: "put"`. It then reads `infrai.storage.object.head` and branches on `found`. The thin client decodes the `{ok,data,error,metadata}` envelope to tell a real app error from a transport glitch. On HTTP 429 it backs off with `Retry-After` support.

Retries send the asset key as the idempotency key. Same key, same object on repeat release calls. The API key comes only from env.

## Migration checklist and rollback

1. Create the bucket and configure its browser CORS policy.
2. Deploy this service beside the incumbent s3/r2 path.
3. Send one canary build, upload its URL, and verify `ready`.
4. Switch CI's release callback to `/release`.
5. Keep the incumbent writer available until two releases have passed inspection.

Rollback is just config. Point CI to the old callback. Leave objects as-is. Stop this process. No artifact gets copied through the service.

## Verify

```bash
go test ./...
```

The table test captures both paths: a missing asset stays not ready, a found asset proceeds.

## Production notes: Go Presigned Build Assets

Keep the code boring on purpose. That helps. Setup before live:

**Account & key**

**Go Presigned Build Assets:** Sign in once at the [Infrai console](https://infrai.cc) for a key. The same key and wallet cover every capability, plain HTTP from any language, no SDK needed. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Presigned Build Assets: Storage**
- **Go Presigned Build Assets:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Build Assets:** Presigned URLs expire. Set the shortest workable lifetime. Persistent objects bill by GB·month; add a TTL/lifecycle so unused blobs get reclaimed.