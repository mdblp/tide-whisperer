package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Uploader struct {
	uploader   *manager.Uploader
	bucketPath string
}

func NewS3Uploader(s3UploadClient manager.UploadAPIClient, bucketPath string) (S3Uploader, error) {
	if s3UploadClient == nil {
		return S3Uploader{}, errors.New("s3 upload client nil")
	}
	if bucketPath == "" {
		return S3Uploader{}, errors.New("bucket path is empty")
	}
	return S3Uploader{
		uploader:   manager.NewUploader(s3UploadClient),
		bucketPath: bucketPath,
	}, nil
}

func (u S3Uploader) Upload(ctx context.Context, filename string, buffer *bytes.Buffer) error {
	_, err := u.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucketPath),
		Key:    aws.String(filename),
		Body:   buffer,
	})
	if err != nil {
		return fmt.Errorf("upload failed filename=[%s], bucketPath=[%s]: %w", filename, u.bucketPath, err)
	}
	return nil
}
