package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildGroupStatusNotifyMessage_AstraMismatch(t *testing.T) {
	group := &Group{ID: 9, Name: "GPT 高速", Platform: PlatformOpenAI}
	latency := int64(4200)
	event := &GroupStatusEvent{
		EventType:   GroupStatusEventAstraMismatch,
		FromStatus:  "",
		ToStatus:    AstraCheckStatusMismatch,
		SubStatus:   astraCheckEventSubStatus("gpt-6-sol", "gpt-6-astra"),
		LatencyMS:   &latency,
		ErrorDetail: "expected GPT-6 Sol · strongest GPT-6 Astra 97.1%/56.7% · valid 32/32 · benchmark 4.5.4-predictive.20260924.2",
		ObservedAt:  time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("My Gateway", group, event)

	require.Equal(t, "[My Gateway] 分组「GPT 高速」指纹不符：GPT-6 Sol（强指向 GPT-6 Astra）", title)
	require.Contains(t, desp, "**分组**：GPT 高速（#9 / openai）")
	require.Contains(t, desp, "**状态**：未知 → 指纹不符")
	require.Contains(t, desp, "**子状态**：gpt-6-sol:winner_gpt-6-astra")
	require.Contains(t, desp, "**延迟**：4200 ms")
	require.Contains(t, desp, "strongest GPT-6 Astra")
}

func TestBuildGroupStatusNotifyMessage_AstraMismatchToOtherModel(t *testing.T) {
	group := &Group{ID: 9, Name: "Claude", Platform: PlatformAnthropic}
	event := &GroupStatusEvent{
		EventType:  GroupStatusEventAstraMismatch,
		ToStatus:   AstraCheckStatusMismatch,
		SubStatus:  astraCheckEventSubStatus("claude-opus-5.5", AstraCheckOtherModel),
		ObservedAt: time.Now(),
	}
	title, _ := buildGroupStatusNotifyMessage("My Gateway", group, event)
	require.Equal(t, "[My Gateway] 分组「Claude」指纹不符：Claude Opus 5.5（强指向 其他模型）", title)
}

func TestBuildGroupStatusNotifyMessage_AstraRecovered(t *testing.T) {
	group := &Group{ID: 9, Name: "GPT 高速", Platform: PlatformOpenAI}
	event := &GroupStatusEvent{
		EventType:  GroupStatusEventAstraRecovered,
		FromStatus: AstraCheckStatusMismatch,
		ToStatus:   AstraCheckStatusPass,
		SubStatus:  astraCheckEventSubStatus("gpt-5.6-sol", "gpt-5.6-sol"),
		ObservedAt: time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("", group, event)

	require.Equal(t, "[Sub2API] 分组「GPT 高速」指纹验证已恢复：GPT-5.6 Sol", title)
	require.Contains(t, desp, "**状态**：指纹不符 → 指纹一致")
}

func TestIsGroupStatusNotifyEvent_IncludesAstraEvents(t *testing.T) {
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventAstraMismatch))
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventAstraRecovered))
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventDown))
	require.True(t, isGroupStatusNotifyEvent(GroupStatusEventUp))
	// 已下线的 Sol Juice / ModelTrace 事件不再推送
	require.False(t, isGroupStatusNotifyEvent("sol_juice_mismatch"))
	require.False(t, isGroupStatusNotifyEvent("modeltrace_mismatch"))
	require.False(t, isGroupStatusNotifyEvent("something_else"))
}
