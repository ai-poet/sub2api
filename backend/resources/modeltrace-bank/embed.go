// Package modeltracebank 内置 ModelTrace 的统一数字指纹库，
// 供分组运行状态的「ModelTrace 指纹验证」使用。数据来源与许可见 NOTICE.md。
package modeltracebank

import _ "embed"

// FileName 是内置指纹库的文件名。
const FileName = "unified_bank.json"

// Bank 是原样嵌入的指纹库 JSON（按字节保存，.gitattributes 设为 -text）。
//
//go:embed unified_bank.json
var Bank []byte
