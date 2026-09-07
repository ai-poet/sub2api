# Astra 指纹基准包来源说明

- 文件：`meow-gpt-baseline--4.5.0-rc4.meow.json`
- 来源：[chen-006/meow-llm-detector](https://github.com/chen-006/meow-llm-detector) v4.5.1 随包基准 `meow-gpt-baseline` 4.5.0-rc4（`benchmarks/official/`）
- 作者：chen-006 与贡献者
- 许可证：[PolyForm Noncommercial License 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0)
- 内容：GPT-6 Astra / GPT-5.6 Sol / Terra / Luna 四个候选模型在五道固定短答题上的答案分布、题族权重与各档阈值。

本仓库只原样嵌入该数据文件用于本地统计判定；判定引擎（归一化、似然、聚合、阈值比较）为本仓库自行实现，不包含 meow-llm-detector 的代码。文件按原许可证条款分发，使用者须自行遵守其非商业限制。
