<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="group" class="space-y-6">
      <div
        class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/80 md:flex-row md:items-center md:justify-between"
      >
        <div>
          <div class="text-base font-semibold text-gray-900 dark:text-white">{{ group.name }}</div>
          <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t(`admin.groups.platforms.${group.platform}`) }}
            <span v-if="group.description" class="ml-2">{{ group.description }}</span>
          </div>
        </div>
        <span :class="['badge', summaryBadgeClass]">{{ summaryStatusText }}</span>
      </div>

      <div
        v-if="loading"
        class="flex items-center justify-center py-10"
      >
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else class="space-y-6">
        <div v-if="loadError" class="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-300">
          {{ loadError }}
        </div>

        <div class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-4 dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.enabled') }}
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.runtimeStatus.enabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-4 dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.notifyEnabled') }}
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.runtimeStatus.notifyEnabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.notify_enabled" />
        </div>

        <div class="grid gap-5 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.probeModel') }}</label>
            <input
              v-model.trim="form.probe_model"
              type="text"
              class="input"
              :placeholder="t('admin.groups.runtimeStatus.probeModelPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.validationMode') }}</label>
            <select v-model="form.validation_mode" class="input">
              <option
                v-for="option in validationModeOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.groups.runtimeStatus.probePrompt') }}</label>
          <textarea
            v-model="form.probe_prompt"
            rows="4"
            class="input"
            :placeholder="t('admin.groups.runtimeStatus.probePromptPlaceholder')"
          ></textarea>
        </div>

        <div v-if="showKeywordEditor">
          <label class="input-label">{{ t('admin.groups.runtimeStatus.expectedKeywords') }}</label>
          <textarea
            v-model="expectedKeywordsText"
            rows="4"
            class="input"
            :placeholder="t('admin.groups.runtimeStatus.expectedKeywordsPlaceholder')"
          ></textarea>
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.groups.runtimeStatus.expectedKeywordsHint') }}
          </p>
        </div>

        <div class="grid gap-5 md:grid-cols-3">
          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.intervalSeconds') }}</label>
            <input
              v-model.number="form.interval_seconds"
              type="number"
              min="10"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.timeoutSeconds') }}</label>
            <input
              v-model.number="form.timeout_seconds"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.slowLatencyMs') }}</label>
            <input
              v-model.number="form.slow_latency_ms"
              type="number"
              min="100"
              step="100"
              class="input"
            />
          </div>
        </div>

        <div
          v-if="modelTraceSupported"
          class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
        >
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.modelTrace.title') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.modelTrace.hint') }}
              </p>
              <p v-if="summary.modeltrace_bank_sha256" class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                {{ t('admin.groups.runtimeStatus.modelTrace.bank') }}: {{ summary.modeltrace_bank_sha256 }}
                <template v-if="summary.modeltrace_bank_built_at">
                  · {{ formatDateTime(summary.modeltrace_bank_built_at) }}
                </template>
              </p>
            </div>
            <Toggle v-model="form.modeltrace_enabled" />
          </div>

          <div class="grid gap-5 md:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.modelTrace.expectedModel') }}</label>
              <select v-model="form.modeltrace_expected_model" class="input">
                <option v-for="target in modelTraceTargets" :key="target.id" :value="target.id">
                  {{ target.display_name }}
                </option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.modelTrace.requestModel') }}</label>
              <input
                v-model.trim="form.modeltrace_request_model"
                type="text"
                class="input"
                :placeholder="form.modeltrace_expected_model"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.modelTrace.intervalSeconds') }}</label>
              <input
                v-model.number="form.modeltrace_interval_seconds"
                type="number"
                min="900"
                step="300"
                class="input"
              />
            </div>
          </div>

          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.modelTrace.latestResult') }}
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing || modelTraceProbing || summary.modeltrace_running"
              @click="handleModelTraceProbe"
            >
              <span
                v-if="modelTraceProbing || summary.modeltrace_running"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{
                modelTraceProbing || summary.modeltrace_running
                  ? t('admin.groups.runtimeStatus.modelTrace.running')
                  : t('admin.groups.runtimeStatus.modelTrace.probeNow')
              }}
            </button>
          </div>

          <div
            v-if="modelTraceProgress"
            class="space-y-3 rounded-lg border border-sky-200 bg-sky-50/40 px-3 py-3 dark:border-sky-900/40 dark:bg-sky-950/20"
          >
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.modelTrace.progress.title', { round: modelTraceProgress.round }) }}
                <span class="ml-2 text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ t(`admin.groups.runtimeStatus.modelTrace.progress.phases.${modelTraceProgress.phase}`) }}
                  <template v-if="modelTraceProgress.account_id">
                    · {{ t('admin.groups.runtimeStatus.modelTrace.progress.account') }} #{{ modelTraceProgress.account_id }}
                  </template>
                  · {{ formatGroupRuntimeLatency(modelTraceProgress.elapsed_ms) }}
                </span>
              </div>
              <div class="text-xs text-gray-600 dark:text-gray-300">
                {{ t('admin.groups.runtimeStatus.modelTrace.progress.accepted') }} {{ modelTraceProgress.accepted }} / {{ modelTraceProgress.target }}
                · {{ t('admin.groups.runtimeStatus.modelTrace.progress.used') }} {{ modelTraceProgress.used }} / {{ modelTraceProgress.planned }}
                · {{ t('admin.groups.runtimeStatus.modelTrace.progress.rejected') }} {{ modelTraceProgress.rejected }}
                · {{ t('admin.groups.runtimeStatus.modelTrace.progress.failed') }} {{ modelTraceProgress.failed }}
                <template v-if="modelTraceProgress.in_flight">
                  · {{ t('admin.groups.runtimeStatus.modelTrace.progress.inFlight') }} {{ modelTraceProgress.in_flight }}
                </template>
              </div>
            </div>
            <div class="h-2 w-full rounded bg-gray-200 dark:bg-dark-700">
              <div class="h-2 rounded bg-sky-500 transition-all" :style="{ width: `${modelTraceProgressPercent}%` }"></div>
            </div>
            <div v-if="(modelTraceProgress.attempts?.length ?? 0) > 0" class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
              <div v-for="attempt in modelTraceProgress.attempts" :key="attempt.seq">
                #{{ attempt.seq }} ·
                <span :class="modelTraceOutcomeClass(attempt.outcome)">
                  {{ t(`admin.groups.runtimeStatus.modelTrace.outcomes.${attempt.outcome}`) }}
                </span>
                · {{ attempt.parsed_numbers }} / {{ attempt.expected_count }}
                <template v-if="attempt.rejection"> · {{ modelTraceRejectionLabel(attempt.rejection) }}</template>
                <template v-if="attempt.http_code && attempt.http_code !== 200"> · HTTP {{ attempt.http_code }}</template>
                · {{ formatGroupRuntimeLatency(attempt.latency_ms) }}
              </div>
            </div>
          </div>

          <div v-if="summary.modeltrace_checked_at" class="space-y-3">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.verdict') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', getModelTraceBadgeClass(modelTraceDisplayStatus)]">
                    {{ modelTraceStatusText }}
                  </span>
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.topCandidate') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.modeltrace_top_model ? modelTraceModelLabel(summary.modeltrace_top_model) : '-' }}
                  <span v-if="summary.modeltrace_top_model" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                    {{ formatMatchPercent(summary.modeltrace_top_probability) }}
                  </span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.expectedProbability', {
                    model: modelTraceModelLabel(summary.modeltrace_run_expected_model || summary.modeltrace_expected_model),
                    probability: formatMatchPercent(summary.modeltrace_expected_probability)
                  }) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.modeltrace_checked_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.modeltrace_checked_at) }}
                  · {{ t('admin.groups.runtimeStatus.modelTrace.validOutputs', { valid: summary.modeltrace_valid_outputs }) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.tokens') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.modeltrace_input_tokens }} / {{ summary.modeltrace_output_tokens }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.modelTrace.lastCost') }}: {{ formatUsd(summary.modeltrace_last_cost_usd) }}
                  · {{ t('admin.groups.runtimeStatus.modelTrace.monthlyEstimate') }}: {{ formatUsd(modelTraceMonthlyCost, 2) }}
                </div>
              </div>
            </div>

            <div
              v-if="(summary.modeltrace_ranking?.length ?? 0) > 0"
              class="space-y-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.modelTrace.ranking') }}
              </div>
              <div v-for="entry in summary.modeltrace_ranking" :key="entry.model">
                <div class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-200">
                  <span :class="entry.model === modelTraceRankingExpected ? 'font-semibold' : ''">
                    {{ modelTraceModelLabel(entry.model) }}
                    <span class="text-gray-400">· {{ entry.family_name }}</span>
                  </span>
                  <span>{{ formatMatchPercent(entry.probability) }}</span>
                </div>
                <div class="mt-1 h-2 w-full rounded bg-gray-200 dark:bg-dark-700">
                  <div
                    class="h-2 rounded"
                    :class="entry.model === modelTraceRankingExpected ? 'bg-emerald-500' : 'bg-rose-500'"
                    :style="{ width: `${Math.min(100, Math.max(0.5, entry.probability * 100))}%` }"
                  ></div>
                </div>
              </div>
            </div>

            <details
              v-if="modelTraceLastRun && (modelTraceLastRun.outputs?.length ?? 0) > 0"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.modelTrace.outputTable.title', { count: modelTraceLastRun.outputs.length }) }}
                <span class="ml-1 font-normal">
                  · {{ t('admin.groups.runtimeStatus.modelTrace.outputTable.runMeta', {
                    valid: modelTraceLastRun.valid_outputs,
                    made: modelTraceLastRun.attempts_made,
                    planned: modelTraceLastRun.attempts_planned,
                    latency: formatGroupRuntimeLatency(modelTraceLastRun.latency_ms)
                  }) }}
                  <template v-if="modelTraceLastRun.account_id">
                    · {{ t('admin.groups.runtimeStatus.modelTrace.progress.account') }} #{{ modelTraceLastRun.account_id }}
                    ({{ modelTraceLastRun.account_type }})
                  </template>
                </span>
              </summary>
              <div class="mt-2 max-h-72 overflow-auto">
                <table class="w-full text-left text-xs">
                  <thead class="text-gray-500 dark:text-gray-400">
                    <tr>
                      <th class="pr-2 font-medium">#</th>
                      <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.modelTrace.outputTable.numbers') }}</th>
                      <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.modelTrace.outputTable.result') }}</th>
                      <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.modelTrace.outputTable.singleTop') }}</th>
                      <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.modelTrace.outputTable.excerpt') }}</th>
                      <th class="font-medium">{{ t('admin.groups.runtimeStatus.modelTrace.outputTable.latency') }}</th>
                    </tr>
                  </thead>
                  <tbody class="text-gray-700 dark:text-gray-200">
                    <tr
                      v-for="(row, index) in modelTraceLastRun.outputs"
                      :key="`${row.seq}-${index}`"
                      class="border-t border-gray-100 dark:border-dark-700"
                    >
                      <td class="py-1 pr-2 text-gray-400">{{ row.seq }}</td>
                      <td class="py-1 pr-2">{{ row.parsed_numbers }} / {{ row.expected_count }}</td>
                      <td class="py-1 pr-2" :class="modelTraceOutcomeClass(modelTraceRowOutcome(row))">
                        {{
                          row.accepted
                            ? t('admin.groups.runtimeStatus.modelTrace.outcomes.accepted')
                            : row.rejection
                              ? modelTraceRejectionLabel(row.rejection)
                              : t('admin.groups.runtimeStatus.modelTrace.outcomes.failed')
                        }}
                        <template v-if="row.http_code && row.http_code !== 200"> · HTTP {{ row.http_code }}</template>
                      </td>
                      <td class="py-1 pr-2">
                        <template v-if="row.top_model">
                          {{ modelTraceModelLabel(row.top_model) }} {{ formatMatchPercent(row.top_probability) }}
                        </template>
                        <template v-else>-</template>
                      </td>
                      <td class="max-w-[16rem] truncate py-1 pr-2" :title="row.error || row.excerpt">
                        {{ row.error || row.excerpt || '-' }}
                      </td>
                      <td class="py-1">{{ formatGroupRuntimeLatency(row.latency_ms) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </details>

            <div v-if="(summary.modeltrace_reasons?.length ?? 0) > 0" class="flex flex-wrap gap-2">
              <span v-for="reason in summary.modeltrace_reasons" :key="reason" class="badge badge-warning">
                {{ modelTraceReasonLabel(reason) }}
              </span>
            </div>

            <div
              v-if="summary.modeltrace_detail"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.modelTrace.detail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.modeltrace_detail }}</pre>
            </div>
          </div>
          <div
            v-else
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.modelTrace.latestResultEmpty') }}
          </div>

          <p class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.groups.runtimeStatus.modelTrace.disclaimer') }}
          </p>
        </div>

        <div
          v-if="group.platform === 'openai'"
          class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
        >
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.astraCheck.title') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.hint') }}
              </p>
              <p v-if="summary.astra_check_benchmark_version" class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                {{ t('admin.groups.runtimeStatus.astraCheck.benchmark') }}: {{ summary.astra_check_benchmark_version }}
                · {{ (summary.astra_check_benchmark_models ?? []).map((m) => astraModelShortName(m.id)).join(' / ') }}
              </p>
            </div>
            <Toggle v-model="form.astra_check_enabled" />
          </div>

          <div class="grid gap-5 md:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.astraCheck.requestModel') }}</label>
              <input
                v-model.trim="form.astra_check_request_model"
                type="text"
                class="input"
                placeholder="gpt-6-astra"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.astraCheck.tier') }}</label>
              <select v-model="form.astra_check_tier" class="input">
                <option v-for="option in astraTierOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.astraCheck.intervalSeconds') }}</label>
              <input
                v-model.number="form.astra_check_interval_seconds"
                type="number"
                min="900"
                step="300"
                class="input"
              />
            </div>
          </div>

          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.astraCheck.latestResult') }}
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing || astraProbing || summary.astra_check_running"
              @click="handleAstraCheckProbe"
            >
              <span
                v-if="astraProbing || summary.astra_check_running"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{
                astraProbing || summary.astra_check_running
                  ? t('admin.groups.runtimeStatus.astraCheck.running')
                  : t('admin.groups.runtimeStatus.astraCheck.probeNow')
              }}
            </button>
          </div>

          <div
            v-if="astraProgress"
            class="space-y-3 rounded-lg border border-emerald-200 bg-emerald-50/40 px-3 py-3 dark:border-emerald-900/40 dark:bg-emerald-950/20"
          >
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.astraCheck.progress.title', { round: astraProgress.round }) }}
                <span class="ml-2 text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ t(`admin.groups.runtimeStatus.astraCheck.progress.phases.${astraProgress.phase}`) }}
                  <template v-if="astraProgress.account_id">
                    · {{ t('admin.groups.runtimeStatus.astraCheck.progress.account') }} #{{ astraProgress.account_id }}
                  </template>
                  · {{ formatGroupRuntimeLatency(astraProgress.elapsed_ms) }}
                </span>
              </div>
              <div class="text-xs text-gray-600 dark:text-gray-300">
                {{ astraProgress.completed }} / {{ astraProgress.planned }}
                · {{ t('admin.groups.runtimeStatus.astraCheck.progress.valid') }} {{ astraProgress.valid }}
                · {{ t('admin.groups.runtimeStatus.astraCheck.progress.invalid') }} {{ astraProgress.invalid }}
                · {{ t('admin.groups.runtimeStatus.astraCheck.progress.failed') }} {{ astraProgress.failed }}
                · {{ t('admin.groups.runtimeStatus.astraCheck.progress.requests') }} {{ astraProgress.requests }}
                <template v-if="astraProgress.in_flight">
                  · {{ t('admin.groups.runtimeStatus.astraCheck.progress.inFlight') }} {{ astraProgress.in_flight }}
                </template>
              </div>
            </div>
            <div class="h-2 w-full rounded bg-gray-200 dark:bg-dark-700">
              <div class="h-2 rounded bg-emerald-500 transition-all" :style="{ width: `${astraProgressPercent}%` }"></div>
            </div>
            <div v-if="(astraProgress.samples?.length ?? 0) > 0" class="max-h-56 overflow-auto">
              <table class="w-full text-left text-xs">
                <thead class="text-gray-500 dark:text-gray-400">
                  <tr>
                    <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.seq') }}</th>
                    <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.cell') }}</th>
                    <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.attempt') }}</th>
                    <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.answer') }}</th>
                    <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.category') }}</th>
                    <th class="pr-2 font-medium">HTTP</th>
                    <th class="font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.latency') }}</th>
                  </tr>
                </thead>
                <tbody class="text-gray-700 dark:text-gray-200">
                  <tr v-for="row in astraSampleRows(astraProgress.samples ?? [])" :key="row.seq" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1 pr-2 text-gray-400">{{ row.seq }}</td>
                    <td class="py-1 pr-2 font-mono">{{ row.cell_id }}</td>
                    <td class="py-1 pr-2">{{ row.attempt }}</td>
                    <td class="max-w-[16rem] truncate py-1 pr-2" :title="row.error || row.answer">{{ row.answer || row.error || '-' }}</td>
                    <td class="py-1 pr-2" :class="astraSampleOutcomeClass(row.outcome)">{{ row.category || row.outcome }}</td>
                    <td class="py-1 pr-2">{{ row.http_code ?? '-' }}</td>
                    <td class="py-1">{{ formatGroupRuntimeLatency(row.latency_ms) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <div v-if="summary.astra_check_checked_at" class="space-y-3">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.verdict') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', getAstraCheckBadgeClass(astraDisplayStatus)]">
                    {{ astraStatusText }}
                  </span>
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.samples') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.astra_check_valid_samples }} / {{ summary.astra_check_planned_samples }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.astra_check_checked_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.astra_check_checked_at) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.tokens') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.astra_check_input_tokens }} / {{ summary.astra_check_output_tokens }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.lastCost') }}: {{ formatUsd(summary.astra_check_last_cost_usd) }}
                  · {{ t('admin.groups.runtimeStatus.astraCheck.monthlyEstimate') }}: {{ formatUsd(astraMonthlyCost, 2) }}
                </div>
              </div>
            </div>

            <div class="space-y-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.matches') }}
              </div>
              <div v-for="m in summary.astra_check_matches" :key="m.model">
                <div class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-200">
                  <span :class="m.passed ? 'font-semibold' : ''">{{ astraModelShortName(m.model) }}</span>
                  <span>
                    {{ formatMatchPercent(m.match) }}
                    <span class="text-gray-400">/ {{ t('admin.groups.runtimeStatus.astraCheck.threshold') }} {{ formatMatchPercent(m.threshold) }}</span>
                  </span>
                </div>
                <div class="mt-1 h-2 w-full rounded bg-gray-200 dark:bg-dark-700">
                  <div
                    class="h-2 rounded"
                    :class="m.passed ? (m.model === 'gpt-6-astra' ? 'bg-emerald-500' : 'bg-rose-500') : 'bg-gray-400'"
                    :style="{ width: `${Math.min(100, Math.round(m.match * 100))}%` }"
                  ></div>
                </div>
              </div>
            </div>

            <details
              v-if="astraLastRun && ((astraLastRun.samples?.length ?? 0) > 0 || (astraLastRun.cells?.length ?? 0) > 0)"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.title', { count: astraLastRun.samples?.length ?? 0 }) }}
                <span class="ml-1 font-normal">
                  · {{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.runMeta', {
                    planned: astraLastRun.requests_planned,
                    completed: astraLastRun.requests_completed,
                    latency: formatGroupRuntimeLatency(astraLastRun.latency_ms)
                  }) }}
                </span>
              </summary>
              <div class="mt-2 space-y-2">
                <div
                  v-for="cell in astraLastRun.cells"
                  :key="cell.cell_id"
                  class="text-xs text-gray-600 dark:text-gray-300"
                >
                  <span class="font-mono font-medium">{{ cell.cell_id }}</span>: {{ formatAstraCellCounts(cell) }}
                </div>
                <div v-if="(astraLastRun.samples?.length ?? 0) > 0" class="max-h-72 overflow-auto">
                  <table class="w-full text-left text-xs">
                    <thead class="text-gray-500 dark:text-gray-400">
                      <tr>
                        <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.seq') }}</th>
                        <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.cell') }}</th>
                        <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.attempt') }}</th>
                        <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.answer') }}</th>
                        <th class="pr-2 font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.category') }}</th>
                        <th class="pr-2 font-medium">HTTP</th>
                        <th class="font-medium">{{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.latency') }}</th>
                      </tr>
                    </thead>
                    <tbody class="text-gray-700 dark:text-gray-200">
                      <tr v-for="row in astraLastRun.samples" :key="row.seq" class="border-t border-gray-100 dark:border-dark-700">
                        <td class="py-1 pr-2 text-gray-400">{{ row.seq }}</td>
                        <td class="py-1 pr-2 font-mono">{{ row.cell_id }}</td>
                        <td class="py-1 pr-2">{{ row.attempt }}</td>
                        <td class="max-w-[16rem] truncate py-1 pr-2" :title="row.error || row.answer">{{ row.answer || row.error || '-' }}</td>
                        <td class="py-1 pr-2" :class="astraSampleOutcomeClass(row.outcome)">{{ row.category || row.outcome }}</td>
                        <td class="py-1 pr-2">{{ row.http_code ?? '-' }}</td>
                        <td class="py-1">{{ formatGroupRuntimeLatency(row.latency_ms) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </details>

            <div
              v-if="(summary.astra_check_reasons?.length ?? 0) > 0"
              class="flex flex-wrap gap-2"
            >
              <span
                v-for="reason in summary.astra_check_reasons"
                :key="reason"
                class="badge badge-warning"
              >
                {{ astraReasonLabel(reason) }}
              </span>
            </div>

            <div
              v-if="summary.astra_check_detail"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.detail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.astra_check_detail }}</pre>
            </div>
          </div>
          <div
            v-else
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.astraCheck.latestResultEmpty') }}
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.latestResult') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.latestResultHint') }}
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing"
              @click="handleProbe"
            >
              <span
                v-if="probing"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{ probing ? t('admin.groups.runtimeStatus.probing') : t('admin.groups.runtimeStatus.probeNow') }}
            </button>
          </div>

          <div v-if="summary.observed_at" class="mt-4 space-y-4">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.currentStatus') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', summaryBadgeClass]">{{ summaryStatusText }}</span>
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.observed_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.observed_at) }}
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.latency') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatGroupRuntimeLatency(summary.latency_ms) }}
                </div>
                <div
                  v-if="summary.total_latency_ms !== null && summary.total_latency_ms !== undefined"
                  class="mt-1 text-xs text-gray-500 dark:text-gray-400"
                >
                  {{ t('admin.groups.runtimeStatus.totalLatency') }}: {{ formatGroupRuntimeLatency(summary.total_latency_ms) }}
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.httpCode') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.http_code ?? '-' }}
                </div>
              </div>
            </div>

            <div
              v-if="summary.sub_status"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"
            >
              <span class="font-medium">{{ t('admin.groups.runtimeStatus.subStatus') }}:</span>
              <span class="ml-2">{{ summary.sub_status }}</span>
            </div>

            <div
              v-if="summary.response_excerpt"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.responseExcerpt') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.response_excerpt }}</pre>
            </div>

            <div
              v-if="summary.error_detail"
              class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-3 dark:border-rose-900/40 dark:bg-rose-950/20"
            >
              <div class="text-xs font-medium text-rose-700 dark:text-rose-300">
                {{ t('admin.groups.runtimeStatus.errorDetail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-rose-700 dark:text-rose-300">{{ summary.error_detail }}</pre>
            </div>
          </div>

          <div
            v-else
            class="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.latestResultEmpty') }}
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full items-center justify-between gap-3">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.runtimeStatus.footerHint') }}
        </div>
        <div class="flex items-center gap-3">
          <button type="button" class="btn btn-secondary" :disabled="saving || probing" @click="emit('close')">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="loading || saving || probing || !group" @click="handleSave">
            <span
              v-if="saving"
              class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
            ></span>
            {{ saving ? t('common.saving') : t('admin.groups.runtimeStatus.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores'
import type {
  AdminGroup,
  AstraCheckCellSummary,
  AstraCheckLastRun,
  AstraCheckProgress,
  AstraCheckSampleRecord,
  AstraCheckTier,
  GroupStatusAdminView,
  GroupStatusSummary,
  GroupStatusValidationMode,
  ModelTraceLastRun,
  ModelTraceOutputRecord,
  ModelTraceProgress,
  ModelTraceTarget,
} from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import {
  astraMismatchTextKey,
  astraModelShortName,
  estimateMonthlyProbeCostUsd,
  formatGroupRuntimeLatency,
  formatMatchPercent,
  formatUsd,
  getAstraCheckBadgeClass,
  getGroupRuntimeStatusBadgeClass,
  getModelTraceBadgeClass,
  joinRuntimeKeywordsText,
  modelTraceModelLabel,
  normalizeAstraCheckStatus,
  normalizeModelTraceStatus,
  normalizeGroupRuntimeStatus,
  shouldShowRuntimeKeywordEditor,
  splitRuntimeKeywordsText,
} from '@/utils/groupStatus'

interface Props {
  show: boolean
  group: AdminGroup | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'updated'): void
}>()

const { t, te } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const probing = ref(false)
const loadError = ref('')
const expectedKeywordsText = ref('')
const currentView = ref<GroupStatusAdminView | null>(null)

const form = reactive({
  enabled: false,
  probe_model: '',
  probe_prompt: '',
  validation_mode: 'non_empty' as GroupStatusValidationMode,
  interval_seconds: 60,
  timeout_seconds: 30,
  slow_latency_ms: 15000,
  notify_enabled: true,
  modeltrace_enabled: false,
  modeltrace_expected_model: '',
  modeltrace_request_model: '',
  modeltrace_interval_seconds: 3600,
  astra_check_enabled: false,
  astra_check_request_model: 'gpt-6-astra',
  astra_check_tier: 'low' as AstraCheckTier,
  astra_check_interval_seconds: 3600,
})

const modelTraceProbing = ref(false)
let modelTracePollTimer: ReturnType<typeof setTimeout> | null = null
let modelTracePollStartedAt = 0
const MODEL_TRACE_POLL_INTERVAL_MS = 3000
const MODEL_TRACE_POLL_MAX_MS = 20 * 60 * 1000
const astraProbing = ref(false)
let astraPollTimer: ReturnType<typeof setTimeout> | null = null
let astraPollStartedAt = 0
const ASTRA_POLL_INTERVAL_MS = 2000
const ASTRA_POLL_MAX_MS = 20 * 60 * 1000

const dialogTitle = computed(() => {
  if (!props.group) {
    return t('admin.groups.runtimeStatus.titlePlain')
  }
  return t('admin.groups.runtimeStatus.title', { name: props.group.name })
})

const validationModeOptions = computed(() => [
  {
    value: 'non_empty' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.nonEmpty')
  },
  {
    value: 'keywords_any' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.keywordsAny')
  },
  {
    value: 'keywords_all' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.keywordsAll')
  }
])

const summary = computed<GroupStatusSummary>(() => {
  return currentView.value?.summary ?? {
    group_id: props.group?.id ?? 0,
    config_id: 0,
    enabled: false,
    probe_model: '',
    latest_status: '',
    stable_status: '',
    response_excerpt: '',
    latency_ms: null,
    total_latency_ms: null,
    http_code: null,
    sub_status: '',
    error_detail: '',
    observed_at: null,
    consecutive_down: 0,
    consecutive_non_down: 0,
    modeltrace_enabled: false,
    modeltrace_expected_model: '',
    modeltrace_request_model: '',
    modeltrace_interval_seconds: 3600,
    modeltrace_verdict: '',
    modeltrace_stable_status: '',
    modeltrace_run_expected_model: '',
    modeltrace_top_model: '',
    modeltrace_top_probability: 0,
    modeltrace_expected_probability: null,
    modeltrace_ranking: [],
    modeltrace_reasons: [],
    modeltrace_detail: '',
    modeltrace_checked_at: null,
    modeltrace_consecutive_mismatch: 0,
    modeltrace_valid_outputs: 0,
    modeltrace_input_tokens: 0,
    modeltrace_output_tokens: 0,
    modeltrace_reasoning_tokens: 0,
    modeltrace_last_cost_usd: 0,
    modeltrace_running: false,
    modeltrace_bank_sha256: '',
    modeltrace_bank_built_at: '',
    astra_check_enabled: false,
    astra_check_request_model: 'gpt-6-astra',
    astra_check_tier: 'low',
    astra_check_interval_seconds: 3600,
    astra_check_verdict: '',
    astra_check_stable_status: '',
    astra_check_winner: '',
    astra_check_matches: [],
    astra_check_reasons: [],
    astra_check_detail: '',
    astra_check_checked_at: null,
    astra_check_consecutive_mismatch: 0,
    astra_check_valid_samples: 0,
    astra_check_planned_samples: 0,
    astra_check_input_tokens: 0,
    astra_check_output_tokens: 0,
    astra_check_reasoning_tokens: 0,
    astra_check_last_cost_usd: 0,
    astra_check_running: false,
    astra_check_benchmark_version: '',
    astra_check_benchmark_models: [],
    astra_check_benchmark_tiers: []
  }
})

const astraTierOptions = computed(() => {
  const tiers = summary.value.astra_check_benchmark_tiers ?? []
  return (['low', 'medium', 'high'] as AstraCheckTier[]).map((tier) => {
    const meta = tiers.find((item) => item.tier === tier)
    const requests = meta?.requests ? ` (${meta.requests})` : ''
    return { value: tier, label: `${t(`admin.groups.runtimeStatus.astraCheck.tiers.${tier}`)}${requests}` }
  })
})

const astraDisplayStatus = computed(() =>
  normalizeAstraCheckStatus(summary.value.astra_check_stable_status, summary.value.astra_check_verdict)
)

const astraStatusText = computed(() => {
  const status = astraDisplayStatus.value
  if (status === 'mismatch') {
    const key = astraMismatchTextKey(summary.value.astra_check_winner, summary.value.astra_check_reasons)
    return t(`admin.groups.runtimeStatus.astraCheck.statuses.${key}`, {
      winner: astraModelShortName(summary.value.astra_check_winner)
    })
  }
  return t(`admin.groups.runtimeStatus.astraCheck.statuses.${status}`)
})

const astraMonthlyCost = computed(() =>
  estimateMonthlyProbeCostUsd(summary.value.astra_check_last_cost_usd, Number(form.astra_check_interval_seconds) || 0)
)

function astraReasonLabel(reason: string): string {
  const key = `admin.groups.runtimeStatus.astraCheck.reasons.${reason}`
  return te(key) ? t(key) : reason
}

// 验证进行中的实时进度（仅运行时后端才返回）与最近一次运行的完整记录
const astraProgress = computed<AstraCheckProgress | null>(() => currentView.value?.astra_check_progress ?? null)
const astraLastRun = computed<AstraCheckLastRun | null>(() => currentView.value?.astra_check_last_run ?? null)
const astraProgressPercent = computed(() => {
  const p = astraProgress.value
  if (!p || p.planned <= 0) {
    return 0
  }
  return Math.min(100, Math.round((p.completed / p.planned) * 100))
})

// 进度面板最新的放最上面
function astraSampleRows(samples: AstraCheckSampleRecord[]): AstraCheckSampleRecord[] {
  return [...samples].reverse()
}

function astraSampleOutcomeClass(outcome: string): string {
  switch (outcome) {
    case 'valid':
      return 'text-emerald-600 dark:text-emerald-400'
    case 'invalid':
      return 'text-amber-600 dark:text-amber-400'
    case 'failed':
      return 'text-rose-600 dark:text-rose-400'
    default:
      return ''
  }
}

function formatAstraCellCounts(cell: AstraCheckCellSummary): string {
  const parts = Object.entries(cell.categories || {})
    .sort((a, b) => b[1] - a[1])
    .map(([category, count]) => `${category}×${count}`)
  const counts = parts.length > 0 ? parts.join(' · ') : '-'
  return `${counts}（${t('admin.groups.runtimeStatus.astraCheck.progress.valid')} ${cell.valid}/${cell.planned}）`
}

// ModelTrace 指纹验证：可选预期模型来自后端（按分组平台），结果与进度都从管理视图读
const modelTraceSupported = computed(() => props.group?.platform === 'openai' || props.group?.platform === 'anthropic')
const modelTraceTargets = computed<ModelTraceTarget[]>(() => currentView.value?.modeltrace_targets ?? [])
const modelTraceProgress = computed<ModelTraceProgress | null>(() => currentView.value?.modeltrace_progress ?? null)
const modelTraceLastRun = computed<ModelTraceLastRun | null>(() => currentView.value?.modeltrace_last_run ?? null)

const modelTraceDisplayStatus = computed(() =>
  normalizeModelTraceStatus(
    summary.value.modeltrace_stable_status,
    summary.value.modeltrace_verdict,
    summary.value.modeltrace_run_expected_model,
    form.modeltrace_expected_model
  )
)

// 排名条高亮最近一次运行所对照的预期模型
const modelTraceRankingExpected = computed(
  () => summary.value.modeltrace_run_expected_model || summary.value.modeltrace_expected_model
)

const modelTraceStatusText = computed(() => {
  const status = modelTraceDisplayStatus.value
  const params = {
    expected: modelTraceModelLabel(modelTraceRankingExpected.value),
    top: modelTraceModelLabel(summary.value.modeltrace_top_model)
  }
  return t(`admin.groups.runtimeStatus.modelTrace.statuses.${status}`, params)
})

const modelTraceMonthlyCost = computed(() =>
  estimateMonthlyProbeCostUsd(summary.value.modeltrace_last_cost_usd, Number(form.modeltrace_interval_seconds) || 0)
)

const modelTraceProgressPercent = computed(() => {
  const p = modelTraceProgress.value
  if (!p || p.target <= 0) {
    return 0
  }
  return Math.min(100, Math.round((p.accepted / p.target) * 100))
})

function modelTraceReasonLabel(reason: string): string {
  const key = `admin.groups.runtimeStatus.modelTrace.reasons.${reason}`
  return te(key) ? t(key) : reason
}

function modelTraceRejectionLabel(rejection: string): string {
  const key = `admin.groups.runtimeStatus.modelTrace.rejections.${rejection}`
  return te(key) ? t(key) : rejection
}

function modelTraceRowOutcome(row: ModelTraceOutputRecord): string {
  if (row.accepted) {
    return 'accepted'
  }
  return row.rejection ? 'rejected' : 'failed'
}

function modelTraceOutcomeClass(outcome: string): string {
  switch (outcome) {
    case 'accepted':
      return 'text-emerald-600 dark:text-emerald-400'
    case 'rejected':
      return 'text-amber-600 dark:text-amber-400'
    case 'failed':
      return 'text-rose-600 dark:text-rose-400'
    default:
      return ''
  }
}

const summaryStatus = computed(() => {
  if (!form.enabled) {
    return 'unknown'
  }
  if (summary.value.stable_status) {
    return normalizeGroupRuntimeStatus(summary.value.stable_status)
  }
  if (summary.value.latest_status) {
    return normalizeGroupRuntimeStatus(summary.value.latest_status)
  }
  return 'unknown'
})

const summaryBadgeClass = computed(() => getGroupRuntimeStatusBadgeClass(summaryStatus.value))

const summaryStatusText = computed(() => {
  if (!form.enabled) {
    return t('admin.groups.runtimeStatus.disabled')
  }
  if (!summary.value.observed_at) {
    return t('admin.groups.runtimeStatus.waiting')
  }
  return t(`modelStatus.statuses.${summaryStatus.value}`)
})

const showKeywordEditor = computed(() => shouldShowRuntimeKeywordEditor(form.validation_mode))

function resetForm() {
  form.enabled = false
  form.probe_model = ''
  form.probe_prompt = ''
  form.validation_mode = 'non_empty'
  form.interval_seconds = 60
  form.timeout_seconds = 30
  form.slow_latency_ms = 15000
  form.notify_enabled = true
  form.modeltrace_enabled = false
  form.modeltrace_expected_model = ''
  form.modeltrace_request_model = ''
  form.modeltrace_interval_seconds = 3600
  form.astra_check_enabled = false
  form.astra_check_request_model = 'gpt-6-astra'
  form.astra_check_tier = 'low'
  form.astra_check_interval_seconds = 3600
  expectedKeywordsText.value = ''
}

// 后端某些路径会把空切片序列化成 null；模板里直接读 .length / .map，这里统一兜成空数组，
// 否则渲染抛错会让整个弹窗组件崩掉。
function normalizeAstraView(view: GroupStatusAdminView): GroupStatusAdminView {
  const summary = view.summary
  if (summary) {
    if (!Array.isArray(summary.astra_check_matches)) summary.astra_check_matches = []
    if (!Array.isArray(summary.astra_check_reasons)) summary.astra_check_reasons = []
    if (!Array.isArray(summary.astra_check_benchmark_models)) summary.astra_check_benchmark_models = []
    if (!Array.isArray(summary.astra_check_benchmark_tiers)) summary.astra_check_benchmark_tiers = []
    if (!Array.isArray(summary.modeltrace_ranking)) summary.modeltrace_ranking = []
    if (!Array.isArray(summary.modeltrace_reasons)) summary.modeltrace_reasons = []
  }
  if (!Array.isArray(view.modeltrace_targets)) view.modeltrace_targets = []
  if (view.modeltrace_progress && !Array.isArray(view.modeltrace_progress.attempts)) {
    view.modeltrace_progress.attempts = []
  }
  const modelTraceRun = view.modeltrace_last_run
  if (modelTraceRun) {
    if (!Array.isArray(modelTraceRun.outputs)) modelTraceRun.outputs = []
    if (!Array.isArray(modelTraceRun.ranking)) modelTraceRun.ranking = []
    if (!Array.isArray(modelTraceRun.reasons)) modelTraceRun.reasons = []
  }
  if (view.astra_check_progress && !Array.isArray(view.astra_check_progress.samples)) {
    view.astra_check_progress.samples = []
  }
  const run = view.astra_check_last_run
  if (run) {
    if (!Array.isArray(run.samples)) run.samples = []
    if (!Array.isArray(run.cells)) run.cells = []
    if (!Array.isArray(run.matches)) run.matches = []
    if (!Array.isArray(run.reasons)) run.reasons = []
  }
  return view
}

function applyView(view: GroupStatusAdminView) {
  currentView.value = normalizeAstraView(view)
  form.enabled = view.config.enabled
  form.probe_model = view.config.probe_model
  form.probe_prompt = view.config.probe_prompt
  form.validation_mode = view.config.validation_mode
  form.interval_seconds = view.config.interval_seconds
  form.timeout_seconds = view.config.timeout_seconds
  form.slow_latency_ms = view.config.slow_latency_ms
  form.notify_enabled = view.config.notify_enabled !== false
  form.modeltrace_enabled = view.config.modeltrace_enabled === true
  form.modeltrace_expected_model =
    view.config.modeltrace_expected_model || view.modeltrace_targets?.[0]?.id || ''
  form.modeltrace_request_model = view.config.modeltrace_request_model || ''
  form.modeltrace_interval_seconds = view.config.modeltrace_interval_seconds || 3600
  form.astra_check_enabled = view.config.astra_check_enabled === true
  form.astra_check_request_model = view.config.astra_check_request_model || 'gpt-6-astra'
  form.astra_check_tier = view.config.astra_check_tier || 'low'
  form.astra_check_interval_seconds = view.config.astra_check_interval_seconds || 3600
  expectedKeywordsText.value = joinRuntimeKeywordsText(view.config.expected_keywords)
}

async function loadRuntimeStatus(groupId: number) {
  loading.value = true
  loadError.value = ''
  try {
    const view = await adminAPI.groups.getRuntimeStatus(groupId)
    applyView(view)
    // 打开弹窗时如果后台（定时调度或上次点击）正在跑 Astra 验证，直接接上进度轮询
    if (view.summary.astra_check_running && !astraPollTimer) {
      astraProbing.value = true
      astraPollStartedAt = Date.now()
      scheduleAstraPoll(groupId)
    }
    if (view.summary.modeltrace_running && !modelTracePollTimer) {
      modelTraceProbing.value = true
      modelTracePollStartedAt = Date.now()
      scheduleModelTracePoll(groupId)
    }
  } catch (error: any) {
    loadError.value = error?.message || t('admin.groups.runtimeStatus.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function saveRuntimeStatus(showToast = true): Promise<GroupStatusAdminView | null> {
  if (!props.group) {
    return null
  }

  saving.value = true
  loadError.value = ''
  try {
    const view = await adminAPI.groups.updateRuntimeStatus(props.group.id, {
      enabled: form.enabled,
      probe_model: form.probe_model.trim(),
      probe_prompt: form.probe_prompt.trim(),
      validation_mode: form.validation_mode,
      expected_keywords: splitRuntimeKeywordsText(expectedKeywordsText.value),
      interval_seconds: Math.max(10, Math.round(Number(form.interval_seconds) || 60)),
      timeout_seconds: Math.max(1, Math.round(Number(form.timeout_seconds) || 30)),
      slow_latency_ms: Math.max(100, Math.round(Number(form.slow_latency_ms) || 15000)),
      notify_enabled: form.notify_enabled,
      ...(modelTraceSupported.value
        ? {
            modeltrace_enabled: form.modeltrace_enabled,
            modeltrace_expected_model: form.modeltrace_expected_model,
            modeltrace_request_model: form.modeltrace_request_model.trim(),
            modeltrace_interval_seconds: Math.max(900, Math.round(Number(form.modeltrace_interval_seconds) || 3600)),
          }
        : {}),
      astra_check_enabled: form.astra_check_enabled,
      astra_check_request_model: form.astra_check_request_model.trim() || 'gpt-6-astra',
      astra_check_tier: form.astra_check_tier,
      astra_check_interval_seconds: Math.max(900, Math.round(Number(form.astra_check_interval_seconds) || 3600)),
    })
    applyView(view)
    if (showToast) {
      appStore.showSuccess(t('admin.groups.runtimeStatus.saved'))
    }
    emit('updated')
    return view
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.failedToSave'))
    return null
  } finally {
    saving.value = false
  }
}

async function handleSave() {
  const saved = await saveRuntimeStatus(true)
  if (saved) {
    emit('close')
  }
}

async function handleProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  probing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatus(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.probeSucceeded'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.probeFailed'))
  } finally {
    probing.value = false
  }
}

function stopModelTracePolling() {
  if (modelTracePollTimer) {
    clearTimeout(modelTracePollTimer)
    modelTracePollTimer = null
  }
  modelTraceProbing.value = false
}

// 一轮 ModelTrace 验证要发 3–6 条长输出请求，后端在后台跑；这里定时拉管理视图直到 running 结束。
function scheduleModelTracePoll(groupId: number) {
  modelTracePollTimer = setTimeout(async () => {
    modelTracePollTimer = null
    if (!props.show || props.group?.id !== groupId) {
      stopModelTracePolling()
      return
    }
    try {
      const view = await adminAPI.groups.getRuntimeStatus(groupId)
      applyView(view)
      if (view.summary.modeltrace_running && Date.now() - modelTracePollStartedAt < MODEL_TRACE_POLL_MAX_MS) {
        scheduleModelTracePoll(groupId)
        return
      }
      stopModelTracePolling()
      emit('updated')
      if (!view.summary.modeltrace_running) {
        appStore.showSuccess(t('admin.groups.runtimeStatus.modelTrace.probeSucceeded'))
      }
    } catch (error: any) {
      stopModelTracePolling()
      appStore.showError(error?.message || t('admin.groups.runtimeStatus.modelTrace.probeFailed'))
    }
  }, MODEL_TRACE_POLL_INTERVAL_MS)
}

async function handleModelTraceProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  modelTraceProbing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatusModelTrace(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.modelTrace.probeStarted'))
    modelTracePollStartedAt = Date.now()
    scheduleModelTracePoll(props.group.id)
  } catch (error: any) {
    modelTraceProbing.value = false
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.modelTrace.probeFailed'))
  }
}

function stopAstraPolling() {
  if (astraPollTimer) {
    clearTimeout(astraPollTimer)
    astraPollTimer = null
  }
  astraProbing.value = false
}

// 一轮 Astra 指纹验证有几十个请求，后端在后台跑；这里每 3s 拉一次管理视图直到 running 结束。
function scheduleAstraPoll(groupId: number) {
  astraPollTimer = setTimeout(async () => {
    astraPollTimer = null
    if (!props.show || props.group?.id !== groupId) {
      stopAstraPolling()
      return
    }
    try {
      const view = await adminAPI.groups.getRuntimeStatus(groupId)
      applyView(view)
      if (view.summary.astra_check_running && Date.now() - astraPollStartedAt < ASTRA_POLL_MAX_MS) {
        scheduleAstraPoll(groupId)
        return
      }
      stopAstraPolling()
      emit('updated')
      if (!view.summary.astra_check_running) {
        appStore.showSuccess(t('admin.groups.runtimeStatus.astraCheck.probeSucceeded'))
      }
    } catch (error: any) {
      stopAstraPolling()
      appStore.showError(error?.message || t('admin.groups.runtimeStatus.astraCheck.probeFailed'))
    }
  }, ASTRA_POLL_INTERVAL_MS)
}

async function handleAstraCheckProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  astraProbing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatusAstraCheck(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.astraCheck.probeStarted'))
    astraPollStartedAt = Date.now()
    scheduleAstraPoll(props.group.id)
  } catch (error: any) {
    astraProbing.value = false
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.astraCheck.probeFailed'))
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show) {
      stopAstraPolling()
      stopModelTracePolling()
    }
  }
)

onBeforeUnmount(() => {
  stopAstraPolling()
  stopModelTracePolling()
})

watch(
  () => [props.show, props.group?.id] as const,
  ([show, groupId]) => {
    if (!show || !groupId) {
      if (!show) {
        resetForm()
        currentView.value = null
        loadError.value = ''
      }
      return
    }
    void loadRuntimeStatus(groupId)
  },
  { immediate: true }
)
</script>
