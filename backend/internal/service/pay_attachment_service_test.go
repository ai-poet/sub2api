package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// ─── test doubles ───
//
// These are deliberately self-contained rather than reusing the shared
// stubSettingRepo/reversibleEncryptor helpers: those live in files guarded by
// `//go:build unit`, and this file has no build tag so it runs under the plain
// `make test` (`go test ./...`).

type payAttachmentSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newPayAttachmentSettingRepo() *payAttachmentSettingRepo {
	return &payAttachmentSettingRepo{values: map[string]string{}}
}

func (r *payAttachmentSettingRepo) Get(context.Context, string) (*Setting, error) { return nil, nil }

func (r *payAttachmentSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *payAttachmentSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *payAttachmentSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *payAttachmentSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *payAttachmentSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *payAttachmentSettingRepo) Delete(context.Context, string) error { return nil }

// payAttachmentEncryptor mirrors the real AES encryptor closely enough that
// decrypting an unencrypted value fails, like it does in production.
type payAttachmentEncryptor struct{}

func (payAttachmentEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (payAttachmentEncryptor) Decrypt(ciphertext string) (string, error) {
	rest, ok := strings.CutPrefix(ciphertext, "enc:")
	if !ok {
		return "", errors.New("not encrypted")
	}
	return rest, nil
}

type fakePayAttachmentStore struct {
	uploadedKey         string
	uploadedContentType string
	uploadedBytes       int

	presignedKey  string
	presignedName string
	presignedTTL  time.Duration

	deletedKey string
}

func (f *fakePayAttachmentStore) Upload(_ context.Context, key, contentType string, data []byte) error {
	f.uploadedKey = key
	f.uploadedContentType = contentType
	f.uploadedBytes = len(data)
	return nil
}

func (f *fakePayAttachmentStore) PresignDownloadURL(_ context.Context, key, downloadName string, expiry time.Duration) (string, error) {
	f.presignedKey = key
	f.presignedName = downloadName
	f.presignedTTL = expiry
	return "https://example.invalid/" + key, nil
}

func (f *fakePayAttachmentStore) Delete(_ context.Context, key string) error {
	f.deletedKey = key
	return nil
}

func newPayAttachmentServiceForTest(t *testing.T) (*PayAttachmentService, *fakePayAttachmentStore) {
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

	store := &fakePayAttachmentStore{}
	svc := NewPayAttachmentService(backup, func(context.Context, *BackupS3Config) (PayAttachmentStore, error) {
		return store, nil
	})
	return svc, store
}

// minimal PDF payload: http.DetectContentType keys off the %PDF- magic bytes.
func pdfBytes() []byte {
	return []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\ntrailer\n%%EOF\n")
}

// ─── key construction ───

func TestPayAttachmentPutBuildsServerControlledKey(t *testing.T) {
	svc, store := newPayAttachmentServiceForTest(t)

	res, err := svc.Put(context.Background(), PayAttachmentPutInput{
		Scope:               "invoice",
		Ref:                 "clx0123456789",
		FileName:            "../../etc/passwd 发票.pdf",
		DeclaredContentType: "application/pdf",
		Data:                pdfBytes(),
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if !strings.HasPrefix(res.Key, PayAttachmentPrefix+"invoice/") {
		t.Fatalf("key %q is not under the invoice prefix", res.Key)
	}
	// The client filename must never leak into the object key.
	if strings.Contains(res.Key, "passwd") || strings.Contains(res.Key, "..") || strings.Contains(res.Key, "发票") {
		t.Fatalf("key %q leaked client-supplied filename", res.Key)
	}
	if !strings.HasSuffix(res.Key, ".pdf") {
		t.Fatalf("key %q should end with the declared type's extension", res.Key)
	}
	if !strings.Contains(res.Key, "clx0123456789-") {
		t.Fatalf("key %q should embed the ref id", res.Key)
	}
	// ...but it is preserved (sanitised) as the download name.
	if strings.ContainsAny(res.FileName, `/\`) {
		t.Fatalf("file name %q still contains path separators", res.FileName)
	}
	if store.uploadedKey != res.Key || store.uploadedContentType != "application/pdf" {
		t.Fatalf("unexpected upload: key=%q type=%q", store.uploadedKey, store.uploadedContentType)
	}
}

func TestPayAttachmentPutKeysAreUnique(t *testing.T) {
	svc, _ := newPayAttachmentServiceForTest(t)

	seen := make(map[string]struct{})
	for i := 0; i < 20; i++ {
		res, err := svc.Put(context.Background(), PayAttachmentPutInput{
			Scope: "invoice", Ref: "clxsame", DeclaredContentType: "application/pdf", Data: pdfBytes(),
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

func TestPayAttachmentPutRejectsBadInput(t *testing.T) {
	svc, _ := newPayAttachmentServiceForTest(t)

	cases := []struct {
		name string
		in   PayAttachmentPutInput
		want error
	}{
		{
			name: "unknown scope",
			in:   PayAttachmentPutInput{Scope: "backup", Ref: "clx1", DeclaredContentType: "application/pdf", Data: pdfBytes()},
			want: ErrPayAttachmentInvalidScope,
		},
		{
			name: "ref with slash would escape the prefix",
			in:   PayAttachmentPutInput{Scope: "invoice", Ref: "../../backups/x", DeclaredContentType: "application/pdf", Data: pdfBytes()},
			want: ErrPayAttachmentInvalidRef,
		},
		{
			name: "empty ref",
			in:   PayAttachmentPutInput{Scope: "invoice", Ref: "", DeclaredContentType: "application/pdf", Data: pdfBytes()},
			want: ErrPayAttachmentInvalidRef,
		},
		{
			name: "disallowed content type",
			in:   PayAttachmentPutInput{Scope: "invoice", Ref: "clx1", DeclaredContentType: "text/html", Data: pdfBytes()},
			want: ErrPayAttachmentInvalidType,
		},
		{
			name: "empty body",
			in:   PayAttachmentPutInput{Scope: "invoice", Ref: "clx1", DeclaredContentType: "application/pdf", Data: nil},
			want: ErrPayAttachmentEmpty,
		},
		{
			name: "over size limit",
			in: PayAttachmentPutInput{
				Scope: "invoice", Ref: "clx1", DeclaredContentType: "application/pdf",
				Data: append(pdfBytes(), make([]byte, MaxPayAttachmentBytes)...),
			},
			want: ErrPayAttachmentTooLarge,
		},
		{
			name: "HTML disguised as PDF",
			in: PayAttachmentPutInput{
				Scope: "invoice", Ref: "clx1", DeclaredContentType: "application/pdf",
				Data: []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>"),
			},
			want: ErrPayAttachmentContentMismatch,
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

func TestPayAttachmentPutAcceptsContentTypeWithCharset(t *testing.T) {
	svc, _ := newPayAttachmentServiceForTest(t)

	if _, err := svc.Put(context.Background(), PayAttachmentPutInput{
		Scope: "invoice", Ref: "clx1", DeclaredContentType: "Application/PDF; charset=binary", Data: pdfBytes(),
	}); err != nil {
		t.Fatalf("Put with parameterised content type: %v", err)
	}
}

// ─── presign / delete authorization boundary ───

func TestPayAttachmentPresignRejectsKeysOutsidePrefix(t *testing.T) {
	svc, _ := newPayAttachmentServiceForTest(t)

	// Without the prefix guard, the internal bridge token would be enough to
	// read the whole database backup out of the shared bucket.
	badKeys := []string{
		"backups/2026/08/14/dump.sql.gz",
		"pay-attachments/../backups/dump.sql.gz",
		"pay-attachments/invoice/../../backups/dump.sql.gz",
		"",
		"   ",
		"images/foo.png",
	}
	for _, key := range badKeys {
		t.Run(key, func(t *testing.T) {
			if _, _, err := svc.Presign(context.Background(), key, "x.pdf", time.Minute); err != ErrPayAttachmentInvalidKey {
				t.Fatalf("Presign(%q): got %v, want ErrPayAttachmentInvalidKey", key, err)
			}
			if err := svc.Delete(context.Background(), key); err != ErrPayAttachmentInvalidKey {
				t.Fatalf("Delete(%q): got %v, want ErrPayAttachmentInvalidKey", key, err)
			}
		})
	}
}

func TestPayAttachmentPresignClampsTTL(t *testing.T) {
	svc, store := newPayAttachmentServiceForTest(t)
	key := PayAttachmentPrefix + "invoice/2026/08/clx1-a1b2c3d4.pdf"

	cases := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{"zero falls back to default", 0, defaultPayAttachmentPresignTTL},
		{"negative falls back to default", -time.Hour, defaultPayAttachmentPresignTTL},
		{"below floor is raised", time.Second, minPayAttachmentPresignTTL},
		{"above ceiling is capped", time.Hour, maxPayAttachmentPresignTTL},
		{"in range is preserved", 3 * time.Minute, 3 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := time.Now()
			_, expiresAt, err := svc.Presign(context.Background(), key, "发票.pdf", tc.ttl)
			if err != nil {
				t.Fatalf("Presign: %v", err)
			}
			if store.presignedTTL != tc.want {
				t.Fatalf("ttl = %v, want %v", store.presignedTTL, tc.want)
			}
			if expiresAt.Before(before.Add(tc.want - time.Second)) {
				t.Fatalf("expiresAt %v is earlier than the clamped ttl %v", expiresAt, tc.want)
			}
			if store.presignedName != "发票.pdf" {
				t.Fatalf("download name = %q, want the UTF-8 name to survive", store.presignedName)
			}
		})
	}
}

// ─── storage not configured ───

func TestPayAttachmentFailsClosedWithoutS3Config(t *testing.T) {
	repo := newPayAttachmentSettingRepo()
	backup := NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, payAttachmentEncryptor{}, nil, nil)
	svc := NewPayAttachmentService(backup, func(context.Context, *BackupS3Config) (PayAttachmentStore, error) {
		t.Fatal("factory must not be called when S3 is unconfigured")
		return nil, nil
	})

	_, err := svc.Put(context.Background(), PayAttachmentPutInput{
		Scope: "invoice", Ref: "clx1", DeclaredContentType: "application/pdf", Data: pdfBytes(),
	})
	if err != ErrPayAttachmentStorageNotConfigured {
		t.Fatalf("got %v, want ErrPayAttachmentStorageNotConfigured", err)
	}
}
