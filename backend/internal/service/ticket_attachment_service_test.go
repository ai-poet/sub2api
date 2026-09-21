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
	return newTicketAttachmentServiceWithTicketsForTest(t, newTicketRepoStub())
}

// newTicketAttachmentServiceWithTicketsForTest 接一个工单仓储（可为 nil），供读取授权的测试注入
// "客服消息引用了哪个 key" 的关系。
func newTicketAttachmentServiceWithTicketsForTest(t *testing.T, tickets SupportTicketRepository) (*TicketAttachmentService, *fakeTicketAttachmentStore) {
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
	}, tickets)
	return svc, store
}

// seedTicketWithMessages 直接往桩仓储里放一张工单及其消息（同包可访问内部 map）。
func seedTicketWithMessages(tickets *ticketRepoStub, ticketID, userID int64, msgs ...*SupportTicketMessage) {
	tickets.mu.Lock()
	defer tickets.mu.Unlock()
	tickets.tickets[ticketID] = &SupportTicket{ID: ticketID, UserID: userID, Status: TicketStatusReplied}
	for i, m := range msgs {
		m.ID = int64(i + 1)
		m.TicketID = ticketID
	}
	tickets.messages[ticketID] = msgs
}

func staffAttachmentMarkdown(key string) string {
	return "看图 ![image](" + TicketAttachmentURLScheme + key + ")"
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

	// 他人 / 未被引用的客服 / 前缀外 / 穿越的 key 一律按不存在处理，不泄露存在性。
	// （客服 key 只有被自己工单里的客服消息引用才放行，见下一条测试。）
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

// 客服回复里贴的图落在 staff/ 前缀下，工单发起人必须能看到——但只限被自己工单里
// 客服消息引用的 key：这是用户读到非自己前缀 key 的唯一通道。
func TestTicketAttachmentOpenForUserAllowsStaffKeyReferencedInOwnTicket(t *testing.T) {
	ctx := context.Background()
	staffKey := DefaultTicketAttachmentStoragePrefix + "staff/7/202601/a1b2c3d4.png"
	otherStaffKey := DefaultTicketAttachmentStoragePrefix + "staff/7/202601/ffffffff.png"
	otherUserKey := DefaultTicketAttachmentStoragePrefix + "43/202601/a1b2c3d4.png"

	tickets := newTicketRepoStub()
	// 工单 1：user 42 的，客服回复引用 staffKey，同时（误）引用了 user 43 的 key。
	seedTicketWithMessages(tickets, 1, 42,
		&SupportTicketMessage{AuthorUserID: 42, AuthorRole: TicketAuthorRoleUser, Body: "求助"},
		&SupportTicketMessage{AuthorUserID: 7, AuthorRole: RoleAdmin, Body: staffAttachmentMarkdown(staffKey) + " " + staffAttachmentMarkdown(otherUserKey)},
	)
	// 工单 2：user 42 自己在正文里夹带另一个客服 key——用户写的消息不构成授权。
	seedTicketWithMessages(tickets, 2, 42,
		&SupportTicketMessage{AuthorUserID: 42, AuthorRole: TicketAuthorRoleUser, Body: staffAttachmentMarkdown(otherStaffKey)},
	)
	svc, store := newTicketAttachmentServiceWithTicketsForTest(t, tickets)

	content, err := svc.OpenForUser(ctx, staffKey, 42)
	if err != nil {
		t.Fatalf("OpenForUser(referenced staff key): %v", err)
	}
	_ = content.Body.Close()
	if store.openedKey != staffKey {
		t.Fatalf("opened key = %q, want %q", store.openedKey, staffKey)
	}

	// 不是工单发起人 → 404；被用户消息夹带的 → 404；客服消息里贴的他人前缀 key → 404；穿越 / 未引用 → 404。
	store.openedKey = ""
	for _, tc := range []struct {
		name   string
		key    string
		userID int64
	}{
		{"other user", staffKey, 43},
		{"smuggled by user message", otherStaffKey, 42},
		{"another user's own-prefix key pasted by staff", otherUserKey, 42},
		{"traversal under staff prefix", DefaultTicketAttachmentStoragePrefix + "staff/../backups/dump.sql.gz", 42},
		{"unreferenced staff key", DefaultTicketAttachmentStoragePrefix + "staff/7/202601/deadbeef.png", 42},
	} {
		if _, err := svc.OpenForUser(ctx, tc.key, tc.userID); err != ErrTicketAttachmentNotFound {
			t.Fatalf("%s: OpenForUser(%q, %d): got %v, want ErrTicketAttachmentNotFound", tc.name, tc.key, tc.userID, err)
		}
		if store.openedKey != "" {
			t.Fatalf("%s: store must not be touched, opened %q", tc.name, store.openedKey)
		}
	}

	// 没接工单仓储（nil）就退回只读自己前缀的旧行为，绝不放行 staff key。
	svcNoTickets, storeNoTickets := newTicketAttachmentServiceWithTicketsForTest(t, nil)
	if _, err := svcNoTickets.OpenForUser(ctx, staffKey, 42); err != ErrTicketAttachmentNotFound {
		t.Fatalf("OpenForUser without ticket repo: got %v, want ErrTicketAttachmentNotFound", err)
	}
	if storeNoTickets.openedKey != "" {
		t.Fatalf("store must not be touched without ticket repo, opened %q", storeNoTickets.openedKey)
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
	}, nil)

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
