<template>
  <!-- 右侧「团队」面板，同客户端 team_panel.rs：标题和阶段、统计、进度条、操作、成员、任务关系图 -->
  <div v-if="team" class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden px-3.5 py-3" data-test="preview-team-panel" :data-phase="team.phase">
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-2">
        <span class="min-w-0 flex-1 truncate text-[14px] font-medium text-[color:var(--cw-text)]">
          {{ t('home.clientWorkflow.team.name') }}
        </span>
        <span
          class="flex shrink-0 items-center gap-1 rounded-[5px] border px-[7px] py-[2px] text-[11.5px]"
          :style="{ color: phase.color, borderColor: phase.color }"
          data-test="preview-team-phase"
        >
          <ClientIcon :name="phase.icon" class="h-[11px] w-[11px]" :class="team.phase === 'running' ? 'cw-spin' : ''" />
          {{ phase.label }}
        </span>
      </div>
      <div class="truncate text-[11.5px] tabular-nums text-[color:var(--cw-text-tertiary)]" data-test="preview-team-stats">
        {{ stats }}
      </div>
    </div>

    <!-- 每个任务一段，颜色之外再用文字写出各状态的数量 -->
    <div class="flex flex-col gap-[5px]">
      <div class="flex h-1.5 gap-0.5">
        <span
          v-for="(state, index) in team.tasks"
          :key="index"
          class="h-full flex-1 rounded-[2px] transition-colors duration-300 motion-reduce:transition-none"
          :style="{ background: TASK_STATES[state].color }"
        ></span>
      </div>
      <div class="cw-team-words truncate text-[11.5px] text-[color:var(--cw-text-tertiary)]">{{ progressWords }}</div>
    </div>

    <!-- 操作：草案时确认 / 退回 / 放弃，运行中只有停止；放不下的按钮折到第二行并被裁掉 -->
    <div v-if="team.phase !== 'done'" class="flex h-7 flex-none flex-wrap items-center gap-2 overflow-hidden">
      <template v-if="team.phase === 'review'">
        <span
          class="cw-primary-button no-key"
          :class="team.approvePressed ? 'is-pressed' : ''"
          data-test="preview-team-approve"
        >{{ t('home.clientWorkflow.team.approve') }}</span>
        <span class="cw-secondary-button">{{ t('home.clientWorkflow.team.revise') }}</span>
        <span class="cw-secondary-button">{{ t('home.clientWorkflow.team.discard') }}</span>
      </template>
      <span v-else class="cw-secondary-button">
        <ClientIcon name="stop" class="h-3 w-3" />
        {{ t('home.clientWorkflow.team.stop') }}
      </span>
    </div>

    <div class="flex flex-col gap-1.5">
      <div class="text-[11.5px] font-medium text-[color:var(--cw-text-tertiary)]">{{ t('home.clientWorkflow.team.members') }}</div>
      <div class="flex flex-col gap-0.5">
        <div
          v-for="(member, index) in members"
          :key="member.key"
          class="flex items-start gap-2 rounded-[7px] px-2 py-1"
          data-test="preview-team-member"
          :data-state="team.members[index]"
        >
          <span class="pt-0.5" :style="{ color: MEMBER_STATES[team.members[index]].color }">
            <ClientIcon
              :name="MEMBER_STATES[team.members[index]].icon"
              class="h-[13px] w-[13px]"
              :class="team.members[index] === 'working' ? 'cw-spin' : ''"
            />
          </span>
          <div class="flex min-w-0 flex-1 flex-col gap-px">
            <div class="flex items-center gap-1.5">
              <span class="text-[12.5px] font-medium text-[color:var(--cw-text)]">{{ member.name }}</span>
              <span class="text-[11.5px]" :style="{ color: MEMBER_STATES[team.members[index]].color }">
                {{ memberStateLabel(team.members[index]) }}
              </span>
            </div>
            <div class="truncate text-[11.5px] text-[color:var(--cw-text-tertiary)]">{{ member.detail }}</div>
            <div v-if="member.current" class="truncate text-[11.5px] text-[color:var(--cw-text-secondary)]">
              {{ member.current }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-col gap-1.5">
      <div class="text-[11.5px] font-medium text-[color:var(--cw-text-tertiary)]">{{ t('home.clientWorkflow.team.tasks') }}</div>
      <!-- 任务关系图：按依赖深度分列，连线从上一列右边出发；未完成的依赖画虚线 -->
      <div class="cw-graph rounded-[10px] p-2.5">
        <div class="relative h-[90px]">
          <svg class="absolute inset-0 h-full w-full overflow-visible" viewBox="0 0 100 90" preserveAspectRatio="none" aria-hidden="true">
            <path
              v-for="edge in edges"
              :key="edge.key"
              :d="edge.d"
              fill="none"
              stroke-width="1.5"
              vector-effect="non-scaling-stroke"
              :stroke="edge.done ? 'var(--cw-border-strong)' : 'var(--cw-text-ghost)'"
              :stroke-dasharray="edge.done ? undefined : '4 4'"
            />
          </svg>
          <div
            v-for="node in nodes"
            :key="node.task"
            class="cw-node absolute flex h-10 flex-col justify-center rounded-[7px] border px-[7px] transition-colors duration-300 motion-reduce:transition-none"
            :style="{ left: `${node.x}%`, top: `${node.y}px`, width: `${NODE_W}%`, borderColor: TASK_STATES[team.tasks[node.task]].color }"
            data-test="preview-team-task"
            :data-state="team.tasks[node.task]"
          >
            <div class="flex min-w-0 items-center gap-1">
              <span class="shrink-0" :style="{ color: TASK_STATES[team.tasks[node.task]].color }">
                <ClientIcon
                  :name="TASK_STATES[team.tasks[node.task]].icon"
                  class="h-2.5 w-2.5"
                  :class="team.tasks[node.task] === 'running' ? 'cw-spin' : ''"
                />
              </span>
              <span class="min-w-0 truncate text-[11.5px] text-[color:var(--cw-text)]">T{{ node.task + 1 }} {{ subject(node.task) }}</span>
            </div>
            <div class="flex min-w-0 items-center gap-1 text-[10.5px]">
              <span class="min-w-0 truncate text-[color:var(--cw-text-tertiary)]">{{ members[TEAM_TASK_OWNERS[node.task]].name }}</span>
              <span
                v-if="team.rounds[node.task] > 1"
                class="shrink-0 text-[color:var(--cw-warning)]"
                data-test="preview-team-round"
              >{{ t('home.clientWorkflow.team.taskRound', { round: team.rounds[node.task] }) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import type { ClientIconName } from './icons'
import {
  TEAM_TASK_DEPS,
  TEAM_TASK_OWNERS,
  type TeamFrame,
  type TeamMemberState,
  type TeamTaskState,
} from './timeline'

const props = defineProps<{
  team: TeamFrame | null
}>()

const { t } = useI18n()

// 颜色和图标同客户端 team_panel.rs 的 phase_badge / member_status / task_state
const TASK_STATES: Record<TeamTaskState, { icon: ClientIconName; color: string; label: string }> = {
  blocked: { icon: 'lock', color: 'var(--cw-text-ghost)', label: 'taskBlocked' },
  open: { icon: 'queue', color: 'var(--cw-text-secondary)', label: 'taskOpen' },
  running: { icon: 'loaderCircle', color: 'var(--cw-accent)', label: 'taskRunning' },
  completed: { icon: 'check', color: 'var(--cw-success)', label: 'taskCompleted' },
}

const MEMBER_STATES: Record<TeamMemberState, { icon: ClientIconName; color: string; label: string }> = {
  waiting: { icon: 'bot', color: 'var(--cw-text-tertiary)', label: 'memberWaiting' },
  working: { icon: 'loaderCircle', color: 'var(--cw-accent)', label: 'memberWorking' },
  idle: { icon: 'check', color: 'var(--cw-text-secondary)', label: 'memberIdle' },
}

// 每个成员用自己的模型和推理强度
const MEMBER_ROUTES = [
  { key: 'architect', model: 'Claude Opus 5.5', effort: 'effortHigh' },
  { key: 'builder', model: 'GPT-6.1 Sol', effort: 'effortMedium' },
  { key: 'tester', model: 'Claude Sonnet 5.5', effort: 'effortMedium' },
] as const

function memberStateLabel(state: TeamMemberState): string {
  return t(`home.clientWorkflow.team.${MEMBER_STATES[state].label}`)
}

function subject(task: number): string {
  return t(`home.clientWorkflow.team.taskSubjects.t${task + 1}`)
}

const phase = computed(() => {
  switch (props.team?.phase) {
    case 'running':
      return { icon: 'loaderCircle' as ClientIconName, color: 'var(--cw-accent)', label: t('home.clientWorkflow.team.phaseRunning') }
    case 'done':
      return { icon: 'check' as ClientIconName, color: 'var(--cw-success)', label: t('home.clientWorkflow.team.phaseDone') }
    default:
      return { icon: 'list' as ClientIconName, color: 'var(--cw-accent)', label: t('home.clientWorkflow.team.phaseReview') }
  }
})

const stats = computed(() => {
  const team = props.team
  if (!team) return ''
  return t('home.clientWorkflow.team.stats', {
    members: team.members.length,
    working: team.members.filter((state) => state === 'working').length,
    done: team.tasks.filter((state) => state === 'completed').length,
    tasks: team.tasks.length,
    messages: team.messages,
  })
})

// 「进行中 1 · 就绪 1 · 被阻塞 1 · 已完成 1」，顺序同客户端
const progressWords = computed(() => {
  const team = props.team
  if (!team) return ''
  const order: TeamTaskState[] = ['running', 'open', 'blocked', 'completed']
  return order
    .map((state) => [state, team.tasks.filter((value) => value === state).length] as const)
    .filter(([, count]) => count > 0)
    .map(([state, count]) => `${t(`home.clientWorkflow.team.${TASK_STATES[state].label}`)} ${count}`)
    .join(' · ')
})

const members = computed(() =>
  MEMBER_ROUTES.map((route, index) => {
    const team = props.team
    const owned = TEAM_TASK_OWNERS.map((owner, task) => ({ owner, task })).filter((entry) => entry.owner === index)
    const done = team ? owned.filter((entry) => team.tasks[entry.task] === 'completed').length : 0
    const current = team?.memberTask[index]
    return {
      key: route.key,
      name: t(`home.clientWorkflow.team.memberNames.${route.key}`),
      detail: [
        route.model,
        t(`home.clientWorkflow.team.${route.effort}`),
        t('home.clientWorkflow.team.memberTasks', { done, total: owned.length }),
      ].join(' · '),
      current:
        current === null || current === undefined
          ? ''
          : t('home.clientWorkflow.team.memberCurrent', { task: `T${current + 1} ${subject(current)}` }),
    }
  }),
)

// 布局同客户端 compact_dag_layout：列是依赖深度，同一列按编号排；这里按百分比排三列
const NODE_W = 28.5
const COL_GAP = (100 - NODE_W * 3) / 2
const NODE_H = 40
const ROW_GAP = 10

function depth(task: number): number {
  const deps = TEAM_TASK_DEPS[task]
  return deps.length === 0 ? 0 : 1 + Math.max(...deps.map(depth))
}

const nodes = computed(() => {
  const rows = new Map<number, number>()
  return TEAM_TASK_DEPS.map((_, task) => {
    const column = depth(task)
    const row = rows.get(column) ?? 0
    rows.set(column, row + 1)
    return { task, x: column * (NODE_W + COL_GAP), y: row * (NODE_H + ROW_GAP) }
  })
})

const edges = computed(() => {
  const team = props.team
  const bend = COL_GAP * 0.55
  return TEAM_TASK_DEPS.flatMap((deps, task) =>
    deps.map((dep) => {
      const from = nodes.value[dep]
      const to = nodes.value[task]
      const start = [from.x + NODE_W, from.y + NODE_H / 2]
      const end = [to.x, to.y + NODE_H / 2]
      return {
        key: `${dep}-${task}`,
        done: team?.tasks[dep] === 'completed',
        d: `M ${start[0]} ${start[1]} C ${start[0] + bend} ${start[1]}, ${end[0] - bend} ${end[1]}, ${end[0]} ${end[1]}`,
      }
    }),
  )
})
</script>

<style scoped>
.cw-graph {
  border: 1px solid var(--cw-border);
  background: var(--cw-inset);
}

.cw-node {
  background: var(--cw-raised);
}

/* 盖在对话上的窄面板里放不下进度说明，只留色条 */
@container cw-main (max-width: 699px) {
  .cw-team-words {
    display: none;
  }
}
</style>
