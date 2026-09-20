package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func newTicketAttachmentStorageFixture(t *testing.T, backupCfg *BackupS3Config) (*TicketAttachmentStorageSettingService, *payAttachmentSettingRepo) {
	t.Helper()
	repo := newPayAttachmentSettingRepo()
	encryptor := payAttachmentEncryptor{}
	backup := NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, encryptor, nil, nil)

	if backupCfg != nil {
		stored := *backupCfg
		stored.SecretAccessKey = "enc:" + stored.SecretAccessKey
		raw, err := json.Marshal(stored)
		if err != nil {
			t.Fatalf("marshal backup config: %v", err)
		}
		if err := repo.Set(context.Background(), settingKeyBackupS3Config, string(raw)); err != nil {
			t.Fatalf("seed backup config: %v", err)
		}
	}
	return NewTicketAttachmentStorageSettingService(repo, encryptor, backup), repo
}

// 未配置过时默认复用备份凭证，保持与发票存储一致的升级行为。
func TestTicketAttachmentStorageDefaultsToReusingBackupCredentials(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())

	cfg, prefix, ok := svc.Resolve(context.Background())
	if !ok {
		t.Fatal("expected the default settings to resolve from the backup credentials")
	}
	if cfg.Bucket != "zeabur" || cfg.AccessKeyID != "minio" || cfg.SecretAccessKey != "sk" {
		t.Fatalf("unexpected resolved config: %+v", cfg)
	}
	if prefix != DefaultTicketAttachmentStoragePrefix {
		t.Fatalf("prefix = %q, want %q", prefix, DefaultTicketAttachmentStoragePrefix)
	}
}

func TestTicketAttachmentStorageFailsClosedWithoutAnyCredentials(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, nil)
	if _, _, ok := svc.Resolve(context.Background()); ok {
		t.Fatal("expected resolve to fail when neither ticket attachment nor backup storage is configured")
	}
}

func TestTicketAttachmentStorageStandaloneCredentialsOverrideBackup(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, TicketAttachmentStorageSettings{
		Bucket: "tickets-only", Prefix: "gongdan/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2", ForcePathStyle: true,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	cfg, prefix, ok := svc.Resolve(ctx)
	if !ok {
		t.Fatal("expected standalone settings to resolve")
	}
	if cfg.Bucket != "tickets-only" || cfg.AccessKeyID != "ak2" || cfg.SecretAccessKey != "sk2" {
		t.Fatalf("unexpected resolved config: %+v", cfg)
	}
	if prefix != "gongdan/" {
		t.Fatalf("prefix = %q, want gongdan/", prefix)
	}
}

// 前缀同时是 content 端点的授权边界：空前缀会让边界失效，'..' 会让规约后的 key 逃出边界。
func TestTicketAttachmentStorageRejectsDangerousPrefix(t *testing.T) {
	for _, prefix := range []string{"/", "../", "a/../../b"} {
		settings := TicketAttachmentStorageSettings{Prefix: prefix, ReuseBackupS3: true}
		if err := normalizeTicketAttachmentStorageSettings(&settings); err != ErrTicketAttachmentStoragePrefixInvalid {
			t.Fatalf("normalize(%q): got %v, want ErrTicketAttachmentStoragePrefixInvalid", prefix, err)
		}
	}
}

// 附件前缀与备份前缀重叠时，content 端点就成了读取数据库备份的通道。
func TestTicketAttachmentStorageRejectsPrefixOverlappingBackup(t *testing.T) {
	ctx := context.Background()

	for _, prefix := range []string{"backups/", "backups/tickets/", "backups/2026/"} {
		t.Run(prefix, func(t *testing.T) {
			svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
			_, err := svc.Update(ctx, TicketAttachmentStorageSettings{ReuseBackupS3: true, Prefix: prefix})
			if err != ErrTicketAttachmentStoragePrefixOverlapsBackup {
				t.Fatalf("Update(prefix=%q): got %v, want ErrTicketAttachmentStoragePrefixOverlapsBackup", prefix, err)
			}
		})
	}
}

// 只是长得像不算重叠；不同桶也没有重叠问题。
func TestTicketAttachmentStorageAllowsNonOverlappingPrefix(t *testing.T) {
	for _, prefix := range []string{"back", "backup-tickets", "tickets-backups"} {
		t.Run(prefix, func(t *testing.T) {
			svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
			if _, err := svc.Update(context.Background(), TicketAttachmentStorageSettings{ReuseBackupS3: true, Prefix: prefix}); err != nil {
				t.Fatalf("Update(prefix=%q): %v", prefix, err)
			}
		})
	}

	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	if _, err := svc.Update(context.Background(), TicketAttachmentStorageSettings{
		Bucket: "another-bucket", Prefix: "backups/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2",
	}); err != nil {
		t.Fatalf("Update(different bucket): %v", err)
	}
}

// 留空密钥表示沿用已保存的值，避免前端回显脱敏值时把密钥清空。
func TestTicketAttachmentStorageKeepsStoredSecretWhenOmitted(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	base := TicketAttachmentStorageSettings{
		Bucket: "tickets", Prefix: "tickets/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2",
	}
	if _, err := svc.Update(ctx, base); err != nil {
		t.Fatalf("first Update: %v", err)
	}

	base.SecretAccessKey = ""
	base.Bucket = "tickets-renamed"
	if _, err := svc.Update(ctx, base); err != nil {
		t.Fatalf("second Update: %v", err)
	}

	cfg, _, ok := svc.Resolve(ctx)
	if !ok || cfg.SecretAccessKey != "sk2" {
		t.Fatalf("secret was lost on the second update: ok=%v cfg=%+v", ok, cfg)
	}
}

// Get 必须脱敏，绝不能把密钥回传给前端。
func TestTicketAttachmentStorageGetMasksSecret(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, TicketAttachmentStorageSettings{
		Bucket: "tickets", Prefix: "tickets/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SecretAccessKey != "" {
		t.Fatalf("Get leaked the secret: %q", got.SecretAccessKey)
	}
	if !svc.SecretConfigured(ctx) {
		t.Fatal("SecretConfigured should report true after saving a secret")
	}
}

// 改设置后必须立即生效，不能等重启。
func TestTicketAttachmentStorageUpdateInvalidatesCache(t *testing.T) {
	svc, _ := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, _, ok := svc.Resolve(ctx); !ok {
		t.Fatal("expected the initial resolve to succeed")
	}

	if _, err := svc.Update(ctx, TicketAttachmentStorageSettings{ReuseBackupS3: true, Prefix: "gongdan/"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	_, prefix, ok := svc.Resolve(ctx)
	if !ok || prefix != "gongdan/" {
		t.Fatalf("prefix = %q (ok=%v), want gongdan/ without a restart", prefix, ok)
	}
}

// 复用备份凭证时不应把密钥再存一份到自己的设置里。
func TestTicketAttachmentStorageDoesNotDuplicateSecretWhenReusingBackup(t *testing.T) {
	svc, repo := newTicketAttachmentStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, TicketAttachmentStorageSettings{
		ReuseBackupS3: true, Prefix: "tickets/", AccessKeyID: "leak", SecretAccessKey: "leak-secret",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	raw, _ := repo.GetValue(ctx, settingKeyTicketAttachmentStorageConfig)
	var stored TicketAttachmentStorageSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.SecretAccessKey != "" || stored.AccessKeyID != "" {
		t.Fatalf("credentials were duplicated into the ticket attachment settings: %+v", stored)
	}
}
