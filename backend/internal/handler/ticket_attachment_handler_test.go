//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ---------- 测试骨架 ----------

type ticketAttachmentFakeStore struct {
	uploadedKey string
	openedKey   string
}

func (s *ticketAttachmentFakeStore) Upload(_ context.Context, key, _ string, _ []byte) error {
	s.uploadedKey = key
	return nil
}

func (s *ticketAttachmentFakeStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	s.openedKey = key
	body := "\x89PNG\r\n\x1a\n stored"
	return io.NopCloser(strings.NewReader(body)), "image/png", int64(len(body)), nil
}

// newTicketAttachmentServiceForHandlerTest 用种子备份配置 + fake store 构造真实服务，
// 覆盖 settings 解析与 key 前缀边界，而不是只测 handler 的空壳。
func newTicketAttachmentServiceForHandlerTest(t *testing.T) (*service.TicketAttachmentService, *ticketAttachmentFakeStore) {
	t.Helper()
	repo := &stubSettingRepoForPublicSettings{values: map[string]string{}}
	backup := service.NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, payBridgeEncryptor{}, nil, nil)

	raw, err := json.Marshal(service.BackupS3Config{
		Region: "us-east-1", Bucket: "sub2api", AccessKeyID: "ak", SecretAccessKey: "sk", Prefix: "backups/",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Set(context.Background(), "backup_s3_config", string(raw)))

	store := &ticketAttachmentFakeStore{}
	settings := service.NewTicketAttachmentStorageSettingService(repo, payBridgeEncryptor{}, backup)
	svc := service.NewTicketAttachmentService(settings, func(context.Context, *service.BackupS3Config) (service.TicketAttachmentStore, error) {
		return store, nil
	})
	return svc, store
}

func newTicketAttachmentUserRouter(t *testing.T, userID int64) (*gin.Engine, *ticketAttachmentFakeStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc, store := newTicketAttachmentServiceForHandlerTest(t)
	h := NewTicketAttachmentHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.POST("/tickets/attachments",
		middleware.RequestBodyLimit(service.MaxTicketAttachmentBytes+(1<<20)), h.Upload)
	router.GET("/tickets/attachments/content", h.Content)
	return router, store
}

func multipartImageRequest(t *testing.T, target, contentType string, data []byte) *http.Request {
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

func handlerTestPNG() []byte {
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")
}

// ---------- 上传 ----------

func TestTicketAttachmentUploadStoresImageAndReturnsKey(t *testing.T) {
	router, store := newTicketAttachmentUserRouter(t, 42)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, multipartImageRequest(t, "/tickets/attachments", "image/png", handlerTestPNG()))

	require.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Data service.TicketAttachmentPutResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, strings.HasPrefix(resp.Data.Key, service.DefaultTicketAttachmentStoragePrefix+"42/"),
		"key %q must land under tickets/<uid>/", resp.Data.Key)
	require.True(t, strings.HasSuffix(resp.Data.Key, ".png"))
	require.Equal(t, "image/png", resp.Data.ContentType)
	require.Equal(t, len(handlerTestPNG()), resp.Data.Size)
	require.Equal(t, resp.Data.Key, store.uploadedKey)
}

func TestTicketAttachmentUploadRejectsBadType(t *testing.T) {
	router, _ := newTicketAttachmentUserRouter(t, 42)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, multipartImageRequest(t, "/tickets/attachments", "text/html", handlerTestPNG()))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "TICKET_ATTACHMENT_BAD_TYPE")

	// HTML 内容伪装成 PNG（魔数嗅探兜底）
	w = httptest.NewRecorder()
	router.ServeHTTP(w, multipartImageRequest(t, "/tickets/attachments", "image/png",
		[]byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>")))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "TICKET_ATTACHMENT_BAD_TYPE")
}

func TestTicketAttachmentUploadRejectsOversizeWith413(t *testing.T) {
	router, _ := newTicketAttachmentUserRouter(t, 42)

	// 文件字节略超 5 MiB（仍在路由 body 上限内），由服务层判 413。
	data := append(handlerTestPNG(), bytes.Repeat([]byte("A"), service.MaxTicketAttachmentBytes)...)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, multipartImageRequest(t, "/tickets/attachments", "image/png", data))
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), "TICKET_ATTACHMENT_TOO_LARGE")

	// 整体超路由 body 上限，由 MaxBytesReader 判 413。
	big := append(handlerTestPNG(), bytes.Repeat([]byte("B"), service.MaxTicketAttachmentBytes+(2<<20))...)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, multipartImageRequest(t, "/tickets/attachments", "image/png", big))
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "body=%s", w.Body.String())
}

// ---------- 同源回传的前缀边界 ----------

func TestTicketAttachmentContentStreamsOwnKey(t *testing.T) {
	router, store := newTicketAttachmentUserRouter(t, 42)

	key := service.DefaultTicketAttachmentStoragePrefix + "42/202601/a1b2c3d4.png"
	req := httptest.NewRequest(http.MethodGet, "/tickets/attachments/content?key="+url.QueryEscape(key), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	require.Equal(t, key, store.openedKey)
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestTicketAttachmentContentHidesOtherUsersKeys(t *testing.T) {
	router, store := newTicketAttachmentUserRouter(t, 42)

	for _, key := range []string{
		service.DefaultTicketAttachmentStoragePrefix + "43/202601/a1b2c3d4.png", // 他人的 key
		service.DefaultTicketAttachmentStoragePrefix + "staff/7/202601/a1b2c3d4.png",
		service.DefaultTicketAttachmentStoragePrefix + "42/../backups/dump.sql.gz",
		"backups/2026/08/14/dump.sql.gz",
	} {
		req := httptest.NewRequest(http.MethodGet, "/tickets/attachments/content?key="+url.QueryEscape(key), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusNotFound, w.Code, fmt.Sprintf("key=%q body=%s", key, w.Body.String()))
		require.Empty(t, store.openedKey, "store must not be touched for out-of-scope keys")
	}
}
