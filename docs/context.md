# Context

Authoritative product-level documents live in the narwhal catalog and are linked here, not copied. This repo's own docs (designs, features, specs) are authoritative in git.

## Product-level documents

- [argus.joelholmes.dev — LLM Context](https://github.com/holmes89/narwhal/blob/main/designs/argus.holmes.haus/CLAUDE.md) — `argus.holmes.haus/CLAUDE.md`

## Docs in this repo

Each doc carries its platform `uuid`; the portal link is the system-of-record view. `body_sha256` in the frontmatter detects drift.

| Doc | Kind | UUID | Portal | Design | File |
|---|---|---|---|---|---|
| Archaea Extension Design | research | `8b2a1ee1-fadc-5422-b693-d67aa5040ff0` | [open](https://joel.holmes.haus/discovery/8b2a1ee1-fadc-5422-b693-d67aa5040ff0) | — | `docs/research/archaea_extensions_design.md` |

## Storage

`storage` wraps `gocloud.dev/blob` as `BlobClient`. `storage.Open(ctx, "s3://bucket?region=...&prefix=dir/")` opens any gocloud bucket URL with the AWS SDK's default credential chain; `prefix` confines the client to a subdirectory of a shared bucket. `BlobClient` can upload, download, stat, delete, check the bucket, and sign browser upload (PUT) and view (GET) URLs; missing keys wrap `storage.ErrNotFound`. `NewMinioClient` and `NewClientFromEnv` (`MINIO_*` variables) remain for services that still use them.
