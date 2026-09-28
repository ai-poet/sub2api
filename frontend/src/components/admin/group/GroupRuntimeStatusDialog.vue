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
          v-if="astraSupported"
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
              <p
                v-for="bench in astraBenchmarks"
                :key="bench.package_id"
                class="mt-1 text-xs text-gray-400 dark:text-gray-500"
              >
                {{ t('admin.groups.runtimeStatus.astraCheck.benchmark') }}: {{ bench.package_id }} {{ bench.version }}
                · {{ astraBenchmarkModelNames(bench) }}
              </p>
            </div>
            <Toggle v-model="form.astra_check_enabled" />
          </div>

          <div class="space-y-2">
            <label class="input-label">{{ t('admin.groups.runtimeStatus.astraCheck.models') }}</label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.runtimeStatus.astraCheck.modelsHint') }}
            </p>
            <div
              v-for="target in astraTargets"
              :key="target.id"
              class="grid items-center gap-3 md:grid-cols-[minmax(0,14rem)_minmax(0,1fr)]"
            >
              <label class="flex items-center gap-2 text-sm text-gray-800 dark:text-gray-100">
                <input
                  v-model="astraModelForm(target.id).selected"
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 dark:border-dark-600"
                  :data-astra-model="target.id"
                />
                <span>{{ target.display_name }}</span>
              </label>
              <input
                v-model.trim="astraModelForm(target.id).request_model"
                type="text"
                class="input"
                :placeholder="target.default_request_model"
                :disabled="!astraModelForm(target.id).selected"
                :aria-label="t('admin.groups.runtimeStatus.astraCheck.requestModel')"
                :data-astra-request-model="target.id"
              />
            </div>
          </div>

          <div class="grid gap-5 md:grid-cols-2">
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
              <span v-if="astraMonthlyTotal !== null" class="ml-2 text-xs font-normal text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.astraCheck.monthlyTotal') }}: {{ formatUsd(astraMonthlyTotal, 2) }}
              </span>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              data-astra-probe-all
              :disabled="astraProbeDisabled || astraSelectedModels.length === 0"
              @click="handleAstraCheckProbe()"
            >
              <span
                v-if="astraBusy"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{
                astraBusy
                  ? t('admin.groups.runtimeStatus.astraCheck.running')
                  : t('admin.groups.runtimeStatus.astraCheck.probeAll')
              }}
            </button>
          </div>

          <div
            v-if="astraProgress"
            class="space-y-3 rounded-lg border border-emerald-200 bg-emerald-50/40 px-3 py-3 dark:border-emerald-900/40 dark:bg-emerald-950/20"
          >
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.astraCheck.progress.title', {
                  model: astraModelName(astraProgress.expected_model),
                  index: astraProgress.model_index || 1,
                  count: astraProgress.model_count || 1,
                  round: astraProgress.round
                }) }}
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

          <div
            v-for="state in astraStates"
            :key="state.expected_model"
            class="space-y-3 rounded-lg border border-gray-200 px-3 py-3 dark:border-dark-700"
            :data-astra-state="state.expected_model"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ state.display_name || astraModelName(state.expected_model) }}
                </span>
                <span :class="['badge', getAstraCheckBadgeClass(astraStateStatus(state))]">
                  {{ astraStateText(state) }}
                </span>
              </div>
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                :data-astra-probe-model="state.expected_model"
                :disabled="astraProbeDisabled"
                @click="handleAstraCheckProbe(state.expected_model)"
              >
                {{ t('admin.groups.runtimeStatus.astraCheck.probeModel') }}
              </button>
            </div>

            <template v-if="state.checked_at">
              <div class="grid gap-3 md:grid-cols-3">
                <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.groups.runtimeStatus.astraCheck.samples') }}
                  </div>
                  <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                    {{ state.valid_samples }} / {{ state.planned_samples }}
                  </div>
                  <div v-if="state.benchmark_version" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.groups.runtimeStatus.astraCheck.benchmark') }} {{ state.benchmark_version }}
                  </div>
                </div>
                <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.groups.runtimeStatus.astraCheck.checkedAt') }}
                  </div>
                  <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                    {{ formatDateTime(state.checked_at) }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ formatRelativeTime(state.checked_at) }}
                  </div>
                </div>
                <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.groups.runtimeStatus.astraCheck.tokens') }}
                  </div>
                  <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                    {{ state.input_tokens }} / {{ state.output_tokens }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.groups.runtimeStatus.astraCheck.lastCost') }}: {{ formatUsd(state.last_cost_usd) }}
                    · {{ t('admin.groups.runtimeStatus.astraCheck.monthlyEstimate') }}:
                    {{ formatUsd(estimateMonthlyProbeCostUsd(state.last_cost_usd, astraIntervalSeconds), 2) }}
                  </div>
                </div>
              </div>

              <div
                v-if="(state.matches?.length ?? 0) > 0"
                class="space-y-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
              >
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.matches') }}
                </div>
                <div v-for="m in state.matches" :key="m.model">
                  <div class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-200">
                    <span :class="m.model === state.expected_model || m.passed ? 'font-semibold' : ''">
                      {{ astraModelName(m.model) }}
                    </span>
                    <span>
                      {{ formatMatchPercent(m.match) }}
                      <span class="text-gray-400">/ {{ t('admin.groups.runtimeStatus.astraCheck.threshold') }} {{ formatMatchPercent(m.threshold) }}</span>
                    </span>
                  </div>
                  <div class="mt-1 h-2 w-full rounded bg-gray-200 dark:bg-dark-700">
                    <div
                      class="h-2 rounded"
                      :class="m.passed ? (m.model === state.expected_model ? 'bg-emerald-500' : 'bg-rose-500') : 'bg-gray-400'"
                      :style="{ width: `${Math.min(100, Math.round(m.match * 100))}%` }"
                    ></div>
                  </div>
                </div>
              </div>

              <div v-if="(state.reasons?.length ?? 0) > 0" class="flex flex-wrap gap-2">
                <span v-for="reason in state.reasons" :key="reason" class="badge badge-warning">
                  {{ astraReasonLabel(reason) }}
                </span>
              </div>

              <div
                v-if="state.detail"
                class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
              >
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.detail') }}
                </div>
                <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ state.detail }}</pre>
              </div>

              <details
                v-if="astraLastRunFor(state.expected_model)"
                class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
              >
                <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.title', { count: astraLastRunFor(state.expected_model)!.samples.length }) }}
                  <span class="ml-1 font-normal">
                    · {{ t('admin.groups.runtimeStatus.astraCheck.sampleTable.runMeta', {
                      planned: astraLastRunFor(state.expected_model)!.requests_planned,
                      completed: astraLastRunFor(state.expected_model)!.requests_completed,
                      latency: formatGroupRuntimeLatency(astraLastRunFor(state.expected_model)!.latency_ms)
                    }) }}
                    <template v-if="astraLastRunFor(state.expected_model)!.account_id">
                      · {{ t('admin.groups.runtimeStatus.astraCheck.progress.account') }} #{{ astraLastRunFor(state.expected_model)!.account_id }}
                    </template>
                    · {{ astraLastRunFor(state.expected_model)!.request_model }}
                  </span>
                </summary>
                <div class="mt-2 space-y-2">
                  <div
                    v-for="cell in astraLastRunFor(state.expected_model)!.cells"
                    :key="cell.cell_id"
                    class="text-xs text-gray-600 dark:text-gray-300"
                  >
                    <span class="font-mono font-medium">{{ cell.cell_id }}</span>: {{ formatAstraCellCounts(cell) }}
                  </div>
                  <div v-if="astraLastRunFor(state.expected_model)!.samples.length > 0" class="max-h-72 overflow-auto">
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
                        <tr
                          v-for="row in astraLastRunFor(state.expected_model)!.samples"
                          :key="row.seq"
                          class="border-t border-gray-100 dark:border-dark-700"
                        >
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
            </template>
            <div
              v-else
              class="rounded-lg border border-dashed border-gray-200 px-4 py-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
            >
              {{ t('admin.groups.runtimeStatus.astraCheck.notChecked') }}
            </div>
          </div>
          <div
            v-if="astraStates.length === 0"
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.astraCheck.latestResultEmpty') }}
          </div>

          <p class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.groups.runtimeStatus.astraCheck.disclaimer') }}
          </p>
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
  AstraBenchmarkMeta,
  AstraCheckCellSummary,
  AstraCheckLastRun,
  AstraCheckModelConfig,
  AstraCheckProgress,
  AstraCheckSampleRecord,
  AstraCheckState,
  AstraCheckTarget,
  AstraCheckTier,
  GroupStatusAdminView,
  GroupStatusSummary,
  GroupStatusValidationMode,
} from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import {
  ASTRA_OTHER_MODEL,
  astraModelLabel,
  estimateMonthlyProbeCostUsd,
  formatGroupRuntimeLatency,
  formatMatchPercent,
  formatUsd,
  getAstraCheckBadgeClass,
  getGroupRuntimeStatusBadgeClass,
  joinRuntimeKeywordsText,
  normalizeAstraCheckStatus,
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
  astra_check_enabled: false,
  astra_check_tier: 'low' as AstraCheckTier,
  astra_check_interval_seconds: 3600,
})

// meow 指纹验证：每个可选目标一行（是否检测 + 请求模型名），在 applyView 里按后端给的目标列表建好
interface AstraModelForm {
  selected: boolean
  request_model: string
}
const astraModelForms = reactive<Record<string, AstraModelForm>>({})
const astraModelFormFallback: AstraModelForm = { selected: false, request_model: '' }

const astraProbing = ref(false)
let astraPollTimer: ReturnType<typeof setTimeout> | null = null
let astraPollStartedAt = 0
const ASTRA_POLL_INTERVAL_MS = 2000
// 一次运行依次检测多个模型，后端总预算最多一小时
const ASTRA_POLL_MAX_MS = 60 * 60 * 1000

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
    astra_check_enabled: false,
    astra_check_models: [],
    astra_check_tier: 'low',
    astra_check_interval_seconds: 3600,
    astra_check_states: [],
    astra_check_running: false,
    astra_check_benchmarks: []
  }
})

// ==================== meow 指纹验证 ====================

const astraSupported = computed(() => props.group?.platform === 'openai' || props.group?.platform === 'anthropic')
const astraTargets = computed<AstraCheckTarget[]>(() => currentView.value?.astra_check_targets ?? [])
const astraStates = computed<AstraCheckState[]>(() => summary.value.astra_check_states ?? [])
const astraBusy = computed(() => astraProbing.value || summary.value.astra_check_running)
const astraProbeDisabled = computed(() => loading.value || saving.value || probing.value || astraBusy.value)
const astraIntervalSeconds = computed(() => Number(form.astra_check_interval_seconds) || 0)

// 只列本平台目标用到的基准包
const astraBenchmarks = computed<AstraBenchmarkMeta[]>(() => {
  const packages = new Set(astraTargets.value.map((target) => target.package_id))
  return (summary.value.astra_check_benchmarks ?? []).filter((bench) => packages.has(bench.package_id))
})

function astraModelForm(id: string): AstraModelForm {
  return astraModelForms[id] ?? astraModelFormFallback
}

// 按目标列表顺序给出勾选的模型（请求模型名留空 = 目标默认名）
const astraSelectedModels = computed<AstraCheckModelConfig[]>(() =>
  astraTargets.value
    .filter((target) => astraModelForms[target.id]?.selected)
    .map((target) => ({
      expected_model: target.id,
      request_model: (astraModelForms[target.id]?.request_model ?? '').trim()
    }))
)

function astraModelName(model?: string | null): string {
  if ((model || '').trim() === ASTRA_OTHER_MODEL) {
    return t('admin.groups.runtimeStatus.astraCheck.otherModel')
  }
  const target = astraTargets.value.find((item) => item.id === model)
  return target?.display_name || astraModelLabel(model)
}

function astraBenchmarkModelNames(bench: AstraBenchmarkMeta): string {
  return (bench.models ?? []).map((model) => astraModelName(model.id)).join(' / ')
}

// 档位后面标出本平台各基准包每个模型的请求数（包之间不同时给范围）
const astraTierOptions = computed(() => {
  return (['low', 'medium', 'high'] as AstraCheckTier[]).map((tier) => {
    const counts = astraBenchmarks.value
      .map((bench) => (bench.tiers ?? []).find((item) => item.tier === tier)?.requests ?? 0)
      .filter((n) => n > 0)
    const min = Math.min(...counts)
    const max = Math.max(...counts)
    const requests = counts.length === 0 ? '' : min === max ? ` (${min})` : ` (${min}–${max})`
    return { value: tier, label: `${t(`admin.groups.runtimeStatus.astraCheck.tiers.${tier}`)}${requests}` }
  })
})

function astraStateStatus(state: AstraCheckState) {
  return normalizeAstraCheckStatus(state.stable_status, state.verdict)
}

function astraStateText(state: AstraCheckState): string {
  const status = astraStateStatus(state)
  if (status === 'mismatch' || status === 'suspect') {
    if (!(state.winner || '').trim()) {
      return t(`admin.groups.runtimeStatus.astraCheck.statuses.${status}NoWinner`)
    }
    return t(`admin.groups.runtimeStatus.astraCheck.statuses.${status}`, { winner: astraModelName(state.winner) })
  }
  return t(`admin.groups.runtimeStatus.astraCheck.statuses.${status}`)
}

// 所有已检测模型按当前间隔折算的月度成本合计；没有成本记录时为 null
const astraMonthlyTotal = computed(() => {
  let total = 0
  let any = false
  for (const state of astraStates.value) {
    const monthly = estimateMonthlyProbeCostUsd(state.last_cost_usd, astraIntervalSeconds.value)
    if (monthly !== null) {
      total += monthly
      any = true
    }
  }
  return any ? total : null
})

function astraReasonLabel(reason: string): string {
  const key = `admin.groups.runtimeStatus.astraCheck.reasons.${reason}`
  return te(key) ? t(key) : reason
}

// 验证进行中的实时进度（仅运行时后端才返回）与每个模型最近一次运行的完整记录
const astraProgress = computed<AstraCheckProgress | null>(() => currentView.value?.astra_check_progress ?? null)
const astraLastRuns = computed<AstraCheckLastRun[]>(() => currentView.value?.astra_check_last_runs ?? [])
const astraProgressPercent = computed(() => {
  const p = astraProgress.value
  if (!p || p.planned <= 0) {
    return 0
  }
  return Math.min(100, Math.round((p.completed / p.planned) * 100))
})

function astraLastRunFor(model: string): AstraCheckLastRun | null {
  return astraLastRuns.value.find((run) => run.expected_model === model) ?? null
}

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

// ==================== 存活探测 ====================

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

function resetAstraModelForms(targets: AstraCheckTarget[], models: AstraCheckModelConfig[]) {
  for (const key of Object.keys(astraModelForms)) {
    delete astraModelForms[key]
  }
  for (const target of targets) {
    const configured = models.find((item) => item.expected_model === target.id)
    astraModelForms[target.id] = {
      selected: Boolean(configured),
      request_model: configured?.request_model ?? ''
    }
  }
}

function resetForm() {
  form.enabled = false
  form.probe_model = ''
  form.probe_prompt = ''
  form.validation_mode = 'non_empty'
  form.interval_seconds = 60
  form.timeout_seconds = 30
  form.slow_latency_ms = 15000
  form.notify_enabled = true
  form.astra_check_enabled = false
  form.astra_check_tier = 'low'
  form.astra_check_interval_seconds = 3600
  resetAstraModelForms([], [])
  expectedKeywordsText.value = ''
}

// 后端某些路径会把空切片序列化成 null；模板里直接读 .length / .map，这里统一兜成空数组，
// 否则渲染抛错会让整个弹窗组件崩掉。
function normalizeAstraView(view: GroupStatusAdminView): GroupStatusAdminView {
  const summary = view.summary
  if (summary) {
    if (!Array.isArray(summary.astra_check_models)) summary.astra_check_models = []
    if (!Array.isArray(summary.astra_check_states)) summary.astra_check_states = []
    if (!Array.isArray(summary.astra_check_benchmarks)) summary.astra_check_benchmarks = []
    for (const state of summary.astra_check_states) {
      if (!Array.isArray(state.matches)) state.matches = []
      if (!Array.isArray(state.reasons)) state.reasons = []
    }
  }
  if (view.config && !Array.isArray(view.config.astra_check_models)) view.config.astra_check_models = []
  if (!Array.isArray(view.astra_check_targets)) view.astra_check_targets = []
  if (!Array.isArray(view.astra_check_last_runs)) view.astra_check_last_runs = []
  if (view.astra_check_progress && !Array.isArray(view.astra_check_progress.samples)) {
    view.astra_check_progress.samples = []
  }
  for (const run of view.astra_check_last_runs) {
    if (!Array.isArray(run.samples)) run.samples = []
    if (!Array.isArray(run.cells)) run.cells = []
    if (!Array.isArray(run.matches)) run.matches = []
    if (!Array.isArray(run.reasons)) run.reasons = []
  }
  return view
}

// syncAstraForm 为 false 时（轮询刷新）只更新结果，不覆盖管理员尚未保存的勾选与输入
function applyView(view: GroupStatusAdminView, syncAstraForm = true) {
  currentView.value = normalizeAstraView(view)
  form.enabled = view.config.enabled
  form.probe_model = view.config.probe_model
  form.probe_prompt = view.config.probe_prompt
  form.validation_mode = view.config.validation_mode
  form.interval_seconds = view.config.interval_seconds
  form.timeout_seconds = view.config.timeout_seconds
  form.slow_latency_ms = view.config.slow_latency_ms
  form.notify_enabled = view.config.notify_enabled !== false
  expectedKeywordsText.value = joinRuntimeKeywordsText(view.config.expected_keywords)
  if (syncAstraForm) {
    form.astra_check_enabled = view.config.astra_check_enabled === true
    form.astra_check_tier = view.config.astra_check_tier || 'low'
    form.astra_check_interval_seconds = view.config.astra_check_interval_seconds || 3600
    resetAstraModelForms(view.astra_check_targets ?? [], view.config.astra_check_models ?? [])
  }
}

async function loadRuntimeStatus(groupId: number) {
  loading.value = true
  loadError.value = ''
  try {
    const view = await adminAPI.groups.getRuntimeStatus(groupId)
    applyView(view)
    // 打开弹窗时如果后台（定时调度或上次点击）正在跑指纹验证，直接接上进度轮询
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
  if (astraSupported.value && form.astra_check_enabled && astraSelectedModels.value.length === 0) {
    appStore.showError(t('admin.groups.runtimeStatus.astraCheck.noModelSelected'))
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
      // 其他平台不带这些字段，后端保留已保存的值
      ...(astraSupported.value
        ? {
            astra_check_enabled: form.astra_check_enabled,
            astra_check_models: astraSelectedModels.value,
            astra_check_tier: form.astra_check_tier,
            astra_check_interval_seconds: Math.max(900, Math.round(Number(form.astra_check_interval_seconds) || 3600)),
          }
        : {}),
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

function stopAstraPolling() {
  if (astraPollTimer) {
    clearTimeout(astraPollTimer)
    astraPollTimer = null
  }
  astraProbing.value = false
}

// 一轮指纹验证每个模型几十个请求，后端在后台依次跑；这里定时拉管理视图直到 running 结束。
function scheduleAstraPoll(groupId: number) {
  astraPollTimer = setTimeout(async () => {
    astraPollTimer = null
    if (!props.show || props.group?.id !== groupId) {
      stopAstraPolling()
      return
    }
    try {
      const view = await adminAPI.groups.getRuntimeStatus(groupId)
      applyView(view, false)
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

// expectedModel 为空时检测全部已勾选的模型；先保存，保证要检测的模型已在配置里
async function handleAstraCheckProbe(expectedModel?: string) {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  astraProbing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatusAstraCheck(props.group.id, expectedModel)
    applyView(view, false)
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
