// Package astrabenchmark 内置 meow LLM detector 的 GPT 行为指纹基准包，
// 供分组运行状态的「Astra 指纹验证」使用。数据来源与许可见 NOTICE.md。
package astrabenchmark

import _ "embed"

// FileName 是内置基准包的文件名（含版本号）。
const FileName = "meow-gpt-baseline--4.5.0-rc4.meow.json"

// Package 是原样嵌入的基准包 JSON。
//
//go:embed meow-gpt-baseline--4.5.0-rc4.meow.json
var Package []byte
