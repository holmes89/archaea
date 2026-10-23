package storage_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/holmes89/archaea/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// awsEnv gives the SDK fixed fake credentials and keeps it away from the machine's own AWS configuration,
// so the tests sign and send requests without any network or account.
func awsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIDEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret-example-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_CONFIG_FILE", os.DevNull)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", os.DevNull)
}

func open(t *testing.T, rawURL string) *storage.BlobClient {
	t.Helper()
	awsEnv(t)
	c, err := storage.Open(context.Background(), rawURL)
	require.NoError(t, err)
	return c
}

func parse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func hasChecksumParam(u *url.URL) bool {
	for k := range u.Query() {
		if strings.Contains(strings.ToLower(k), "checksum") {
			return true
		}
	}
	return false
}

func TestOpen_RejectsUnknownScheme(t *testing.T) {
	_, err := storage.Open(context.Background(), "nope://bucket")
	assert.Error(t, err)
}

func TestSignedPutURL_PathStyleEndpoint(t *testing.T) {
	c := open(t, "s3://fox-assets?region=eu-west-1&endpoint=http://127.0.0.1:9&use_path_style=true")

	raw, err := c.SignedPutURL(context.Background(), "uploads/abc", 15*time.Minute)

	require.NoError(t, err)
	u := parse(t, raw)
	q := u.Query()
	assert.Equal(t, "http", u.Scheme)
	assert.Equal(t, "127.0.0.1:9", u.Host)
	assert.Equal(t, "/fox-assets/uploads/abc", u.Path)
	assert.Equal(t, "PutObject", q.Get("x-id"))
	assert.Equal(t, "900", q.Get("X-Amz-Expires"))
	assert.Equal(t, "AWS4-HMAC-SHA256", q.Get("X-Amz-Algorithm"))
	assert.Contains(t, q.Get("X-Amz-Credential"), "AKIDEXAMPLE/")
	assert.Contains(t, q.Get("X-Amz-Credential"), "/eu-west-1/s3/aws4_request")
	assert.Equal(t, "host", q.Get("X-Amz-SignedHeaders"), "nothing but the host is signed, so a browser can send any body")
	assert.NotEmpty(t, q.Get("X-Amz-Signature"))
}

func TestSignedGetURL_VirtualHostedAWSAddress(t *testing.T) {
	c := open(t, "s3://fox-assets?region=eu-west-1")

	raw, err := c.SignedGetURL(context.Background(), "blobs/abc", time.Hour)

	require.NoError(t, err)
	u := parse(t, raw)
	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "fox-assets.s3.eu-west-1.amazonaws.com", u.Host)
	assert.Equal(t, "/blobs/abc", u.Path)
	assert.Equal(t, "GetObject", u.Query().Get("x-id"))
	assert.Equal(t, "3600", u.Query().Get("X-Amz-Expires"))
}

func TestSignedPutURL_CarriesNoChecksum(t *testing.T) {
	c := open(t, "s3://fox-assets?region=eu-west-1")

	put, err := c.SignedPutURL(context.Background(), "uploads/abc", time.Minute)

	require.NoError(t, err)
	assert.False(t, hasChecksumParam(parse(t, put)), "a checksum in the query would have to match the browser's body: %s", put)
}

func TestSignedGetURL_OnlyAsksForChecksumHeadersBack(t *testing.T) {
	c := open(t, "s3://fox-assets?region=eu-west-1")

	get, err := c.SignedGetURL(context.Background(), "uploads/abc", time.Minute)

	require.NoError(t, err)
	q := parse(t, get).Query()
	for k := range q {
		if strings.Contains(strings.ToLower(k), "checksum") {
			assert.Equal(t, "X-Amz-Checksum-Mode", k, "the SDK's response-validation flag is the only checksum a GET link may carry")
		}
	}
}

// putHeaders runs one Upload against a stand-in server and returns the headers the SDK sent.
func putHeaders(t *testing.T, rawURLSuffix string) http.Header {
	t.Helper()
	var (
		mu  sync.Mutex
		got http.Header
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		mu.Lock()
		got = r.Header.Clone()
		mu.Unlock()
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := open(t, "s3://b?region=us-east-1&use_path_style=true&disable_https=true&endpoint="+url.QueryEscape(srv.URL)+rawURLSuffix)
	require.NoError(t, c.Upload(context.Background(), "k", strings.NewReader("hello"), 5, "text/plain"))

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, got, "the server saw no request")
	return got
}

func TestOpen_UploadsWithoutChecksumsByDefault(t *testing.T) {
	h := putHeaders(t, "")

	assert.Empty(t, h.Get("X-Amz-Checksum-Crc32"))
	assert.Empty(t, h.Get("X-Amz-Sdk-Checksum-Algorithm"))
	assert.Empty(t, h.Get("X-Amz-Trailer"))
}

func TestOpen_URLCanKeepTheSDKChecksumDefault(t *testing.T) {
	h := putHeaders(t, "&request_checksum_calculation=when_supported")

	assert.NotEmpty(t, h.Get("X-Amz-Sdk-Checksum-Algorithm")+h.Get("X-Amz-Checksum-Crc32")+h.Get("X-Amz-Trailer"),
		"when_supported should add a checksum, which shows the default in Open is what removes it: %v", h)
}

// s3Replying starts a server that answers every request the way S3 answers a bucket listing, with the
// given status and error code (empty code means a good, empty listing), and opens a client on it.
func s3Replying(t *testing.T, status int, code string) *storage.BlobClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		if code == "" {
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>b</Name><IsTruncated>false</IsTruncated></ListBucketResult>`))
			return
		}
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>` + code + `</Code><Message>x</Message></Error>`))
	}))
	t.Cleanup(srv.Close)
	return open(t, "s3://b?region=us-east-1&use_path_style=true&disable_https=true&endpoint="+url.QueryEscape(srv.URL))
}

func TestCheckBucket_S3(t *testing.T) {
	t.Run("a reachable bucket passes", func(t *testing.T) {
		assert.NoError(t, s3Replying(t, http.StatusOK, "").CheckBucket(context.Background()))
	})
	t.Run("a missing bucket is ErrNotFound", func(t *testing.T) {
		err := s3Replying(t, http.StatusNotFound, "NoSuchBucket").CheckBucket(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("a forbidden bucket is an error but not ErrNotFound", func(t *testing.T) {
		err := s3Replying(t, http.StatusForbidden, "AccessDenied").CheckBucket(context.Background())
		require.Error(t, err)
		assert.NotErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestOpen_PrefixConfinesKeysToASubdirectory(t *testing.T) {
	c := open(t, "s3://shared?region=eu-west-1&prefix=fox/")

	put, err := c.SignedPutURL(context.Background(), "uploads/abc", time.Minute)
	require.NoError(t, err)
	get, err := c.SignedGetURL(context.Background(), "blobs/def", time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "/fox/uploads/abc", parse(t, put).Path)
	assert.Equal(t, "/fox/blobs/def", parse(t, get).Path)
}

func TestOpen_PrefixAppliesToRequestsToo(t *testing.T) {
	var (
		mu    sync.Mutex
		paths []string
		lists []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = new(bytes.Buffer).ReadFrom(r.Body)
		mu.Lock()
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			lists = append(lists, r.URL.Query().Get("prefix"))
		} else {
			paths = append(paths, r.Method+" "+r.URL.Path)
		}
		mu.Unlock()
		w.Header().Set("ETag", `"abc"`)
		w.Header().Set("Content-Type", "application/xml")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><IsTruncated>false</IsTruncated></ListBucketResult>`))
		}
	}))
	t.Cleanup(srv.Close)
	c := open(t, "s3://shared?region=us-east-1&use_path_style=true&disable_https=true&prefix=fox/&endpoint="+url.QueryEscape(srv.URL))

	require.NoError(t, c.Upload(context.Background(), "uploads/abc", strings.NewReader("x"), 1, "text/plain"))
	require.NoError(t, c.Delete(context.Background(), "uploads/abc"))
	require.NoError(t, c.CheckBucket(context.Background()))

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, paths, "PUT /shared/fox/uploads/abc")
	assert.Equal(t, []string{"fox/"}, lists, "the bucket check must list only the subdirectory, which is all an IAM policy scoped to it allows")
}
