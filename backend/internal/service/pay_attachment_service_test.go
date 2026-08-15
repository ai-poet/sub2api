package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

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

	openedKey string

	deletedKey string
}

func (f *fakePayAttachmentStore) Upload(_ context.Context, key, contentType string, data []byte) error {
	f.uploadedKey = key
	f.uploadedContentType = contentType
	f.uploadedBytes = len(data)
	return nil
}

func (f *fakePayAttachmentStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	f.openedKey = key
	body := "%PDF-1.7 stored bytes"
	return io.NopCloser(strings.NewReader(body)), "application/pdf", int64(len(body)), nil
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
	settings := NewInvoiceStorageSettingService(repo, encryptor, backup)
	svc := NewPayAttachmentService(settings, func(context.Context, *BackupS3Config) (PayAttachmentStore, error) {
		return store, nil
	})
	return svc, store
}

// testInvoicePrefix 是未保存过发票存储设置时生效的默认前缀。
const testInvoicePrefix = DefaultInvoiceStoragePrefix

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

	if !strings.HasPrefix(res.Key, testInvoicePrefix) {
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

// ─── 读取 / 删除的授权边界 ───

func TestPayAttachmentOpenRejectsKeysOutsidePrefix(t *testing.T) {
	svc, _ := newPayAttachmentServiceForTest(t)

	// Without the prefix guard, the internal bridge token would be enough to
	// read the whole database backup out of the shared bucket.
	badKeys := []string{
		"backups/2026/08/14/dump.sql.gz",
		testInvoicePrefix + "../backups/dump.sql.gz",
		testInvoicePrefix + "2026/08/../../backups/dump.sql.gz",
		"",
		"   ",
		"images/foo.png",
	}
	for _, key := range badKeys {
		t.Run(key, func(t *testing.T) {
			if _, err := svc.Open(context.Background(), key); err != ErrPayAttachmentInvalidKey {
				t.Fatalf("Open(%q): got %v, want ErrPayAttachmentInvalidKey", key, err)
			}
			if err := svc.Delete(context.Background(), key); err != ErrPayAttachmentInvalidKey {
				t.Fatalf("Delete(%q): got %v, want ErrPayAttachmentInvalidKey", key, err)
			}
		})
	}
}

func TestPayAttachmentOpenReturnsStoredBytes(t *testing.T) {
	svc, store := newPayAttachmentServiceForTest(t)
	key := testInvoicePrefix + "2026/08/clx1-a1b2c3d4.pdf"

	content, err := svc.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = content.Body.Close() }()

	if store.openedKey != key {
		t.Fatalf("opened key = %q, want %q", store.openedKey, key)
	}
	data, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(data) == 0 || content.ContentType != "application/pdf" {
		t.Fatalf("unexpected content: type=%q len=%d", content.ContentType, len(data))
	}
	if content.Size != int64(len(data)) {
		t.Fatalf("size = %d, want %d", content.Size, len(data))
	}
}

// 中文文件名要按 RFC 6266 同时给出 ASCII 回退名和 UTF-8 编码名，
// 否则浏览器存下来会是乱码或随机的对象 key。
func TestAttachmentContentDispositionEncodesUTF8Name(t *testing.T) {
	got := AttachmentContentDisposition("发票 2026.pdf")
	if !strings.Contains(got, "filename*=UTF-8''") {
		t.Fatalf("missing RFC 5987 filename*: %s", got)
	}
	if !strings.Contains(got, "%E5%8F%91%E7%A5%A8") {
		t.Fatalf("UTF-8 name not percent-encoded: %s", got)
	}
	if !strings.HasPrefix(got, "attachment; filename=") {
		t.Fatalf("missing ascii fallback: %s", got)
	}
}

// 头注入防护：文件名里的 CRLF 与引号必须被剥掉，否则能往响应头里塞任意字段。
func TestAttachmentContentDispositionStripsHeaderInjection(t *testing.T) {
	got := AttachmentContentDisposition("a\r\nSet-Cookie: x=1\"; drop.pdf")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("disposition still contains CRLF: %q", got)
	}
	if strings.Contains(got, `x=1"`) {
		t.Fatalf("disposition still contains a raw quote: %q", got)
	}
}

// ─── storage not configured ───

func TestPayAttachmentFailsClosedWithoutS3Config(t *testing.T) {
	repo := newPayAttachmentSettingRepo()
	backup := NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, payAttachmentEncryptor{}, nil, nil)
	settings := NewInvoiceStorageSettingService(repo, payAttachmentEncryptor{}, backup)
	svc := NewPayAttachmentService(settings, func(context.Context, *BackupS3Config) (PayAttachmentStore, error) {
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
