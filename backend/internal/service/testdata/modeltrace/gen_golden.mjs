// 生成 ModelTrace 评分对齐测试数据（开发期工具，不参与构建）。
//
// 用 ModelTrace（MIT）自带的 JS 评分器对若干参考回答算出期望结果，Go 实现必须逐位对齐：
//
//   node backend/internal/service/testdata/modeltrace/gen_golden.mjs <ModelTrace 目录> \
//     > backend/internal/service/testdata/modeltrace/reference_cases.json
//
// 评分器与参考回答都在运行时从 ModelTrace 目录读取，不拷进本仓库；输出里只保留用到的回答原文。
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const root = path.resolve(process.argv[2] || "ModelTrace-main");
const scorerPath = path.join(root, "static", "fingerprint-core.js");
const bankPath = path.join(root, "data", "unified_bank.json");
const core = await import(pathToFileURL(scorerPath).href);
const bankBytes = fs.readFileSync(bankPath);
const bank = JSON.parse(bankBytes);
const sha256 = (bytes) => crypto.createHash("sha256").update(bytes).digest("hex");

const rows = ["gpt_reference.jsonl", "claude_reference.jsonl"]
  .flatMap((file) => fs.readFileSync(path.join(root, "data", file), "utf8").split("\n"))
  .filter(Boolean)
  .map((line) => JSON.parse(line))
  .filter((row) => row.strict_valid);

function pick(source, condition, count, offset = 0) {
  const selected = rows
    .filter((row) => row.source === source && row.condition_id === condition)
    .sort((a, b) => a.challenge_id.localeCompare(b.challenge_id))
    .slice(offset, offset + count);
  if (selected.length !== count) throw new Error(`not enough rows for ${source} ${condition}`);
  return selected.map((row) => ({ text: row.text, expected_count: row.requested_count }));
}

function analyze(name, expected, outputs, verdict) {
  const result = core.analyzeGlobalOutputs(outputs, bank);
  return {
    name,
    expected_model: expected,
    expected_verdict: verdict,
    outputs,
    prediction: result.prediction,
    used_outputs: result.used_outputs,
    calibration_queries: Number(result.calibration.queries),
    beta: result.calibration.beta,
    diagnostics: result.diagnostics,
    results: result.results.map((item) => ({
      model: item.model,
      probability: item.probability,
      score: item.score,
      conditional_probability: item.conditional_probability,
    })),
    family_probabilities: result.family_probabilities.map((item) => ({
      family: item.family,
      probability: item.probability,
    })),
  };
}

const cases = [];
// 每个库内模型：3 条中文无前缀环境（与本仓库挑战提示最接近）的参考回答，应判为自身
for (const model of bank.models) {
  cases.push(analyze(`self-${model.id}`, model.id, pick(model.id, "environment-04", 3), "match"));
}
// 1 条（JSON 数组风格）与 2 条（用户前缀环境）回答，覆盖不同的校准温度
cases.push(analyze("single-output-gpt-5.6-sol", "gpt-5.6-sol", pick("gpt-5.6-sol", "environment-01", 1), "inconclusive"));
cases.push(analyze("two-outputs-claude-opus-5-5", "claude-opus-5-5", pick("claude-opus-5-5", "environment-10", 2), "match"));
// 一条回答被截断到最少数字以下：只计入另外两条
{
  const outputs = pick("gpt-6-sol", "environment-05", 3);
  const numbers = core.parseNumbers(outputs[2].text).slice(0, 60);
  outputs[2] = { text: numbers.join(", "), expected_count: outputs[2].expected_count };
  cases.push(analyze("truncated-output-gpt-6-sol", "gpt-6-sol", outputs, "match"));
}
// 混合回答，得到不那么极端的概率
cases.push(analyze("mixed-sol-terra", "gpt-5.6-sol", [
  ...pick("gpt-5.6-sol", "environment-06", 2),
  ...pick("gpt-5.6-terra", "environment-06", 1),
], ""));
// 预期 Sol、实际 Terra：应判为不符
cases.push(analyze("terra-claimed-as-sol", "gpt-5.6-sol", pick("gpt-5.6-terra", "environment-04", 3), "mismatch"));
// 预期 Opus 5.5、实际 GPT-6 Astra：跨家族，应判为不符
cases.push(analyze("astra-claimed-as-opus-5-5", "claude-opus-5-5", pick("gpt-6-astra", "environment-04", 3), "mismatch"));

const parserInputs = [
  "12, 7, 355, 356, 0, 1",
  "```json\n[3, 44, 5]\n```",
  "前言 1 2 3 然后 4 5 6 7 结束 8",
  "1、2、3、4",
  "abc 10 20 def 30",
  "１２, 13, 14",
  "0007, 99999999999999999999, 5",
  "",
];
const parserCases = parserInputs.map((text) => ({ text, numbers: core.parseNumbers(text) }));

process.stdout.write(JSON.stringify({
  generator: "gen_golden.mjs",
  bank_sha256: sha256(bankBytes),
  scorer_sha256: sha256(fs.readFileSync(scorerPath)),
  cases,
  parser_cases: parserCases,
}, null, 1) + "\n");
