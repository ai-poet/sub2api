package repository

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// S3PayAttachmentStore 用 S3 兼容对象存储实现 service.PayAttachmentStore。
//
// 与 S3BackupStore 分开实现的唯一原因是下载文件名：备份实现把
// Content-Disposition 硬编码成 filename="<path.Base(key)>"，而发票的对象 key
// 是随机生成的（clxyz-a1b2c3d4.pdf），且用户期望的文件名是中文的。这里按
// RFC 5987/6266 同时给出 ASCII 回退名与 UTF-8 百分号编码名。
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

// PresignDownloadURL 生成带附件下载头的临时链接。downloadName 为展示给用户的
// 文件名（可含中文）；调用方负责保证 key 已通过前缀校验。
func (s *S3PayAttachmentStore) PresignDownloadURL(ctx context.Context, key, downloadName string, expiry time.Duration) (string, error) {
	disposition := attachmentDisposition(downloadName)
	presignClient := s3.NewPresignClient(s.client)
	finish := servertiming.ObserveDependency(ctx, "s3")
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     &s.bucket,
		Key:                        &key,
		ResponseContentDisposition: &disposition,
	}, s3.WithPresignExpires(expiry))
	finish()
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return result.URL, nil
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

// attachmentDisposition 构造 RFC 6266 的 Content-Disposition：
// ASCII 回退名给老客户端，filename* 给支持 RFC 5987 的现代浏览器。
func attachmentDisposition(name string) string {
	name = sanitizeAttachmentName(name)
	if name == "" {
		name = "attachment"
	}
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiFallbackName(name), url.PathEscape(name))
}

// sanitizeAttachmentName 去掉换行与路径分隔符，避免头注入与目录穿越。
func sanitizeAttachmentName(name string) string {
	name = strings.NewReplacer("\r", "", "\n", "", "\x00", "", "/", "_", "\\", "_", `"`, "").Replace(name)
	name = strings.TrimSpace(name)
	if len(name) > 180 {
		name = name[:180]
	}
	return name
}

// asciiFallbackName 把非 ASCII 字符替换成 '_'，用于 filename= 回退项。
func asciiFallbackName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r > 0x7e {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	fallback := strings.TrimSpace(b.String())
	if fallback == "" {
		return "attachment"
	}
	return fallback
}
