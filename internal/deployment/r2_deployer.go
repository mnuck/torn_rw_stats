package deployment

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"time"

	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// R2UploadTimeout bounds a single object upload to R2
	R2UploadTimeout = 30 * time.Second

	// r2CacheControl matches the headers nginx sent for the live JSON, so
	// browsers and Cloudflare's edge always fetch the latest copy
	r2CacheControl = "no-cache, no-store, must-revalidate"
)

// R2Deployer handles deployment to a Cloudflare R2 bucket via its
// S3-compatible API, writing each file as an object keyed by filename.
type R2Deployer struct {
	client *s3.Client
	bucket string
}

// NewR2Deployer creates a new R2 deployer for the given account and bucket
func NewR2Deployer(accountID, accessKeyID, secretAccessKey, bucket string) *R2Deployer {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	return newR2Deployer(endpoint, accessKeyID, secretAccessKey, bucket)
}

// newR2Deployer creates an R2 deployer against an explicit endpoint
func newR2Deployer(endpoint, accessKeyID, secretAccessKey, bucket string) *R2Deployer {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle: true,
		// Only send checksums when an operation requires them; R2 does not
		// support every checksum the SDK would otherwise add by default
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})

	return &R2Deployer{
		client: client,
		bucket: bucket,
	}
}

// DeployData uploads data from an io.Reader as an R2 object named filename.
func (d *R2Deployer) DeployData(data io.Reader, size int64, filename string) error {
	// The SDK needs a seekable body to sign the request; buffer anything else
	body, ok := data.(io.ReadSeeker)
	if !ok {
		buf, err := io.ReadAll(data)
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(context.Background(), R2UploadTimeout)
	defer cancel()

	_, err := d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(d.bucket),
		Key:           aws.String(filename),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		CacheControl:  aws.String(r2CacheControl),
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to R2 bucket %s: %w", filename, d.bucket, err)
	}

	slog.Info("Successfully deployed data to R2",
		"bucket", d.bucket,
		"key", filename,
		"size", size)

	return nil
}
