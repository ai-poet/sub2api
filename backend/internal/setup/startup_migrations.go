package setup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/repository"
)

// startupMigrationTimeout 是正常启动时的迁移超时。
// 比安装期更宽松：升级往往一次性积压多个版本的迁移，其中包含
// CONCURRENTLY 建索引这类耗时操作。
const startupMigrationTimeout = 10 * time.Minute

// MigrateOnStartup 在服务正常启动时应用尚未执行的数据库迁移。
//
// 背景：迁移原本只在首次安装路径（Install / AutoSetupFromEnv）里执行，
// 已安装实例升级重启时 NeedsSetup() 为 false，会直接进入主服务而跳过迁移，
// 导致新版本代码访问尚不存在的列。这里让每次启动都对齐一次 schema。
//
// 重复执行是安全的：迁移器持有 PostgreSQL advisory lock（多实例只有一个执行），
// 按文件名记录已应用的迁移，并用 SHA256 校验和防止已应用文件被篡改。
//
// SKIP_SETUP=true 的实例被视为只读接入一个已初始化的库（例如多个实例共用一套
// PostgreSQL，只由其中一个负责升级）：除了跳过安装流程，这里也整体跳过，
// 连数据库连接都不建立，保证该实例不会对 schema 做任何改动。
// 代价是这类实例要求库的 schema 不低于其镜像版本，否则启动后会因缺列报错。
func MigrateOnStartup(cfg *config.Config) error {
	if skipSetupEnabled() {
		logger.LegacyPrintf("setup", "%s", "SKIP_SETUP enabled, skipping startup database migrations")
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("nil config")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open postgres for startup migrations: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close startup migration connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), startupMigrationTimeout)
	defer cancel()

	if err := repository.ApplyMigrations(ctx, db); err != nil {
		return fmt.Errorf("apply startup migrations: %w", err)
	}
	return nil
}
