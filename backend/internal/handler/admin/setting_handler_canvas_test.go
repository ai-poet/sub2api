package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestDiffSettingsCanvasStudioEnabled(t *testing.T) {
	before := &service.SystemSettings{CanvasStudioEnabled: false}
	after := &service.SystemSettings{CanvasStudioEnabled: true}

	changed := diffSettings(before, after, nil, nil, UpdateSettingsRequest{})
	if len(changed) != 1 || changed[0] != service.SettingKeyCanvasStudioEnabled {
		t.Fatalf("changed = %v, want only canvas_studio_enabled", changed)
	}
}
