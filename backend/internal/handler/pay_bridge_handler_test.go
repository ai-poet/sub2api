package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Self-contained stubs: the shared handler-package stubs live behind
// `//go:build unit`, and this file is untagged so it runs under `make test`.

type payBridgeSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *payBridgeSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (r *payBridgeSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *payBridgeSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *payBridgeSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *payBridgeSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *payBridgeSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *payBridgeSettingRepo) Delete(context.Context, string) error { return nil }

type payBridgeEncryptor struct{}

func (payBridgeEncryptor) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (payBridgeEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type payBridgeMemoryStore struct{ uploadedKey string }

func (s *payBridgeMemoryStore) Upload(_ context.Context, key, _ string, _ []byte) error {
	s.uploadedKey = key
	return nil
}

func (s *payBridgeMemoryStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	body := "%PDF-1.7 stored"
	return io.NopCloser(strings.NewReader(body)), "application/pdf", int64(len(body)), nil
}

func (s *payBridgeMemoryStore) Delete(context.Context, string) error { return nil }

// newPayBridgeRouter mirrors the real registration in RegisterPayRoutes, including
// the body-size middleware, so the 413 mapping is exercised end to end.
func newPayBridgeRouter(t *testing.T) (*gin.Engine, *payBridgeMemoryStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &payBridgeSettingRepo{values: map[string]string{}}
	backup := service.NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, payBridgeEncryptor{}, nil, nil)

	raw, err := json.Marshal(service.BackupS3Config{
		Region: "us-east-1", Bucket: "sub2api", AccessKeyID: "ak", SecretAccessKey: "sk", Prefix: "backups/",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Set(context.Background(), "backup_s3_config", string(raw)))

	store := &payBridgeMemoryStore{}
	// 未保存发票存储设置时默认复用备份凭证，所以这里只 seed 备份配置即可。
	invoiceStorage := service.NewInvoiceStorageSettingService(repo, payBridgeEncryptor{}, backup)
	attachments := service.NewPayAttachmentService(invoiceStorage, func(context.Context, *service.BackupS3Config) (service.PayAttachmentStore, error) {
		return store, nil
	})
	h := NewPayBridgeHandler(attachments, nil)

	r := gin.New()
	group := r.Group("/api/internal/pay/attachments")
	group.Use(middleware.RequestBodyLimit(service.MaxPayAttachmentBytes))
	group.POST("", h.UploadAttachment)
	group.GET("/content", h.DownloadAttachment)
	group.DELETE("", h.DeleteAttachment)
	return r, store
}

func TestPayBridgeUploadRejectsOversizedBodyWith413(t *testing.T) {
	r, _ := newPayBridgeRouter(t)

	body := bytes.Repeat([]byte("A"), service.MaxPayAttachmentBytes+1024)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/pay/attachments?scope=invoice&ref=clx1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/pdf")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "body=%s", w.Body.String())
}

func TestPayBridgeUploadStoresPDFAndReturnsKey(t *testing.T) {
	r, store := newPayBridgeRouter(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/internal/pay/attachments?scope=invoice&ref=clx0123&filename="+url.QueryEscape("发票 2026.pdf"),
		strings.NewReader("%PDF-1.7\ntrailer\n%%EOF\n"),
	)
	req.Header.Set("Content-Type", "application/pdf")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Data service.PayAttachmentPutResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, strings.HasPrefix(resp.Data.Key, service.DefaultInvoiceStoragePrefix))
	require.Equal(t, resp.Data.Key, store.uploadedKey)
	require.Equal(t, "发票 2026.pdf", resp.Data.FileName, "the UTF-8 download name must survive URL decoding")
}

// The internal bridge token is shared with sub2apipay; without this guard a bug
// (or a leaked token) in the pay service could read the database backups that
// live in the same bucket.
func TestPayBridgeDownloadRejectsKeyOutsideAttachmentPrefix(t *testing.T) {
	r, _ := newPayBridgeRouter(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/internal/pay/attachments/content?key="+url.QueryEscape("backups/2026/08/14/dump.sql.gz"),
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	require.NotContains(t, w.Body.String(), "%PDF", "no object bytes may be returned")
}

// 同源流式回传：绕开父页面 CSP 的 frame-src 与 HTTPS→HTTP 混合内容两道拦截。
func TestPayBridgeDownloadStreamsWithUTF8Filename(t *testing.T) {
	r, _ := newPayBridgeRouter(t)

	key := service.DefaultInvoiceStoragePrefix + "2026/08/clx1-abcd1234.pdf"
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/internal/pay/attachments/content?key="+url.QueryEscape(key)+"&filename="+url.QueryEscape("发票 2026.pdf"),
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), "%PDF")
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))

	disposition := w.Header().Get("Content-Disposition")
	require.Contains(t, disposition, "attachment;")
	require.Contains(t, disposition, "filename*=UTF-8''")
	// 发票带着抬头和税号，任何一层都不该缓存。
	require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}
