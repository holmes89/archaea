package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"
)

// ErrNotFound is wrapped by the errors BlobClient returns for a key (or bucket) that does not exist, so a
// caller can test errors.Is(err, storage.ErrNotFound) without importing gocloud.dev. The gocloud error stays
// in the chain.
var ErrNotFound = errors.New("storage: not found")

// checksumParam is the s3blob URL parameter that sets when the AWS SDK adds request checksums.
const checksumParam = "request_checksum_calculation"

// Open opens the bucket named by a gocloud.dev blob URL, for example
//
//	s3://my-bucket?region=eu-west-1
//	s3://my-bucket?region=us-east-1&endpoint=http://localhost:7070&use_path_style=true
//	s3://shared-bucket?region=eu-west-1&prefix=fox/
//
// The second form reaches an S3-compatible server. The third confines the client to a subdirectory of a
// shared bucket: every key, signed URL and listing gets the prefix, which should end in "/".
//
// Credentials come from the AWS SDK's default chain (environment, shared config, instance or task role), so
// nothing secret appears in the URL.
//
// For s3:// URLs that do not say otherwise, request_checksum_calculation is set to when_required. The SDK's
// default adds an X-Amz-Checksum-Crc32 header to every upload, which AWS accepts but gocloud.dev warns some
// S3-compatible providers do not. It makes no difference to the signed URLs, which carry no checksum.
// Pass request_checksum_calculation=when_supported to keep the SDK default.
func Open(ctx context.Context, rawURL string) (*BlobClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse bucket url: %w", err)
	}
	if u.Scheme == s3blob.Scheme {
		q := u.Query()
		if q.Get(checksumParam) == "" {
			q.Set(checksumParam, "when_required")
			u.RawQuery = q.Encode()
		}
	}
	b, err := blob.OpenBucket(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("open blob bucket: %w", err)
	}
	return FromBucket(b), nil
}

// FromBucket wraps a bucket that is already open, for a driver Open does not reach or for tests (for
// example gocloud.dev/blob/memblob).
func FromBucket(b *blob.Bucket) *BlobClient {
	return &BlobClient{bucket: b}
}

// wrapErr adds context to err and, when gocloud reports the key or bucket as missing, ErrNotFound.
func wrapErr(msg string, err error) error {
	if gcerrors.Code(err) == gcerrors.NotFound {
		return fmt.Errorf("%s: %w: %w", msg, ErrNotFound, err)
	}
	return fmt.Errorf("%s: %w", msg, err)
}
