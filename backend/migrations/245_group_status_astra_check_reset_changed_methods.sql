-- GPT-5.6 Sol 改用 Juice 读数、Claude Opus 5.5 改用 ModelTrace 后，这两个目标上还留着 meow 基准时代的结果：
-- 稳定结论（含官 key 下 meow 误判出的「不符」）、连续计数与检测时间。旧结果不再代表新方法，而且检测时间会让
-- 新方法要等满一个间隔才首次运行，期间界面一直显示旧的红色。这里只清空这两类旧行的判定字段并把检测时间置空，
-- 让调度器立即按新方法重跑；新方法自己的结果（benchmark_package_id 为 sol-juice / modeltrace-bank）不动。
UPDATE group_status_astra_check_states
SET verdict = '',
    stable_status = '',
    winner_model = '',
    matches = '[]'::jsonb,
    reasons = '[]'::jsonb,
    detail = NULL,
    checked_at = NULL,
    consecutive_mismatch = 0,
    updated_at = NOW()
WHERE (expected_model = 'gpt-5.6-sol' AND benchmark_package_id <> 'sol-juice')
   OR (expected_model = 'claude-opus-5.5' AND benchmark_package_id <> 'modeltrace-bank');
