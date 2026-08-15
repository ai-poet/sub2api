package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const settingKeyInvoiceStorageConfig = "invoice_storage_config"

// DefaultInvoiceStoragePrefix 是发票附件的默认对象前缀。
const DefaultInvoiceStoragePrefix = "invoices/"

var (
	ErrInvoiceStorageIncomplete = infraerrors.BadRequest(
		"INVOICE_STORAGE_INCOMPLETE",
		"invoice storage is missing bucket / access_key_id / secret_access_key",
	)
	ErrInvoiceStoragePrefixInvalid = infraerrors.BadRequest(
		"INVOICE_STORAGE_PREFIX_INVALID",
		"invoice storage prefix must be a non-empty relative path and must not contain '..'",
	)
	// ErrInvoiceStoragePrefixOverlapsBackup 阻止把发票前缀设成与备份前缀重叠的值。
	//
	// 这条路由不挂 step-up 2FA（改发票存储目标的影响面远小于备份），代价是必须在
	// 这里挡住「把发票前缀指向 backups/」——否则就等于绕开 step-up 打开了一条为
	// 数据库备份签发下载链接的通道。
	ErrInvoiceStoragePrefixOverlapsBackup = infraerrors.BadRequest(
		"INVOICE_STORAGE_PREFIX_OVERLAPS_BACKUP",
		"invoice storage prefix overlaps the database backup prefix in the same bucket; pick a different prefix",
	)
)

// InvoiceStorageSettings 是后台可编辑的发票附件对象存储配置。
//
// 与备份 S3 拆开是有意为之：两者的影响面完全不同——改备份目标可以把整库数据导向
// 外部账号，改发票目标最多影响此后新上传的发票文件。拆开后发票功能也不再依赖
// 「数据库备份」被配置过，并且可以放到与备份不同的桶甚至不同的服务商。
//
// ReuseBackupS3 为真时不保存自己的凭证，直接借用备份已配置的端点与密钥，只用
// Bucket/Prefix 区分对象；这样单桶部署无需重复填一遍。
type InvoiceStorageSettings struct {
	ReuseBackupS3 bool `json:"reuse_backup_s3"`

	Bucket string `json:"bucket"` // 留空且复用备份时，沿用备份桶
	Prefix string `json:"prefix"`

	// 以下仅在 ReuseBackupS3 为假时使用
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key,omitempty"` //nolint:revive // field name follows AWS convention
	ForcePathStyle  bool   `json:"force_path_style"`
}

// InvoiceStorageSettingService 读写后台设置，并解析成可直接使用的 S3 配置。
//
// 解析结果带缓存：每次上传/下载都要用，不能每次查库。保存设置时 Invalidate 清缓存，
// 下一次调用即重建——这是「后台改完立即生效、无需重启」的实现。
type InvoiceStorageSettingService struct {
	settingRepo SettingRepository
	encryptor   SecretEncryptor
	backup      *BackupService

	mu       sync.Mutex
	resolved bool
	cfg      *BackupS3Config
	prefix   string
}

func NewInvoiceStorageSettingService(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	backup *BackupService,
) *InvoiceStorageSettingService {
	return &InvoiceStorageSettingService{settingRepo: settingRepo, encryptor: encryptor, backup: backup}
}

// Invalidate 丢弃缓存，使下一次调用按最新设置重新解析。
func (s *InvoiceStorageSettingService) Invalidate() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.resolved = false
	s.cfg = nil
	s.prefix = ""
	s.mu.Unlock()
}

// Resolve 返回可用的 S3 配置与对象前缀；未配置完整时返回 ok=false。
func (s *InvoiceStorageSettingService) Resolve(ctx context.Context) (cfg *BackupS3Config, prefix string, ok bool) {
	if s == nil {
		return nil, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resolved {
		return s.cfg, s.prefix, s.cfg != nil
	}
	s.resolved = true
	s.cfg, s.prefix = nil, ""

	settings, err := s.load(ctx)
	if err != nil {
		logger.L().Warn("invoice_storage.settings_load_failed; invoice attachments stay disabled", zap.Error(err))
		return nil, "", false
	}
	if settings == nil {
		// 从未保存过：默认复用备份凭证，保持升级前的行为不被打断。
		settings = &InvoiceStorageSettings{ReuseBackupS3: true, Prefix: DefaultInvoiceStoragePrefix}
	}

	resolved, err := s.toS3Config(ctx, settings)
	if err != nil || resolved == nil || !resolved.IsConfigured() {
		if err != nil {
			logger.L().Warn("invoice_storage.resolve_failed; invoice attachments stay disabled", zap.Error(err))
		}
		return nil, "", false
	}

	s.cfg = resolved
	s.prefix = settings.Prefix
	return s.cfg, s.prefix, true
}

// Get 返回后台设置（SecretAccessKey 已脱敏）。
func (s *InvoiceStorageSettingService) Get(ctx context.Context) (*InvoiceStorageSettings, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = &InvoiceStorageSettings{ReuseBackupS3: true, Prefix: DefaultInvoiceStoragePrefix}
	}
	settings.SecretAccessKey = ""
	return settings, nil
}

// SecretConfigured 供前端展示「已配置」占位符。
func (s *InvoiceStorageSettingService) SecretConfigured(ctx context.Context) bool {
	settings, err := s.load(ctx)
	if err != nil || settings == nil {
		return false
	}
	if settings.ReuseBackupS3 {
		cfg, err := s.backupCredentials(ctx)
		return err == nil && cfg != nil && cfg.SecretAccessKey != ""
	}
	return settings.SecretAccessKey != ""
}

// Update 保存设置并立即生效。SecretAccessKey 留空表示沿用已保存的值。
func (s *InvoiceStorageSettingService) Update(ctx context.Context, in InvoiceStorageSettings) (*InvoiceStorageSettings, error) {
	if err := normalizeInvoiceStorageSettings(&in); err != nil {
		return nil, err
	}

	if in.ReuseBackupS3 {
		// 复用备份凭证时不落自己的密钥，避免同一份密钥在库里存两份。
		in.Endpoint, in.Region, in.AccessKeyID, in.SecretAccessKey = "", "", "", ""
		in.ForcePathStyle = false
	} else if in.SecretAccessKey == "" {
		if old, err := s.load(ctx); err == nil && old != nil {
			in.SecretAccessKey = old.SecretAccessKey
		}
	} else {
		// 拒绝用自动生成的临时密钥加密：重启后密文无法解密（#4524）。
		if s.backup == nil || !s.backup.EncryptionKeyConfigured() {
			return nil, ErrSecretEncryptionKeyNotConfigured
		}
		encrypted, err := s.encryptor.Encrypt(in.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt secret: %w", err)
		}
		in.SecretAccessKey = encrypted
	}

	if err := s.assertPrefixNotOverlappingBackup(ctx, &in); err != nil {
		return nil, err
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal invoice storage settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, settingKeyInvoiceStorageConfig, string(data)); err != nil {
		return nil, fmt.Errorf("save invoice storage settings: %w", err)
	}
	s.Invalidate()

	in.SecretAccessKey = ""
	return &in, nil
}

// TestConnection 用给定设置试连一次（HeadBucket），供后台「测试连接」按钮使用。
// 与 Update 一样支持留空 SecretAccessKey 表示沿用已保存的值。
func (s *InvoiceStorageSettingService) TestConnection(ctx context.Context, in InvoiceStorageSettings) error {
	if err := normalizeInvoiceStorageSettings(&in); err != nil {
		return err
	}
	if !in.ReuseBackupS3 && in.SecretAccessKey == "" {
		if old, err := s.load(ctx); err == nil && old != nil {
			in.SecretAccessKey = old.SecretAccessKey
		}
	}
	if err := s.assertPrefixNotOverlappingBackup(ctx, &in); err != nil {
		return err
	}
	cfg, err := s.toS3Config(ctx, &in)
	if err != nil {
		return err
	}
	if cfg == nil || !cfg.IsConfigured() {
		return ErrInvoiceStorageIncomplete
	}
	if s.backup == nil || s.backup.storeFactory == nil {
		return errors.New("backup object store factory is unavailable")
	}
	// 复用备份的 store 工厂：HeadBucket 只是探活，没必要再造一套客户端构造逻辑。
	store, err := s.backup.storeFactory(ctx, cfg)
	if err != nil {
		return err
	}
	return store.HeadBucket(ctx)
}

// assertPrefixNotOverlappingBackup 在同桶时拒绝与备份前缀重叠的发票前缀。
func (s *InvoiceStorageSettingService) assertPrefixNotOverlappingBackup(ctx context.Context, in *InvoiceStorageSettings) error {
	backupCfg, err := s.backupCredentials(ctx)
	if err != nil || backupCfg == nil {
		return nil //nolint:nilerr // 备份未配置时无从比较，交由后续步骤判断
	}

	bucket := in.Bucket
	if bucket == "" && in.ReuseBackupS3 {
		bucket = backupCfg.Bucket
	}
	if bucket == "" || bucket != backupCfg.Bucket {
		return nil // 不同桶，互不干扰
	}

	backupPrefix := strings.TrimSpace(backupCfg.Prefix)
	if backupPrefix == "" {
		// 备份写在桶根目录，任何前缀都在其"射程"内。
		return ErrInvoiceStoragePrefixOverlapsBackup
	}
	if !strings.HasSuffix(backupPrefix, "/") {
		backupPrefix += "/"
	}
	if strings.HasPrefix(in.Prefix, backupPrefix) || strings.HasPrefix(backupPrefix, in.Prefix) {
		return ErrInvoiceStoragePrefixOverlapsBackup
	}
	return nil
}

func (s *InvoiceStorageSettingService) toS3Config(ctx context.Context, in *InvoiceStorageSettings) (*BackupS3Config, error) {
	cfg := &BackupS3Config{
		Endpoint:        in.Endpoint,
		Region:          in.Region,
		Bucket:          in.Bucket,
		AccessKeyID:     in.AccessKeyID,
		SecretAccessKey: in.SecretAccessKey,
		// Prefix 不放进 S3 客户端配置：发票的对象前缀由 PayAttachmentService 统一
		// 拼接与校验，避免两处各拼一遍。
		ForcePathStyle: in.ForcePathStyle,
	}

	if in.ReuseBackupS3 {
		backupCfg, err := s.backupCredentials(ctx)
		if err != nil {
			return nil, err
		}
		if backupCfg == nil {
			return nil, errors.New("invoice storage is set to reuse the backup S3 configuration, but no backup S3 configuration exists")
		}
		cfg.Endpoint = backupCfg.Endpoint
		cfg.Region = backupCfg.Region
		cfg.AccessKeyID = backupCfg.AccessKeyID
		cfg.SecretAccessKey = backupCfg.SecretAccessKey
		cfg.ForcePathStyle = backupCfg.ForcePathStyle
		if cfg.Bucket == "" {
			cfg.Bucket = backupCfg.Bucket
		}
		return cfg, nil
	}

	if cfg.SecretAccessKey != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.SecretAccessKey)
		if err != nil {
			// 兼容未加密的旧数据，与备份配置的处理保持一致。
			logger.L().Warn("invoice_storage secret decrypt failed; treating the stored value as plaintext", zap.Error(err))
		} else {
			cfg.SecretAccessKey = decrypted
		}
	}
	return cfg, nil
}

// backupCredentials 取备份已配置的 S3 凭证（已解密）。
func (s *InvoiceStorageSettingService) backupCredentials(ctx context.Context) (*BackupS3Config, error) {
	if s.backup == nil {
		return nil, errors.New("backup service is unavailable")
	}
	return s.backup.loadS3Config(ctx)
}

func (s *InvoiceStorageSettingService) load(ctx context.Context) (*InvoiceStorageSettings, error) {
	if s.settingRepo == nil {
		return nil, nil //nolint:nilnil // no repository means no stored settings
	}
	raw, err := s.settingRepo.GetValue(ctx, settingKeyInvoiceStorageConfig)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, nil //nolint:nilnil // never configured is a valid state
	}
	var settings InvoiceStorageSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("parse invoice storage settings: %w", err)
	}
	if settings.Prefix == "" {
		settings.Prefix = DefaultInvoiceStoragePrefix
	}
	return &settings, nil
}

// normalizeInvoiceStorageSettings 归一化并校验前缀。
//
// 前缀同时是 presign/delete 的授权边界，所以这里的校验不是格式美化：空前缀会让
// 边界失效，'..' 会让规约后的 key 逃出边界。
func normalizeInvoiceStorageSettings(in *InvoiceStorageSettings) error {
	in.Bucket = strings.TrimSpace(in.Bucket)
	in.Endpoint = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(in.Endpoint), "/"))
	in.Region = strings.TrimSpace(in.Region)
	in.AccessKeyID = strings.TrimSpace(in.AccessKeyID)
	in.SecretAccessKey = strings.TrimSpace(in.SecretAccessKey)

	prefix := strings.TrimSpace(in.Prefix)
	if prefix == "" {
		prefix = DefaultInvoiceStoragePrefix
	}
	prefix = strings.TrimPrefix(prefix, "/")
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if prefix == "/" || strings.Contains(prefix, "..") {
		return ErrInvoiceStoragePrefixInvalid
	}
	in.Prefix = prefix

	if !in.ReuseBackupS3 && in.Region == "" {
		in.Region = "auto"
	}
	return nil
}
