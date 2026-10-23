package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/holmes89/archaea/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
	"gocloud.dev/gcerrors"
)

func newMem(t *testing.T) *storage.BlobClient {
	t.Helper()
	b := memblob.OpenBucket(nil)
	t.Cleanup(func() { _ = b.Close() })
	return storage.FromBucket(b)
}

func upload(t *testing.T, c *storage.BlobClient, key, contentType string, data []byte) {
	t.Helper()
	require.NoError(t, c.Upload(context.Background(), key, bytes.NewReader(data), int64(len(data)), contentType))
}

func TestStat_ReportsSizeAndContentType(t *testing.T) {
	c := newMem(t)
	upload(t, c, "blobs/a", "image/png", []byte("12345"))

	got, err := c.Stat(context.Background(), "blobs/a")

	require.NoError(t, err)
	assert.Equal(t, int64(5), got.Size)
	assert.Equal(t, "image/png", got.ContentType)
	assert.WithinDuration(t, time.Now(), got.ModTime, time.Minute)
}

func TestStat_MissingKeyIsErrNotFound(t *testing.T) {
	_, err := newMem(t).Stat(context.Background(), "nope")

	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Equal(t, gcerrors.NotFound, gcerrors.Code(err), "the gocloud error stays in the chain")
}

func TestExists(t *testing.T) {
	c := newMem(t)
	upload(t, c, "k", "text/plain", []byte("x"))

	yes, err := c.Exists(context.Background(), "k")
	require.NoError(t, err)
	no, err := c.Exists(context.Background(), "other")
	require.NoError(t, err)

	assert.True(t, yes)
	assert.False(t, no)
}

func TestDelete_RemovesTheObject(t *testing.T) {
	c := newMem(t)
	upload(t, c, "k", "text/plain", []byte("x"))

	require.NoError(t, c.Delete(context.Background(), "k"))

	ok, err := c.Exists(context.Background(), "k")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDelete_MissingKeyIsNotAnError(t *testing.T) {
	assert.NoError(t, newMem(t).Delete(context.Background(), "never-there"))
}

func TestDownload_ReturnsWhatWasUploaded(t *testing.T) {
	c := newMem(t)
	upload(t, c, "k", "text/plain", []byte("hello"))

	r, err := c.Download(context.Background(), "k")
	require.NoError(t, err)
	defer r.Close() //nolint:errcheck
	got, err := io.ReadAll(r)

	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestDownload_MissingKeyIsErrNotFound(t *testing.T) {
	_, err := newMem(t).Download(context.Background(), "nope")

	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Contains(t, err.Error(), "open blob reader for nope", "the message callers saw before is kept")
}

func TestCheckBucket_OpenBucketIsAccessible(t *testing.T) {
	assert.NoError(t, newMem(t).CheckBucket(context.Background()))
}

func TestSignedURL_DriverThatCannotSignReturnsAnError(t *testing.T) {
	c := newMem(t)

	_, putErr := c.SignedPutURL(context.Background(), "k", time.Minute)
	_, getErr := c.SignedGetURL(context.Background(), "k", time.Minute)

	require.Error(t, putErr)
	require.Error(t, getErr)
	assert.Equal(t, gcerrors.Unimplemented, gcerrors.Code(putErr))
}
