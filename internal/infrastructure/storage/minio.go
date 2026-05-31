package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type MinioClient struct {
	Client *minio.Client
	Bucket string
	Log    *logrus.Logger
}

func NewMinioClient(endpoint, accessKey, secretKey, bucket string, useSSL bool, log *logrus.Logger) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	mc := &MinioClient{Client: client, Bucket: bucket, Log: log}

	if err := mc.EnsureBucket(context.Background()); err != nil {
		return nil, err
	}

	return mc, nil
}

func (m *MinioClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.Client.BucketExists(ctx, m.Bucket)
	if err != nil {
		return fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := m.Client.MakeBucket(ctx, m.Bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("minio make bucket: %w", err)
		}
		m.Log.WithField("bucket", m.Bucket).Info("minio bucket created")
	}
	return nil
}

func (m *MinioClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("minio upload: open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("minio upload: stat file: %w", err)
	}

	_, err = m.Client.PutObject(ctx, m.Bucket, objectName, file, stat.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio upload: put object %q: %w", objectName, err)
	}

	m.Log.WithFields(logrus.Fields{
		"object": objectName,
		"size":   stat.Size(),
	}).Info("file uploaded to minio")

	return nil
}
