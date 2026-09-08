package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildGroupStatusNotifyMessage_AstraMismatch(t *testing.T) {
	group := &Group{ID: 9, Name: "Astra 高速", Platform: PlatformOpenAI}
	latency := int64(4200)
	event := &GroupStatusEvent{
		EventType:   GroupStatusEventAstraMismatch,
		FromStatus:  "",
		ToStatus:    AstraCheckStatusMismatch,
		SubStatus:   "winner_gpt-5.6-sol",
		LatencyMS:   &latency,
		ErrorDetail: "Astra 0.010/0.812 · Sol 0.980/0.659 · valid 20/20 · benchmark 4.5.0-rc4",
		ObservedAt:  time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("My Gateway", group, event)

	require.Equal(t, "[My Gateway] 分组「Astra 高速」Astra 指纹疑似非 Astra（强指向 Sol）", title)
	require.Contains(t, desp, "**分组**：Astra 高速（#9 / openai）")
	require.Contains(t, desp, "**状态**：未知 → Astra 指纹不符")
	require.Contains(t, desp, "**子状态**：winner_gpt-5.6-sol")
	require.Contains(t, desp, "**延迟**：4200 ms")
	require.Contains(t, desp, "Sol 0.980/0.659")
}

func TestBuildGroupStatusNotifyMessage_AstraSoftMismatch(t *testing.T) {
	group := &Group{ID: 9, Name: "Astra 高速", Platform: PlatformOpenAI}
	event := &GroupStatusEvent{
		EventType:  GroupStatusEventAstraMismatch,
		ToStatus:   AstraCheckStatusMismatch,
		SubStatus:  "closest_gpt-5.6-luna",
		ObservedAt: time.Now(),
	}
	title, _ := buildGroupStatusNotifyMessage("My Gateway", group, event)
	require.Equal(t, "[My Gateway] 分组「Astra 高速」Astra 指纹疑似非 Astra（最接近 Luna，Astra 未达自身阈值）", title)
}

func TestBuildGroupStatusNotifyMessage_AstraRecovered(t *testing.T) {
	group := &Group{ID: 9, Name: "Astra 高速", Platform: PlatformOpenAI}
	event := &GroupStatusEvent{
		EventType:  GroupStatusEventAstraRecovered,
		FromStatus: AstraCheckStatusMismatch,
		ToStatus:   AstraCheckStatusPass,
		SubStatus:  "winner_gpt-6-astra",
		ObservedAt: time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("", group, event)

	require.Equal(t, "[Sub2API] 分组「Astra 高速」Astra 指纹验证已恢复", title)
	require.Contains(t, desp, "**状态**：Astra 指纹不符 → Astra 指纹正常")
}

func TestIsGroupStatusNotifyEvent_IncludesAstraEvents(t *testing.T) {
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventAstraMismatch))
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventAstraRecovered))
	// Sol Juice 与存活事件保持不变
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventSolJuiceMismatch))
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventDown))
	require.False(t, isGroupStatusNotifyEvent("something_else"))
}
