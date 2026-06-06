// 侧边栏 workspaces 静态数据，被 sidebar 与父组件共享：
// 父组件用它把当前激活的工作区名/diff 渲染到顶栏。
export interface WorkspaceDiff {
  additions: string
  deletions: string
}

export interface Workspace {
  name: string
  statusClass: 'active' | 'muted' | 'ready' | 'creating'
  attention?: boolean
  script?: boolean
  creating?: boolean
  persona?: string
  summary?: string
  diff?: WorkspaceDiff
}

export const workspaces: Workspace[] = [
  {
    name: 'homepage-billing',
    statusClass: 'active',
    attention: true,
    script: true,
    diff: { additions: '+42', deletions: '-8' },
  },
  {
    name: 'pricing-copy',
    statusClass: 'muted',
    persona: 'Reviewer',
    summary: '3 skills',
    diff: { additions: '+12', deletions: '-2' },
  },
  {
    name: 'api-metering',
    statusClass: 'ready',
    persona: 'Builder',
    summary: '2 skills',
    diff: { additions: '+28', deletions: '-4' },
  },
  {
    name: 'docs-worktree',
    statusClass: 'muted',
    persona: 'Writer',
    summary: '1 skill',
    diff: { additions: '+6', deletions: '-1' },
  },
  {
    name: 'new-agent-flow',
    statusClass: 'creating',
    creating: true,
  },
]

// 仅用于轮转 / 选中的工作区（排除 creating 占位）。
export const selectableWorkspaces = workspaces.filter((w) => !w.creating)
