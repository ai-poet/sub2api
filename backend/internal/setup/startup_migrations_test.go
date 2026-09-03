package setup

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// SKIP_SETUP=true 的实例只读接入一个已初始化的库：
// 启动期迁移必须整体跳过，且不能尝试建立数据库连接。
// 这里故意指向一个不可达的端口，若真的去连库会立刻返回错误。
func TestMigrateOnStartupSkipsWhenSkipSetupIsEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "true", value: "true"},
		{name: "one", value: "1"},
		{name: "yes", value: "yes"},
		{name: "trimmed mixed case true", value: "  TrUe  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SKIP_SETUP", tc.value)

			cfg := &config.Config{}
			cfg.Database.Host = "127.0.0.1"
			cfg.Database.Port = 1
			cfg.Database.User = "nobody"
			cfg.Database.DBName = "nonexistent"
			cfg.Database.SSLMode = "disable"

			if err := MigrateOnStartup(cfg); err != nil {
				t.Fatalf("MigrateOnStartup() error = %v, want nil when SKIP_SETUP is enabled", err)
			}
		})
	}
}

// SKIP_SETUP 关闭时不能被其他条件短路：nil 配置必须报错，
// 说明代码走到了真正的迁移路径。
func TestMigrateOnStartupRejectsNilConfigWhenSkipSetupIsDisabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "false", value: "false"},
		{name: "empty", value: ""},
		{name: "invalid", value: "enabled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SKIP_SETUP", tc.value)

			if err := MigrateOnStartup(nil); err == nil {
				t.Fatal("MigrateOnStartup(nil) error = nil, want error when SKIP_SETUP is disabled")
			}
		})
	}
}
