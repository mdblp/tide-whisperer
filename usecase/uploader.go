package usecase

import (
	"bytes"
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const bucketPrefix = "com.diabeloop.yourloops.export."

type Uploader struct {
	uploader     *manager.Uploader
	bucketSuffix string
}

func NewUploader(s3UploadClient manager.UploadAPIClient, bucketSuffix string) (Uploader, error) {
	if s3UploadClient == nil {
		return Uploader{}, errors.New("invalid s3 upload client")
	}
	if bucketSuffix == "" {
		return Uploader{}, errors.New("invalid bucket suffix")
	}
	return Uploader{
		uploader:     manager.NewUploader(s3UploadClient),
		bucketSuffix: bucketSuffix,
	}, nil
}

func (u Uploader) Upload(ctx context.Context, filename string, buffer *bytes.Buffer) error {
	_, err := u.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketPrefix + u.bucketSuffix),
		Key:    aws.String(filename),
		Body:   buffer,
	})
	if err != nil {
		return err
	}
	return nil
}
