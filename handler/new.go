package gsftp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/pkg/sftp"
	"google.golang.org/api/option"
)

func GoogleCloudStorageHandler(ctx context.Context, bucketName string, opts ...option.ClientOption) (*sftp.Handlers, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("Storage Client Error: %s", err)
	}

	bucket := client.Bucket(bucketName)

	handler := &gcsHandler{
		client: client,
		bucket: bucket,
	}

	return &sftp.Handlers{handler, handler, handler, handler}, nil
}
