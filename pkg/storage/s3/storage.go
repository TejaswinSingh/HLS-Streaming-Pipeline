package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService interface {
	UploadFile(ctx context.Context, bucket, key string, body io.Reader) error
	DownloadFile(ctx context.Context, bucket, key string) ([]byte, error)
	ListFiles(ctx context.Context, bucket, prefix string) ([]string, error)
}

type S3Service struct {
	client *s3.Client
}

func NewS3Service(ctx context.Context) (StorageService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &S3Service{client: client}, nil
}

// UploadFile uploads data to S3 under specified bucket/key
func (s *S3Service) UploadFile(ctx context.Context, bucket, key string, body io.Reader) error {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, body)
	if err != nil {
		return fmt.Errorf("error reading upload body: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
		// ACL:    types.ObjectCannedACLPrivate, // Optional: or public-read
	})
	if err != nil {
		return fmt.Errorf("error uploading to S3: %w", err)
	}

	return nil
}

// DownloadFile fetches an object from s3 and returns it as []byte
func (s *S3Service) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("error downloading from S3: %w", err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading S3 object body: %w", err)
	}

	return data, nil
}

func (s *S3Service) ListFiles(ctx context.Context, bucket, prefix string) ([]string, error) {
	var keys []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing s3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}
