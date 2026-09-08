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
          v-if="group.platform === 'openai'"
          class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
        >
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.solJuice.title') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.solJuice.hint') }}
              </p>
            </div>
            <Toggle v-model="form.sol_juice_enabled" />
          </div>

          <div class="grid gap-5 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.solJuice.intervalSeconds') }}</label>
              <input
                v-model.number="form.sol_juice_interval_seconds"
                type="number"
                min="300"
                step="60"
                class="input"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.solJuice.model') }}</label>
              <input
                v-model.trim="form.sol_juice_model"
                type="text"
                class="input"
                :placeholder="t('admin.groups.runtimeStatus.solJuice.modelPlaceholder')"
              />
            </div>
          </div>

          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.solJuice.latestResult') }}
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing || solJuiceProbing"
              @click="handleSolJuiceProbe"
            >
              <span
                v-if="solJuiceProbing"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{ solJuiceProbing ? t('admin.groups.runtimeStatus.solJuice.probing') : t('admin.groups.runtimeStatus.solJuice.probeNow') }}
            </button>
          </div>

          <div v-if="summary.sol_juice_checked_at" class="space-y-3">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.status') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', getSolJuiceBadgeClass(solJuiceDisplayStatus)]">
                    {{ t(`admin.groups.runtimeStatus.solJuice.statuses.${solJuiceDisplayStatus}`) }}
                  </span>
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.value') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.sol_juice_value || '-' }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.sol_juice_checked_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.sol_juice_checked_at) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.tokens') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.sol_juice_input_tokens }} / {{ summary.sol_juice_output_tokens }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.reasoningTokens') }}: {{ summary.sol_juice_reasoning_tokens }}
                </div>
              </div>
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.lastCost') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatUsd(summary.sol_juice_last_cost_usd) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.monthlyEstimate') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatUsd(solJuiceMonthlyCost, 2) }}
                </div>
              </div>
            </div>

            <div
              v-if="summary.sol_juice_detail"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.solJuice.detail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.sol_juice_detail }}</pre>
            </div>
          </div>
          <div
            v-else
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.solJuice.latestResultEmpty') }}
          </div>
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
                · {{ summary.astra_check_benchmark_models.map((m) => astraModelShortName(m.id)).join(' / ') }}
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
            <div v-if="astraProgress.samples.length > 0" class="max-h-56 overflow-auto">
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
                  <tr v-for="row in astraSampleRows(astraProgress.samples)" :key="row.seq" class="border-t border-gray-100 dark:border-dark-700">
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
              v-if="astraLastRun && (astraLastRun.samples.length > 0 || astraLastRun.cells.length > 0)"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.title', { count: astraLastRun.samples.length }) }}
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
                <div v-if="astraLastRun.samples.length > 0" class="max-h-72 overflow-auto">
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
              v-if="summary.astra_check_reasons.length > 0"
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
} from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import {
  astraModelShortName,
  estimateSolJuiceMonthlyCostUsd,
  formatGroupRuntimeLatency,
  formatMatchPercent,
  formatUsd,
  getAstraCheckBadgeClass,
  getGroupRuntimeStatusBadgeClass,
  getSolJuiceBadgeClass,
  joinRuntimeKeywordsText,
  normalizeAstraCheckStatus,
  normalizeSolJuiceStatus,
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
  sol_juice_enabled: false,
  sol_juice_interval_seconds: 900,
  sol_juice_model: 'gpt-5.6-sol',
  astra_check_enabled: false,
  astra_check_request_model: 'gpt-6-astra',
  astra_check_tier: 'low' as AstraCheckTier,
  astra_check_interval_seconds: 3600,
})

const solJuiceProbing = ref(false)
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
    http_code: null,
    sub_status: '',
    error_detail: '',
    observed_at: null,
    consecutive_down: 0,
    consecutive_non_down: 0,
    sol_juice_enabled: false,
    sol_juice_model: '',
    sol_juice_interval_seconds: 900,
    sol_juice_status: '',
    sol_juice_stable_status: '',
    sol_juice_value: '',
    sol_juice_detail: '',
    sol_juice_checked_at: null,
    sol_juice_consecutive_mismatch: 0,
    sol_juice_input_tokens: 0,
    sol_juice_output_tokens: 0,
    sol_juice_reasoning_tokens: 0,
    sol_juice_last_cost_usd: 0,
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
  if (astraDisplayStatus.value === 'mismatch') {
    return t('admin.groups.runtimeStatus.astraCheck.statuses.mismatch', {
      winner: astraModelShortName(summary.value.astra_check_winner)
    })
  }
  return t(`admin.groups.runtimeStatus.astraCheck.statuses.${astraDisplayStatus.value}`)
})

const astraMonthlyCost = computed(() =>
  estimateSolJuiceMonthlyCostUsd(summary.value.astra_check_last_cost_usd, Number(form.astra_check_interval_seconds) || 0)
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

const solJuiceDisplayStatus = computed(() =>
  normalizeSolJuiceStatus(summary.value.sol_juice_stable_status || summary.value.sol_juice_status)
)

const solJuiceMonthlyCost = computed(() =>
  estimateSolJuiceMonthlyCostUsd(summary.value.sol_juice_last_cost_usd, Number(form.sol_juice_interval_seconds) || 0)
)

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
  form.sol_juice_enabled = false
  form.sol_juice_interval_seconds = 900
  form.sol_juice_model = 'gpt-5.6-sol'
  form.astra_check_enabled = false
  form.astra_check_request_model = 'gpt-6-astra'
  form.astra_check_tier = 'low'
  form.astra_check_interval_seconds = 3600
  expectedKeywordsText.value = ''
}

function applyView(view: GroupStatusAdminView) {
  currentView.value = view
  form.enabled = view.config.enabled
  form.probe_model = view.config.probe_model
  form.probe_prompt = view.config.probe_prompt
  form.validation_mode = view.config.validation_mode
  form.interval_seconds = view.config.interval_seconds
  form.timeout_seconds = view.config.timeout_seconds
  form.slow_latency_ms = view.config.slow_latency_ms
  form.notify_enabled = view.config.notify_enabled !== false
  form.sol_juice_enabled = view.config.sol_juice_enabled === true
  form.sol_juice_interval_seconds = view.config.sol_juice_interval_seconds || 900
  form.sol_juice_model = view.config.sol_juice_model || 'gpt-5.6-sol'
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
      sol_juice_enabled: form.sol_juice_enabled,
      sol_juice_interval_seconds: Math.max(300, Math.round(Number(form.sol_juice_interval_seconds) || 900)),
      sol_juice_model: form.sol_juice_model.trim() || 'gpt-5.6-sol',
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

async function handleSolJuiceProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  solJuiceProbing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatusSolJuice(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.solJuice.probeSucceeded'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.solJuice.probeFailed'))
  } finally {
    solJuiceProbing.value = false
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
    }
  }
)

onBeforeUnmount(() => {
  stopAstraPolling()
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
