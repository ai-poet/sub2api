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

// S3TicketAttachmentStore 用 S3 兼容对象存储实现 service.TicketAttachmentStore。
type S3TicketAttachmentStore struct {
	client *s3.Client
	bucket string
}

var _ service.TicketAttachmentStore = (*S3TicketAttachmentStore)(nil)

// NewTicketAttachmentStoreFactory 返回一个按 S3 配置构造工单附件存储的工厂。
func NewTicketAttachmentStoreFactory() service.TicketAttachmentStoreFactory {
	return func(ctx context.Context, cfg *service.BackupS3Config) (service.TicketAttachmentStore, error) {
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
		return &S3TicketAttachmentStore{client: client, bucket: cfg.Bucket}, nil
	}
}

func (s *S3TicketAttachmentStore) Upload(ctx context.Context, key, contentType string, data []byte) error {
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
// 刻意不用预签名链接，与发票附件同源回传的理由一致：工单图片直接内嵌在对话页里，
// 让浏览器跳到对象存储会撞上 CSP 的 img-src 与 HTTPS→HTTP 混合内容两堵墙；服务端
// 取回再同源回传后，对象存储不必对公网暴露。
func (s *S3TicketAttachmentStore) Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
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
