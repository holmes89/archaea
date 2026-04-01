// Package storage provides an S3-compatible blob storage client backed by
// gocloud.dev. NewClientFromEnv reads credentials from environment variables,
// making it suitable for use in cmd/worker/main.go.
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// BlobClient is an S3-compatible object storage client using gocloud.dev/blob.
type BlobClient struct {
	bucket *blob.Bucket
}

// MinioClient is an alias kept for call-site compatibility.
type MinioClient = BlobClient

// NewMinioClient opens a gocloud.dev/blob bucket pointed at an S3-compatible
// endpoint. useSSL controls whether the endpoint uses https.
func NewMinioClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*BlobClient, error) {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	baseEndpoint := fmt.Sprintf("%s://%s", scheme, endpoint)

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(baseEndpoint)
		o.UsePathStyle = true
	})

	b, err := s3blob.OpenBucketV2(context.Background(), s3Client, bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("open blob bucket: %w", err)
	}

	return &BlobClient{bucket: b}, nil
}

// NewClientFromEnv creates a BlobClient from standard environment variables:
//
//	MINIO_ENDPOINT   — required; if empty, returns nil (uploads skipped)
//	MINIO_ACCESS_KEY
//	MINIO_SECRET_KEY
//	MINIO_BUCKET     — overrides defaultBucket if set
//
// Returns nil when MINIO_ENDPOINT is unset rather than panicking, so callers
// can treat nil as "storage disabled" and skip upload steps.
func NewClientFromEnv(defaultBucket string) *BlobClient {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		log.Println("MINIO_ENDPOINT not set — storage uploads will be skipped")
		return nil
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := defaultBucket
	if b := os.Getenv("MINIO_BUCKET"); b != "" {
		bucket = b
	}
	mc, err := NewMinioClient(endpoint, accessKey, secretKey, bucket, false)
	if err != nil {
		log.Printf("failed to create storage client: %v — uploads will be skipped", err)
		return nil
	}
	log.Printf("storage client configured: endpoint=%s bucket=%s", endpoint, bucket)
	return mc
}

// Upload writes r into the bucket under key with the given content type.
func (c *BlobClient) Upload(ctx context.Context, key string, r io.Reader, _ int64, contentType string) error {
	w, err := c.bucket.NewWriter(ctx, key, &blob.WriterOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("open blob writer for %s: %w", key, err)
	}
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return fmt.Errorf("write blob %s: %w", key, err)
	}
	return w.Close()
}
