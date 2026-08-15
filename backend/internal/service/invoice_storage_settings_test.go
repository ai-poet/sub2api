package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func newInvoiceStorageFixture(t *testing.T, backupCfg *BackupS3Config) (*InvoiceStorageSettingService, *payAttachmentSettingRepo) {
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
	return NewInvoiceStorageSettingService(repo, encryptor, backup), repo
}

func defaultBackupCfg() *BackupS3Config {
	return &BackupS3Config{
		Endpoint: "http://127.0.0.1:9000", Region: "us-east-1", Bucket: "zeabur",
		AccessKeyID: "minio", SecretAccessKey: "sk", Prefix: "backups/", ForcePathStyle: true,
	}
}

// 未配置过时默认复用备份凭证，保证升级前已配好备份的部署不被打断。
func TestInvoiceStorageDefaultsToReusingBackupCredentials(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())

	cfg, prefix, ok := svc.Resolve(context.Background())
	if !ok {
		t.Fatal("expected the default settings to resolve from the backup credentials")
	}
	if cfg.Bucket != "zeabur" || cfg.AccessKeyID != "minio" || cfg.SecretAccessKey != "sk" {
		t.Fatalf("unexpected resolved config: %+v", cfg)
	}
	if !cfg.ForcePathStyle {
		t.Fatal("force_path_style must be inherited from the backup config")
	}
	if prefix != DefaultInvoiceStoragePrefix {
		t.Fatalf("prefix = %q, want %q", prefix, DefaultInvoiceStoragePrefix)
	}
}

func TestInvoiceStorageFailsClosedWithoutAnyCredentials(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, nil)
	if _, _, ok := svc.Resolve(context.Background()); ok {
		t.Fatal("expected resolve to fail when neither invoice nor backup storage is configured")
	}
}

func TestInvoiceStorageStandaloneCredentialsOverrideBackup(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, InvoiceStorageSettings{
		Bucket: "invoices-only", Prefix: "fapiao/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2", ForcePathStyle: true,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	cfg, prefix, ok := svc.Resolve(ctx)
	if !ok {
		t.Fatal("expected standalone settings to resolve")
	}
	if cfg.Bucket != "invoices-only" || cfg.AccessKeyID != "ak2" {
		t.Fatalf("unexpected resolved config: %+v", cfg)
	}
	// 密钥落库时加密，解析时解密。
	if cfg.SecretAccessKey != "sk2" {
		t.Fatalf("secret = %q, want the decrypted value", cfg.SecretAccessKey)
	}
	if prefix != "fapiao/" {
		t.Fatalf("prefix = %q, want fapiao/", prefix)
	}
}

func TestInvoiceStorageNormalizesPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", DefaultInvoiceStoragePrefix},
		{"invoice", "invoice/"},
		{"/invoice/", "invoice/"},
		{"  fapiao  ", "fapiao/"},
	}
	for _, tc := range cases {
		settings := InvoiceStorageSettings{Prefix: tc.in, ReuseBackupS3: true}
		if err := normalizeInvoiceStorageSettings(&settings); err != nil {
			t.Fatalf("normalize(%q): %v", tc.in, err)
		}
		if settings.Prefix != tc.want {
			t.Fatalf("normalize(%q) = %q, want %q", tc.in, settings.Prefix, tc.want)
		}
	}
}

// 空前缀会让 presign 的授权边界失效，'..' 会让规约后的 key 逃出边界。
func TestInvoiceStorageRejectsDangerousPrefix(t *testing.T) {
	for _, prefix := range []string{"/", "../", "a/../../b"} {
		settings := InvoiceStorageSettings{Prefix: prefix, ReuseBackupS3: true}
		if err := normalizeInvoiceStorageSettings(&settings); err != ErrInvoiceStoragePrefixInvalid {
			t.Fatalf("normalize(%q): got %v, want ErrInvoiceStoragePrefixInvalid", prefix, err)
		}
	}
}

// 这条守卫替代了 step-up 2FA：没有它，管理员可以把发票前缀指向 backups/，
// 从而绕开 step-up 打开一条给数据库备份签发下载链接的通道。
func TestInvoiceStorageRejectsPrefixOverlappingBackup(t *testing.T) {
	ctx := context.Background()

	// 归一化后带尾斜杠，所以覆盖两个方向：发票前缀在备份之下、备份前缀在发票之下。
	for _, prefix := range []string{"backups/", "backups/invoices/", "backups/2026/"} {
		t.Run(prefix, func(t *testing.T) {
			svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
			_, err := svc.Update(ctx, InvoiceStorageSettings{ReuseBackupS3: true, Prefix: prefix})
			if err != ErrInvoiceStoragePrefixOverlapsBackup {
				t.Fatalf("Update(prefix=%q): got %v, want ErrInvoiceStoragePrefixOverlapsBackup", prefix, err)
			}
		})
	}
}

// 只是长得像不算重叠：back/ 与 backups/ 在 S3 里是两个互不包含的命名空间，
// 不该被误伤。
func TestInvoiceStorageAllowsPrefixThatMerelyLooksLikeBackup(t *testing.T) {
	for _, prefix := range []string{"back", "backup-invoices", "invoices-backups"} {
		t.Run(prefix, func(t *testing.T) {
			svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
			if _, err := svc.Update(context.Background(), InvoiceStorageSettings{ReuseBackupS3: true, Prefix: prefix}); err != nil {
				t.Fatalf("Update(prefix=%q): %v", prefix, err)
			}
		})
	}
}

// 不同桶就没有重叠问题，同名前缀应当放行。
func TestInvoiceStorageAllowsBackupPrefixInADifferentBucket(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())

	if _, err := svc.Update(context.Background(), InvoiceStorageSettings{
		Bucket: "another-bucket", Prefix: "backups/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

// 备份写在桶根目录时，同桶内任何前缀都可能与之重叠。
func TestInvoiceStorageRejectsAnyPrefixWhenBackupUsesBucketRoot(t *testing.T) {
	cfg := defaultBackupCfg()
	cfg.Prefix = ""
	svc, _ := newInvoiceStorageFixture(t, cfg)

	_, err := svc.Update(context.Background(), InvoiceStorageSettings{ReuseBackupS3: true, Prefix: "invoices/"})
	if err != ErrInvoiceStoragePrefixOverlapsBackup {
		t.Fatalf("got %v, want ErrInvoiceStoragePrefixOverlapsBackup", err)
	}
}

// 留空密钥表示沿用已保存的值，避免前端回显脱敏值时把密钥清空。
func TestInvoiceStorageKeepsStoredSecretWhenOmitted(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	base := InvoiceStorageSettings{
		Bucket: "invoices", Prefix: "invoices/", Endpoint: "https://s3.example.com",
		Region: "us-east-1", AccessKeyID: "ak2", SecretAccessKey: "sk2",
	}
	if _, err := svc.Update(ctx, base); err != nil {
		t.Fatalf("first Update: %v", err)
	}

	base.SecretAccessKey = ""
	base.Bucket = "invoices-renamed"
	if _, err := svc.Update(ctx, base); err != nil {
		t.Fatalf("second Update: %v", err)
	}

	cfg, _, ok := svc.Resolve(ctx)
	if !ok || cfg.SecretAccessKey != "sk2" {
		t.Fatalf("secret was lost on the second update: ok=%v cfg=%+v", ok, cfg)
	}
}

// Get 必须脱敏，绝不能把密钥回传给前端。
func TestInvoiceStorageGetMasksSecret(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, InvoiceStorageSettings{
		Bucket: "invoices", Prefix: "invoices/", Endpoint: "https://s3.example.com",
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
func TestInvoiceStorageUpdateInvalidatesCache(t *testing.T) {
	svc, _ := newInvoiceStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, _, ok := svc.Resolve(ctx); !ok {
		t.Fatal("expected the initial resolve to succeed")
	}

	if _, err := svc.Update(ctx, InvoiceStorageSettings{ReuseBackupS3: true, Prefix: "fapiao/"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	_, prefix, ok := svc.Resolve(ctx)
	if !ok || prefix != "fapiao/" {
		t.Fatalf("prefix = %q (ok=%v), want fapiao/ without a restart", prefix, ok)
	}
}

// 复用备份凭证时不应把密钥再存一份到自己的设置里。
func TestInvoiceStorageDoesNotDuplicateSecretWhenReusingBackup(t *testing.T) {
	svc, repo := newInvoiceStorageFixture(t, defaultBackupCfg())
	ctx := context.Background()

	if _, err := svc.Update(ctx, InvoiceStorageSettings{
		ReuseBackupS3: true, Prefix: "invoices/", AccessKeyID: "leak", SecretAccessKey: "leak-secret",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	raw, _ := repo.GetValue(ctx, settingKeyInvoiceStorageConfig)
	var stored InvoiceStorageSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.SecretAccessKey != "" || stored.AccessKeyID != "" {
		t.Fatalf("credentials were duplicated into the invoice settings: %+v", stored)
	}
}
