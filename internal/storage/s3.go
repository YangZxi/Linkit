package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	cfgpkg "linkit/internal/config"
)

type S3Storage struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
	logger    *slog.Logger
}

func NewS3(cfg cfgpkg.Config, logger *slog.Logger) (*S3Storage, error) {
	if cfg.AppConfig.S3Bucket == "" || cfg.AppConfig.S3AccessKey == "" || cfg.AppConfig.S3SecretKey == "" || cfg.AppConfig.S3Endpoint == "" {
		return nil, fmt.Errorf("缺少 S3 配置")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: cfg.AppConfig.S3Endpoint, HostnameImmutable: true}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AppConfig.S3Region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: cfg.AppConfig.S3AccessKey, SecretAccessKey: cfg.AppConfig.S3SecretKey}, nil
		})),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Storage{bucket: cfg.AppConfig.S3Bucket, client: client, presigner: s3.NewPresignClient(client), logger: logger}, nil
}

func (s *S3Storage) Platform() BucketPlatform {
	return PlatformS3
}

func (s *S3Storage) Write(objectKey string, r io.Reader, size int64, contentType string) (string, error) {
	normalized, err := NormalizeObjectKey(objectKey)
	if err != nil {
		return "", err
	}

	// 若 size <0 需要缓存到内存以便 ContentLength
	var body io.Reader = r
	var buf *bytes.Buffer
	if size < 0 {
		buf = new(bytes.Buffer)
		if _, err := io.Copy(buf, r); err != nil {
			return "", err
		}
		size = int64(buf.Len())
		body = buf
	}

	contentLength := aws.Int64(size)

	_, err = s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &normalized,
		Body:          body,
		ContentLength: contentLength,
		ContentType:   &contentType,
		ACL:           types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return "", err
	}
	return BuildStoredPath(PlatformS3, s.bucket, normalized)
}

func (s *S3Storage) GetURL(storedPath string, expires time.Duration) (string, error) {
	platform, bucket, key, err := ParseStoredPath(storedPath)
	if err != nil {
		return "", err
	}
	if platform != PlatformS3 {
		return "", fmt.Errorf("存储路径与 S3 不匹配")
	}
	exp := expires
	if exp <= 0 {
		exp = 30 * time.Minute
	}
	presigned, err := s.presigner.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = exp
	})
	if err != nil {
		return "", err
	}
	return presigned.URL, nil
}

func (s *S3Storage) Delete(storedPath string) error {
	platform, bucket, key, err := ParseStoredPath(storedPath)
	if err != nil {
		return err
	}
	if platform != PlatformS3 {
		return fmt.Errorf("存储路径与 S3 不匹配")
	}
	if bucket == "" {
		bucket = s.bucket
	}
	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	return err
}
