package service

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// Self-contained doubles reuse the shared payAttachmentSettingRepo /
// payAttachmentEncryptor from pay_attachment_service_test.go（同包、无 build tag）。

type fakeTicketAttachmentStore struct {
	uploadedKey         string
	uploadedContentType string
	uploadedBytes       int

	openedKey string
}

func (f *fakeTicketAttachmentStore) Upload(_ context.Context, key, contentType string, data []byte) error {
	f.uploadedKey = key
	f.uploadedContentType = contentType
	f.uploadedBytes = len(data)
	return nil
}

func (f *fakeTicketAttachmentStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	f.openedKey = key
	body := "\x89PNG\r\n\x1a\n stored bytes"
	return io.NopCloser(strings.NewReader(body)), "image/png", int64(len(body)), nil
}

func newTicketAttachmentServiceForTest(t *testing.T) (*TicketAttachmentService, *fakeTicketAttachmentStore) {
	t.Helper()

	repo := newPayAttachmentSettingRepo()
	encryptor := payAttachmentEncryptor{}
	backup := NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, encryptor, nil, nil)

	raw, err := json.Marshal(BackupS3Config{
		Endpoint:        "http://127.0.0.1:9000",
		Region:          "us-east-1",
		Bucket:          "sub2api",
		AccessKeyID:     "ak",
		SecretAccessKey: "enc:sk",
		Prefix:          "backups/",
		ForcePathStyle:  true,
	})
	if err != nil {
		t.Fatalf("marshal backup s3 config: %v", err)
	}
	if err := repo.Set(context.Background(), settingKeyBackupS3Config, string(raw)); err != nil {
		t.Fatalf("seed backup s3 config: %v", err)
	}

	store := &fakeTicketAttachmentStore{}
	settings := NewTicketAttachmentStorageSettingService(repo, encryptor, backup)
	svc := NewTicketAttachmentService(settings, func(context.Context, *BackupS3Config) (TicketAttachmentStore, error) {
		return store, nil
	})
	return svc, store
}

// pngBytes 复用 image_storage_test.go 里的最小 PNG 魔数负载。

// ─── key construction ───

func TestTicketAttachmentPutBuildsServerControlledKey(t *testing.T) {
	svc, store := newTicketAttachmentServiceForTest(t)

	res, err := svc.Put(context.Background(), TicketAttachmentPutInput{
		UserID:              42,
		DeclaredContentType: "image/png",
		Data:                pngBytes,
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// tickets/<uid>/<yyyymm>/<rand8hex>.<ext>
	wantPrefix := DefaultTicketAttachmentStoragePrefix + "42/" + time.Now().UTC().Format("200601") + "/"
	if !strings.HasPrefix(res.Key, wantPrefix) {
		t.Fatalf("key %q is not under %q", res.Key, wantPrefix)
	}
	if !strings.HasSuffix(res.Key, ".png") {
		t.Fatalf("key %q should end with the declared type's extension", res.Key)
	}
	if strings.Contains(res.Key, "..") {
		t.Fatalf("key %q must never contain traversal segments", res.Key)
	}
	if res.Size != len(pngBytes) || res.ContentType != "image/png" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if store.uploadedKey != res.Key || store.uploadedContentType != "image/png" || store.uploadedBytes != len(pngBytes) {
		t.Fatalf("unexpected upload: key=%q type=%q bytes=%d", store.uploadedKey, store.uploadedContentType, store.uploadedBytes)
	}
}

func TestTicketAttachmentPutStaffKeyLandsUnderStaffPrefix(t *testing.T) {
	svc, _ := newTicketAttachmentServiceForTest(t)

	res, err := svc.Put(context.Background(), TicketAttachmentPutInput{
		UserID:              7,
		Staff:               true,
		DeclaredContentType: "image/png",
		Data:                pngBytes,
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !strings.HasPrefix(res.Key, DefaultTicketAttachmentStoragePrefix+"staff/7/") {
		t.Fatalf("staff key %q is not under the staff prefix", res.Key)
	}
}

func TestTicketAttachmentPutKeysAreUnique(t *testing.T) {
	svc, _ := newTicketAttachmentServiceForTest(t)

	seen := make(map[string]struct{})
	for i := 0; i < 20; i++ {
		res, err := svc.Put(context.Background(), TicketAttachmentPutInput{
			UserID: 1, DeclaredContentType: "image/png", Data: pngBytes,
		})
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		if _, dup := seen[res.Key]; dup {
			t.Fatalf("duplicate key generated: %s", res.Key)
		}
		seen[res.Key] = struct{}{}
	}
}

// ─── input validation ───

func TestTicketAttachmentPutRejectsBadInput(t *testing.T) {
	svc, _ := newTicketAttachmentServiceForTest(t)

	cases := []struct {
		name string
		in   TicketAttachmentPutInput
		want error
	}{
		{
			name: "disallowed content type",
			in:   TicketAttachmentPutInput{UserID: 1, DeclaredContentType: "text/html", Data: pngBytes},
			want: ErrTicketAttachmentBadType,
		},
		{
			name: "SVG is not allowed (scriptable)",
			in:   TicketAttachmentPutInput{UserID: 1, DeclaredContentType: "image/svg+xml", Data: pngBytes},
			want: ErrTicketAttachmentBadType,
		},
		{
			name: "empty body",
			in:   TicketAttachmentPutInput{UserID: 1, DeclaredContentType: "image/png", Data: nil},
			want: ErrTicketAttachmentEmpty,
		},
		{
			name: "over size limit",
			in: TicketAttachmentPutInput{
				UserID: 1, DeclaredContentType: "image/png",
				Data: append(append([]byte{}, pngBytes...), make([]byte, MaxTicketAttachmentBytes)...),
			},
			want: ErrTicketAttachmentTooLarge,
		},
		{
			name: "HTML disguised as PNG",
			in: TicketAttachmentPutInput{
				UserID: 1, DeclaredContentType: "image/png",
				Data: []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>"),
			},
			want: ErrTicketAttachmentBadType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Put(context.Background(), tc.in)
			if err != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestTicketAttachmentPutAcceptsAllWhitelistedImageTypes(t *testing.T) {
	svc, _ := newTicketAttachmentServiceForTest(t)

	bodies := map[string][]byte{
		"image/jpeg": []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01"),
		"image/png":  pngBytes,
		"image/gif":  []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00"),
		"image/webp": []byte("RIFF\x1a\x00\x00\x00WEBPVP8 "),
	}
	for contentType, data := range bodies {
		res, err := svc.Put(context.Background(), TicketAttachmentPutInput{
			UserID: 1, DeclaredContentType: contentType, Data: data,
		})
		if err != nil {
			t.Fatalf("Put(%s): %v", contentType, err)
		}
		if !strings.HasSuffix(res.Key, ticketAttachmentAllowedTypes[contentType]) {
			t.Fatalf("Put(%s): key %q has the wrong extension", contentType, res.Key)
		}
	}
}

// ─── 读取的授权边界 ───

func TestTicketAttachmentOpenForUserEnforcesPerUserPrefix(t *testing.T) {
	svc, store := newTicketAttachmentServiceForTest(t)
	ctx := context.Background()

	ownKey := DefaultTicketAttachmentStoragePrefix + "42/202601/a1b2c3d4.png"
	content, err := svc.OpenForUser(ctx, ownKey, 42)
	if err != nil {
		t.Fatalf("OpenForUser(own key): %v", err)
	}
	_ = content.Body.Close()
	if store.openedKey != ownKey {
		t.Fatalf("opened key = %q, want %q", store.openedKey, ownKey)
	}

	// 他人 / 客服 / 前缀外 / 穿越的 key 一律按不存在处理，不泄露存在性。
	badKeys := []string{
		DefaultTicketAttachmentStoragePrefix + "43/202601/a1b2c3d4.png",
		DefaultTicketAttachmentStoragePrefix + "staff/7/202601/a1b2c3d4.png",
		"backups/2026/08/14/dump.sql.gz",
		DefaultTicketAttachmentStoragePrefix + "42/../backups/dump.sql.gz",
		DefaultTicketAttachmentStoragePrefix + "42/202601/../../backups/dump.sql.gz",
		"",
		"   ",
	}
	for _, key := range badKeys {
		if _, err := svc.OpenForUser(ctx, key, 42); err != ErrTicketAttachmentNotFound {
			t.Fatalf("OpenForUser(%q): got %v, want ErrTicketAttachmentNotFound", key, err)
		}
	}
}

func TestTicketAttachmentOpenForStaffEnforcesAttachmentPrefix(t *testing.T) {
	svc, store := newTicketAttachmentServiceForTest(t)
	ctx := context.Background()

	// 客服侧可读前缀下任意 key（用户的与 staff 的）。
	for _, key := range []string{
		DefaultTicketAttachmentStoragePrefix + "42/202601/a1b2c3d4.png",
		DefaultTicketAttachmentStoragePrefix + "staff/7/202601/a1b2c3d4.png",
	} {
		content, err := svc.OpenForStaff(ctx, key)
		if err != nil {
			t.Fatalf("OpenForStaff(%q): %v", key, err)
		}
		_ = content.Body.Close()
		if store.openedKey != key {
			t.Fatalf("opened key = %q, want %q", store.openedKey, key)
		}
	}

	// 没有前缀守卫，客服侧就能读走同一个桶里的数据库备份。
	badKeys := []string{
		"backups/2026/08/14/dump.sql.gz",
		DefaultTicketAttachmentStoragePrefix + "../backups/dump.sql.gz",
		"images/foo.png",
		"",
	}
	for _, key := range badKeys {
		if _, err := svc.OpenForStaff(ctx, key); err != ErrTicketAttachmentInvalidKey {
			t.Fatalf("OpenForStaff(%q): got %v, want ErrTicketAttachmentInvalidKey", key, err)
		}
	}
}

// ─── storage not configured ───

func TestTicketAttachmentFailsClosedWithoutS3Config(t *testing.T) {
	repo := newPayAttachmentSettingRepo()
	backup := NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, payAttachmentEncryptor{}, nil, nil)
	settings := NewTicketAttachmentStorageSettingService(repo, payAttachmentEncryptor{}, backup)
	svc := NewTicketAttachmentService(settings, func(context.Context, *BackupS3Config) (TicketAttachmentStore, error) {
		t.Fatal("factory must not be called when S3 is unconfigured")
		return nil, nil
	})

	_, err := svc.Put(context.Background(), TicketAttachmentPutInput{
		UserID: 1, DeclaredContentType: "image/png", Data: pngBytes,
	})
	if err != ErrTicketAttachmentStorageNotConfigured {
		t.Fatalf("Put: got %v, want ErrTicketAttachmentStorageNotConfigured", err)
	}
	if _, err := svc.OpenForUser(context.Background(), "tickets/1/202601/x.png", 1); err != ErrTicketAttachmentStorageNotConfigured {
		t.Fatalf("OpenForUser: got %v, want ErrTicketAttachmentStorageNotConfigured", err)
	}
	if _, err := svc.OpenForStaff(context.Background(), "tickets/1/202601/x.png"); err != ErrTicketAttachmentStorageNotConfigured {
		t.Fatalf("OpenForStaff: got %v, want ErrTicketAttachmentStorageNotConfigured", err)
	}
}
