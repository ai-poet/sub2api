# ModelTrace 指纹库来源说明

- 文件：`unified_bank.json`（按字节原样嵌入，`.gitattributes` 设为 `-text`）
- sha256：`1c2cb74d372f9f0f30d0dabbb7b7a838660d2f769a88d0c8489e4c662e088c21`
- 构建时间（`built_at`）：2026-09-23T06:17:08Z
- 来源：[xqy2006/ModelTrace](https://github.com/xqy2006/ModelTrace) `data/unified_bank.json`
- 作者：xqy2006
- 许可证：MIT（全文见下）
- 内容：16 个模型的「逐项凭第一反应输出 1..355 整数」数字分布指纹，包括 Hellinger 与有序分块特征的均值、尺度、干扰方向基、模型中心，以及 1/2/3 条回答的校准温度。
  - GPT（采集自官方订阅 Codex）：gpt-5.4、gpt-5.5、gpt-5.6-sol、gpt-5.6-terra、gpt-5.6-luna、gpt-6-astra、gpt-6-sol、gpt-6-luna
  - Claude（采集自 OAIPro API）：claude-haiku-4-5-20251001、claude-sonnet-4-6、claude-sonnet-5、claude-opus-4-6、claude-opus-4-7、claude-opus-4-8、claude-opus-5、claude-opus-5-5

## 本仓库取用的部分

- 本目录的指纹库数据。
- 挑战提示词的素材池：开场、动作、结尾、分隔提示四组措辞，与 ModelTrace 网页版（https://xqy2006.github.io/ModelTrace/ 的 `challenge-browser.js`）逐字一致。这些措辞原样沿用，因为指纹只在与建库时相同的提示下有效。
- `backend/internal/service/testdata/modeltrace/` 里的少量参考回答，以及用 ModelTrace 自带的 JS 评分器生成的期望概率。它们只用于数值对齐测试。

本仓库把它用于指纹验证里的 **Claude Opus 5.5**（`claude-opus-5-5`）、**Claude Opus 5**（`claude-opus-5`）与 **GPT-6.1 Sol** 目标。GPT-6.1 Sol 不在库中，按数字指纹几乎相同的 `gpt-6-astra` 判定（归因为 Astra 即算一致）。Claude Fable 5.1 不在库中，仍用 meow 基准；GPT-5.6 Sol 用 Juice 读数，GPT-6 Sol / Astra 用 meow 基准。

评分引擎、判定规则与探测执行由本仓库用 Go 实现（`backend/internal/service/group_status_modeltrace*.go`，稳定结论与推送复用指纹验证的按模型状态机），以 ModelTrace 的 `static/fingerprint-core.js` 为规格，并由 golden 测试逐位对齐。

## 结果的含义

这是在库内候选之间、按均匀先验计算的闭集归因概率。库外模型也会被归到最相似的已收录候选。所以「指纹不符」表示「强烈像另一个已收录模型」，是参考信号，不能当作确证。

## 更新指纹库

1. 用上游新版 `data/unified_bank.json` 按字节覆盖本目录的文件，并更新上面的 sha256 与模型列表。
2. 重新生成对齐测试数据：`node backend/internal/service/testdata/modeltrace/gen_golden.mjs <ModelTrace 目录> > backend/internal/service/testdata/modeltrace/reference_cases.json`
3. 若新库不再包含某个 ModelTrace 目标（`astraCheckTargets` 里 `Method` 为 `modeltrace` 的 `TraceModelID`），测试会失败，需同步调整目标列表。

## MIT License

```
MIT License

Copyright (c) 2026 xqy2006

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
