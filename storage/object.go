package storage

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

// Attributes describes a stored object.
type Attributes struct {
	Size        int64
	ContentType string
	ModTime     time.Time
}

// Stat returns the attributes of the object at key, or an error wrapping ErrNotFound.
func (c *BlobClient) Stat(ctx context.Context, key string) (Attributes, error) {
	a, err := c.bucket.Attributes(ctx, key)
	if err != nil {
		return Attributes{}, wrapErr(fmt.Sprintf("stat blob %s", key), err)
	}
	return Attributes{Size: a.Size, ContentType: a.ContentType, ModTime: a.ModTime}, nil
}

// Exists reports whether an object is stored at key.
func (c *BlobClient) Exists(ctx context.Context, key string) (bool, error) {
	ok, err := c.bucket.Exists(ctx, key)
	if err != nil {
		return false, wrapErr(fmt.Sprintf("check blob %s", key), err)
	}
	return ok, nil
}

// Delete removes the object at key. A key that is not there is not an error. S3 cannot tell a missing key
// from a missing bucket on this path, so use CheckBucket to find a missing bucket.
func (c *BlobClient) Delete(ctx context.Context, key string) error {
	err := c.bucket.Delete(ctx, key)
	if err == nil || gcerrors.Code(err) == gcerrors.NotFound {
		return nil
	}
	return fmt.Errorf("delete blob %s: %w", key, err)
}

// CheckBucket reports whether the bucket can be listed. A missing bucket wraps ErrNotFound. It lists one
// key, so on S3 it needs s3:ListBucket.
func (c *BlobClient) CheckBucket(ctx context.Context) error {
	ok, err := c.bucket.IsAccessible(ctx)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !ok {
		return fmt.Errorf("check bucket: bucket does not exist: %w", ErrNotFound)
	}
	return nil
}

// SignedPutURL returns a URL that accepts a PUT of the object at key until ttl passes, so a browser can
// upload straight to the bucket. The URL does not limit the size or Content-Type of the body. Not every
// driver can sign (memblob cannot): the error then says so.
func (c *BlobClient) SignedPutURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return c.signed(ctx, key, ttl, http.MethodPut)
}

// SignedGetURL returns a URL that serves the object at key until ttl passes.
func (c *BlobClient) SignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return c.signed(ctx, key, ttl, http.MethodGet)
}

func (c *BlobClient) signed(ctx context.Context, key string, ttl time.Duration, method string) (string, error) {
	u, err := c.bucket.SignedURL(ctx, key, &blob.SignedURLOptions{Expiry: ttl, Method: method})
	if err != nil {
		return "", fmt.Errorf("sign %s url for %s: %w", method, key, err)
	}
	return u, nil
}
