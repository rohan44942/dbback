package storage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Adapter struct {
	client *minio.Client
	bucket string
	prefix string
}

type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return strings.TrimSuffix(endpoint, "/")
}

func parseRegionFromEndpoint(endpoint string) string {
	endpoint = normalizeEndpoint(endpoint)
	// s3.ap-southeast-2.amazonaws.com → ap-southeast-2
	if strings.HasPrefix(endpoint, "s3.") && strings.HasSuffix(endpoint, ".amazonaws.com") {
		host := strings.TrimSuffix(strings.TrimPrefix(endpoint, "s3."), ".amazonaws.com")
		if host != "" && host != "amazonaws" {
			return host
		}
	}
	return ""
}

func NewS3Adapter(cfg S3Config) (*S3Adapter, error) {
	endpoint := normalizeEndpoint(cfg.Endpoint)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = parseRegionFromEndpoint(endpoint)
	}
	if endpoint == "" {
		endpoint = "s3.amazonaws.com"
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	}
	if region != "" {
		opts.Region = region
	}

	minioClient, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}
	// ensure bucket exists (caller must ensure permissions)
	ctx := context.Background()
	found, err := minioClient.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !found {
		// try to create (best-effort)
		mbOpts := minio.MakeBucketOptions{}
		if region != "" {
			mbOpts.Region = region
		}
		if err := minioClient.MakeBucket(ctx, cfg.Bucket, mbOpts); err != nil {
			// some providers will prevent creation; ignore error if bucket exists via race
			return nil, err
		}
	}
	return &S3Adapter{client: minioClient, bucket: cfg.Bucket}, nil
}

func (s *S3Adapter) Save(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	// minio requires size if not streaming with PutObjectOptions.Size; we can buffer if size unknown
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}
	return objectName, nil
}

func (s *S3Adapter) Open(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objname, err := ObjectNameFromLocation(objectName)
	if err != nil {
		return nil, err
	}
	if objname == "" {
		return nil, ErrNoObjectName
	}
	obj, err := s.client.GetObject(ctx, s.bucket, objname, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// attempt to stat to validate existence
	_, err = obj.Stat()
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *S3Adapter) Delete(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}

func (s *S3Adapter) PresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return presigned.String(), nil
}

// helper: extract object name from stored location. If stored path is full URL, attempt to parse.
func ObjectNameFromLocation(loc string) (string, error) {
	if strings.HasPrefix(loc, "http") {
		u, err := url.Parse(loc)
		if err != nil {
			return "", err
		}
		return strings.TrimPrefix(u.Path, "/"), nil
	}

	// normalize Windows-style paths
	loc = strings.ReplaceAll(loc, "\\", "/")
	return strings.TrimPrefix(loc, "/"), nil
}

var ErrNoObjectName = errors.New("no object name")
