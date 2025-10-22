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
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func NewS3Adapter(cfg S3Config) (*S3Adapter, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
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
		if err := minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
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
	info, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}
	return info.Location, nil
}

func (s *S3Adapter) Open(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
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
func objectNameFromLocation(loc string) (string, error) {
	// naive: if contains bucket then find last slash
	if strings.HasPrefix(loc, "http") {
		u, err := url.Parse(loc)
		if err != nil {
			return "", err
		}
		return u.Path[1:], nil // drop leading slash
	}
	// otherwise assume it's object name
	return loc, nil
}

var ErrNoObjectName = errors.New("no object name")
