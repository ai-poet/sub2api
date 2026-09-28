// Package astrabenchmark 内置 meow LLM detector 的行为指纹基准包（GPT 与 Claude），
// 供分组运行状态的「meow 指纹验证」使用。数据来源与许可见 NOTICE.md。
package astrabenchmark

import "embed"

// FS 原样嵌入全部基准包（.gitattributes 设为 -text，按字节保存）。
//
//go:embed *.meow.json
var FS embed.FS

// Files 是内置基准包的文件名，按加载顺序排列。
var Files = []string{
	"meow-gpt-other-cap98-efficient--4.5.4-predictive.20260924.2.meow.json",
	"meow-gpt-other-cap98--4.5.3-predictive.2.meow.json",
	"meow-claude-other-cap98-efficient--4.5.4-predictive.20260924.1.meow.json",
}
