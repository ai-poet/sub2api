//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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

// ---------- 测试骨架（本包无共享 stub，自带最小实现） ----------

type ticketAttachmentSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *ticketAttachmentSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (r *ticketAttachmentSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *ticketAttachmentSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *ticketAttachmentSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *ticketAttachmentSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *ticketAttachmentSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *ticketAttachmentSettingRepo) Delete(context.Context, string) error { return nil }

type ticketAttachmentEncryptor struct{}

func (ticketAttachmentEncryptor) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (ticketAttachmentEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type ticketAttachmentMemStore struct {
	uploadedKey string
	openedKey   string
}

func (s *ticketAttachmentMemStore) Upload(_ context.Context, key, _ string, _ []byte) error {
	s.uploadedKey = key
	return nil
}

func (s *ticketAttachmentMemStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	s.openedKey = key
	body := "\x89PNG\r\n\x1a\n stored"
	return io.NopCloser(strings.NewReader(body)), "image/png", int64(len(body)), nil
}

// newTicketAttachmentAdminRouter 以 operator 身份挂客服侧附件路由（operator 与 admin 同权）。
func newTicketAttachmentAdminRouter(t *testing.T, userID int64, role string) (*gin.Engine, *ticketAttachmentMemStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &ticketAttachmentSettingRepo{values: map[string]string{}}
	backup := service.NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, ticketAttachmentEncryptor{}, nil, nil)
	raw, err := json.Marshal(service.BackupS3Config{
		Region: "us-east-1", Bucket: "sub2api", AccessKeyID: "ak", SecretAccessKey: "sk", Prefix: "backups/",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Set(context.Background(), "backup_s3_config", string(raw)))

	store := &ticketAttachmentMemStore{}
	settings := service.NewTicketAttachmentStorageSettingService(repo, ticketAttachmentEncryptor{}, backup)
	// 客服侧只走 OpenForStaff（整个前缀可读），用不到工单仓储。
	svc := service.NewTicketAttachmentService(settings, func(context.Context, *service.BackupS3Config) (service.TicketAttachmentStore, error) {
		return store, nil
	}, nil)
	h := NewTicketAttachmentHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), role)
		c.Next()
	})
	router.POST("/admin/tickets/attachments",
		middleware.RequestBodyLimit(service.MaxTicketAttachmentBytes+(1<<20)), h.Upload)
	router.GET("/admin/tickets/attachments/content", h.Content)
	return router, store
}

func staffAttachmentUploadRequest(t *testing.T, target, contentType string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="shot.png"`},
		"Content-Type":        {contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, target, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func adminTestPNG() []byte {
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")
}

// ---------- 上传 ----------

func TestAdminTicketAttachmentUploadLandsUnderStaffPrefix(t *testing.T) {
	router, store := newTicketAttachmentAdminRouter(t, 7, service.RoleOperator)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, staffAttachmentUploadRequest(t, "/admin/tickets/attachments", "image/png", adminTestPNG()))

	require.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Data service.TicketAttachmentPutResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, strings.HasPrefix(resp.Data.Key, service.DefaultTicketAttachmentStoragePrefix+"staff/7/"),
		"staff key %q must land under tickets/staff/<uid>/", resp.Data.Key)
	require.Equal(t, resp.Data.Key, store.uploadedKey)
}

func TestAdminTicketAttachmentUploadRejectsBadTypeAndOversize(t *testing.T) {
	router, _ := newTicketAttachmentAdminRouter(t, 7, service.RoleAdmin)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, staffAttachmentUploadRequest(t, "/admin/tickets/attachments", "text/html", adminTestPNG()))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "TICKET_ATTACHMENT_BAD_TYPE")

	data := append(adminTestPNG(), bytes.Repeat([]byte("A"), service.MaxTicketAttachmentBytes)...)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, staffAttachmentUploadRequest(t, "/admin/tickets/attachments", "image/png", data))
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), "TICKET_ATTACHMENT_TOO_LARGE")
}

// ---------- 同源回传：operator 可读附件前缀下任意 key ----------

func TestAdminTicketAttachmentContentAllowsAnyKeyWithinPrefix(t *testing.T) {
	router, store := newTicketAttachmentAdminRouter(t, 7, service.RoleOperator)

	for _, key := range []string{
		service.DefaultTicketAttachmentStoragePrefix + "42/202601/a1b2c3d4.png",
		service.DefaultTicketAttachmentStoragePrefix + "staff/7/202601/a1b2c3d4.png",
	} {
		req := httptest.NewRequest(http.MethodGet, "/admin/tickets/attachments/content?key="+url.QueryEscape(key), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "key=%q body=%s", key, w.Body.String())
		require.Equal(t, key, store.openedKey)
		require.Equal(t, "image/png", w.Header().Get("Content-Type"))
		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
	}
}

func TestAdminTicketAttachmentContentRejectsTraversalAndForeignPrefix(t *testing.T) {
	router, store := newTicketAttachmentAdminRouter(t, 7, service.RoleOperator)

	for _, key := range []string{
		"backups/2026/08/14/dump.sql.gz",
		service.DefaultTicketAttachmentStoragePrefix + "../backups/dump.sql.gz",
		service.DefaultTicketAttachmentStoragePrefix + "42/202601/../../backups/dump.sql.gz",
		"",
	} {
		req := httptest.NewRequest(http.MethodGet, "/admin/tickets/attachments/content?key="+url.QueryEscape(key), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, "key=%q body=%s", key, w.Body.String())
		require.Empty(t, store.openedKey, "store must not be touched for out-of-scope keys")
	}
}
