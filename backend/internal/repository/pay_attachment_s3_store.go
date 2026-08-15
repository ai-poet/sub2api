package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// S3PayAttachmentStore 用 S3 兼容对象存储实现 service.PayAttachmentStore。
type S3PayAttachmentStore struct {
	client *s3.Client
	bucket string
}

var _ service.PayAttachmentStore = (*S3PayAttachmentStore)(nil)

// NewPayAttachmentStoreFactory 返回一个按 S3 配置构造附件存储的工厂。
func NewPayAttachmentStoreFactory() service.PayAttachmentStoreFactory {
	return func(ctx context.Context, cfg *service.BackupS3Config) (service.PayAttachmentStore, error) {
		client, err := newS3Client(ctx, s3ClientParams{
			Endpoint:        cfg.Endpoint,
			Region:          cfg.Region,
			AccessKeyID:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
			ForcePathStyle:  cfg.ForcePathStyle,
		})
		if err != nil {
			return nil, err
		}
		return &S3PayAttachmentStore{client: client, bucket: cfg.Bucket}, nil
	}
}

func (s *S3PayAttachmentStore) Upload(ctx context.Context, key, contentType string, data []byte) error {
	finish := servertiming.ObserveDependency(ctx, "s3")
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	finish()
	if err != nil {
		return fmt.Errorf("S3 PutObject: %w", err)
	}
	return nil
}

// Open 取回对象内容。
//
// 刻意不用预签名链接：下载页跑在 iframe 里，浏览器直接跳到对象存储会同时撞上两道
// 墙——父页面 CSP 的 frame-src 只允许 'self'，以及 HTTPS 页面里的子框架不允许跳到
// http://。改由服务端取回再同源回传，就与对象存储的域名和协议完全解耦，存储端甚至
// 不需要对公网暴露。
func (s *S3PayAttachmentStore) Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	finish := servertiming.ObserveDependency(ctx, "s3")
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	finish()
	if err != nil {
		return nil, "", 0, fmt.Errorf("S3 GetObject: %w", err)
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}
	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}
	return result.Body, contentType, size, nil
}

func (s *S3PayAttachmentStore) Delete(ctx context.Context, key string) error {
	finish := servertiming.ObserveDependency(ctx, "s3")
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	finish()
	if err != nil {
		return fmt.Errorf("S3 DeleteObject: %w", err)
	}
	return nil
}
