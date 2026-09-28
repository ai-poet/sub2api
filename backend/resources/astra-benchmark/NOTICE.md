# meow 指纹基准包来源说明

- 来源：[chen-006/meow-llm-detector](https://github.com/chen-006/meow-llm-detector) v4.5.4 随包官方基准（`benchmarks/official/`）
- 作者：chen-006 与贡献者
- 许可证：[PolyForm Noncommercial License 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0)
- 评分版本：`meow-fingerprint-v3-predictive`（每题 Dirichlet 先验质量 1 + 训练计数，`other` 取整轮最接近的单一外部参考源）

| 文件 | sha256 | 候选模型 | 本仓库用于 |
|---|---|---|---|
| `meow-gpt-other-cap98-efficient--4.5.4-predictive.20260924.2.meow.json` | `98f8d12c83100352addf44db15d8b57aa183338a4fb5f83ba30ffdbc06d78612` | gpt-6-astra、gpt-6-sol、gpt-5.6-terra、gpt-6-luna、other | GPT-6 Sol、GPT-6 Astra |
| `meow-claude-other-cap98-efficient--4.5.4-predictive.20260924.1.meow.json` | `df005a15ef72cc805b57fd0caf8c8f6b99496ca37d73c6cc759adaa43158ef8c` | claude-fable-5.1、claude-opus-5.5、claude-sonnet-5、claude-haiku-4.5、other | Claude Opus 5.5、Claude Fable 5.1 |

各包含固定短答题（题面、请求参数）、每题的类别词表与各来源的 Dirichlet 参数（`fitted`）、低 / 中 / 高三档的请求配额与强指向线（`tiers`）、校准与验证记录。

本仓库只原样嵌入这些数据文件（`.gitattributes` 设为 `-text`，按字节保存，sha256 与 meow 的 `benchmarks/index.json` 一致）用于本地统计判定；判定引擎（归一化、Dirichlet-multinomial 证据、nearest-source 聚合、强指向线比较）按 meow 技术报告公布的公式由本仓库自行实现，不包含 meow-llm-detector 的代码。文件按原许可证条款分发，使用者须自行遵守其非商业限制。

结果只在包内候选之间做判定：「强指向其他模型」表示答案分布明显更像另一个候选（或 other 参考源），不等于确证冒充；库外模型同样可能被归到最接近的候选。GPT-5.6 Sol 不走 meow 基准，而是用本仓库自写的 Juice 读数（reasoning=high 一条请求，Sol 回 40），所以不再内置只为它服务的 4.5.3 包。

更新基准包时替换文件、同步 `embed.go` 的 `Files` 与本表，并确认 `astraCheckTargets` 里的目标仍在对应包内（测试会检查）。
