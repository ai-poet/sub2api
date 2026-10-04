import { describe, expect, it } from 'vitest'
import {
  CHAPTERS,
  CYCLE_MS,
  EXEC_ROW_COUNT,
  FADE_IN_MS,
  FADE_OUT_MS,
  MOMENTS,
  PLAN_ROW_COUNT,
  STATIC_T,
  SUBAGENT_COUNT,
  TEAM_TASK_DEPS,
  TEAM_TASK_OWNERS,
  chapterIndexAt,
  chapterProgressAt,
  deriveFrame,
  formatTokens,
  settleT,
} from '../timeline'

const m = MOMENTS

function samples(from: number, to: number, step = 100): number[] {
  const out: number[] = []
  for (let t = from; t < to; t += step) out.push(t)
  return out
}

describe('client preview timeline: chapters', () => {
  it('covers the whole loop with three back-to-back chapters', () => {
    expect(CHAPTERS.map((chapter) => chapter.id)).toEqual(['plan', 'team', 'image'])
    expect(CHAPTERS[0].start).toBe(0)
    expect(CHAPTERS[CHAPTERS.length - 1].end).toBe(CYCLE_MS)
    for (let i = 1; i < CHAPTERS.length; i++) {
      expect(CHAPTERS[i].start).toBe(CHAPTERS[i - 1].end)
    }
  })

  it('changes chapter exactly at each boundary', () => {
    CHAPTERS.forEach((chapter, index) => {
      expect(deriveFrame(chapter.start).chapter).toBe(chapter.id)
      expect(chapterIndexAt(chapter.start)).toBe(index)
      expect(deriveFrame(chapter.end - 1).chapter).toBe(chapter.id)
    })
  })

  it('fades only in the first and last moments of a chapter', () => {
    for (const chapter of CHAPTERS) {
      expect(deriveFrame(chapter.start).fading).toBe(true)
      expect(deriveFrame(chapter.start + FADE_IN_MS - 1).fading).toBe(true)
      expect(deriveFrame(chapter.start + FADE_IN_MS).fading).toBe(false)
      expect(deriveFrame(chapter.end - FADE_OUT_MS - 1).fading).toBe(false)
      expect(deriveFrame(chapter.end - FADE_OUT_MS).fading).toBe(true)
    }
  })

  it('puts each still frame inside its chapter, fully visible and at rest', () => {
    for (const chapter of CHAPTERS) {
      const frame = deriveFrame(chapter.staticT)
      expect(frame.chapter).toBe(chapter.id)
      expect(frame.fading).toBe(false)
      expect(frame.pickerOpen).toBe(false)
      expect(frame.planApprovePressed).toBe(false)
      expect(frame.team?.approvePressed ?? false).toBe(false)
    }
    expect(STATIC_T).toBe(CHAPTERS[0].staticT)
  })

  it('shows the plan waiting for approval in the first still frame', () => {
    expect(deriveFrame(CHAPTERS[0].staticT)).toMatchObject({
      plan: 'pending',
      planCard: true,
      panel: 'plan',
      sheet: 'plan',
      status: 'waitingForYou',
      planRows: PLAN_ROW_COUNT,
    })
  })

  it('shows the team mid-run with a task on its second round in the second still frame', () => {
    const team = deriveFrame(CHAPTERS[1].staticT).team
    expect(team?.phase).toBe('running')
    expect(team?.rounds[1]).toBe(2)
    expect(team?.tasks.filter((state) => state === 'completed')).toHaveLength(1)
  })

  it('shows the finished picture in the third still frame', () => {
    expect(deriveFrame(CHAPTERS[2].staticT)).toMatchObject({ scene: 'image', imageJob: 'done', charges: 3 })
  })

  it('reports progress through the current chapter', () => {
    expect(chapterProgressAt(0)).toBe(0)
    expect(chapterProgressAt(CHAPTERS[0].end / 2)).toBeCloseTo(0.5)
    expect(chapterProgressAt(CHAPTERS[1].start)).toBe(0)
    expect(chapterProgressAt(CYCLE_MS - 1)).toBeCloseTo(1, 3)
  })

  it('settles a paused time onto a visible frame', () => {
    expect(settleT(0)).toBe(FADE_IN_MS)
    expect(deriveFrame(settleT(CHAPTERS[0].end - 100)).fading).toBe(false)
    expect(deriveFrame(settleT(CHAPTERS[0].end - 100)).runDone).toBe(true)
    expect(settleT(5000)).toBe(5000)
  })

  it('loops back to the start and accepts negative time', () => {
    expect(deriveFrame(CYCLE_MS + 100)).toEqual(deriveFrame(100))
    expect(deriveFrame(-100)).toEqual(deriveFrame(CYCLE_MS - 100))
  })
})

describe('client preview timeline: plan → run', () => {
  it('opens on a draft in plan mode with the model picker showing briefly', () => {
    expect(deriveFrame(0)).toMatchObject({ draft: true, sent: false, mode: 'plan', status: null, pickerOpen: false })
    expect(deriveFrame(m.pickerOpen).pickerOpen).toBe(true)
    expect(deriveFrame(m.pickerClose - 1).pickerOpen).toBe(true)
    expect(deriveFrame(m.pickerClose).pickerOpen).toBe(false)
  })

  it('sends, plans in three rows and walks the status line', () => {
    expect(deriveFrame(m.send)).toMatchObject({ draft: false, sent: true, status: 'waitingForModel', planRows: 0 })
    expect(deriveFrame(m.planRows[0])).toMatchObject({ planRows: 1, status: 'runningTool' })
    expect(deriveFrame(m.planRows[1])).toMatchObject({ planRows: 2, status: 'thinking' })
    expect(deriveFrame(m.planRows[2])).toMatchObject({ planRows: 3, status: 'writing' })
  })

  it('hands over the plan and waits for approval with the token count held', () => {
    const ready = deriveFrame(m.planReady)
    expect(ready).toMatchObject({ plan: 'pending', planCard: true, panel: 'plan', sheet: 'plan', status: 'waitingForYou' })
    expect(deriveFrame(m.approved - 1).turnTokens).toBe(ready.turnTokens)
    expect(deriveFrame(m.approvePress).planApprovePressed).toBe(true)
    expect(deriveFrame(m.approvePress - 1).planApprovePressed).toBe(false)
  })

  it('runs after approval with two sub-agents in parallel and the panel kept beside it', () => {
    const approved = deriveFrame(m.approved)
    expect(approved).toMatchObject({ plan: 'approved', planCard: false, panel: 'plan', sheet: 'none', mode: 'build' })

    const parallel = deriveFrame(m.subagentRows[1])
    expect(parallel).toMatchObject({
      subagentRows: SUBAGENT_COUNT,
      subagentsRunning: true,
      backgroundLive: true,
      status: 'runningTools',
      statusTools: SUBAGENT_COUNT,
    })

    expect(deriveFrame(m.subagentsDone)).toMatchObject({ subagentsRunning: false, backgroundLive: false })
  })

  it('edits and tests, then settles the turn and charges once', () => {
    expect(deriveFrame(m.execRows[0])).toMatchObject({ execRows: 1, status: 'runningTool' })
    expect(deriveFrame(m.execRows[2]).execRows).toBe(EXEC_ROW_COUNT)

    const done = deriveFrame(m.runDone)
    expect(done).toMatchObject({ runDone: true, status: null, charges: 1, workSeconds: 31 })
    expect(deriveFrame(m.teamSessionCue - 100).workSeconds).toBe(done.workSeconds)
    expect(deriveFrame(m.teamSessionCue).activeSession).toBe('team')
  })

  it('never lets the token count go down', () => {
    let last = 0
    for (const t of samples(0, CHAPTERS[0].end)) {
      const tokens = deriveFrame(t).turnTokens
      expect(tokens).toBeGreaterThanOrEqual(last)
      last = tokens
    }
    expect(last).toBeGreaterThan(1000)
  })
})

describe('client preview timeline: agent team', () => {
  it('starts with the team plan waiting in the panel', () => {
    const frame = deriveFrame(CHAPTERS[1].start)
    expect(frame).toMatchObject({ scene: 'agent', panel: 'team', sheet: 'team', teamReply: 'draft', plan: 'none' })
    expect(frame.team).toMatchObject({ phase: 'review', messages: 0 })
    expect(frame.team?.members).toEqual(['waiting', 'waiting', 'waiting'])
    expect(deriveFrame(m.teamApprovePress).team?.approvePressed).toBe(true)
  })

  it('runs every task only after the tasks it waits on are done', () => {
    for (const t of samples(CHAPTERS[1].start, CHAPTERS[1].end)) {
      const team = deriveFrame(t).team!
      team.tasks.forEach((state, task) => {
        if (state === 'running' || state === 'completed') {
          for (const dep of TEAM_TASK_DEPS[task]) expect(team.tasks[dep]).toBe('completed')
        }
      })
      team.memberTask.forEach((task, member) => {
        if (task === null) return
        expect(team.tasks[task]).toBe('running')
        expect(TEAM_TASK_OWNERS[task]).toBe(member)
        expect(team.members[member]).toBe('working')
      })
    }
  })

  it('sends one task back for a second round', () => {
    expect(deriveFrame(m.teamRound2 - 100).team?.rounds[1]).toBe(1)
    expect(deriveFrame(m.teamRound2).team?.rounds[1]).toBe(2)
  })

  it('finishes with every task done, a closing reply and a second charge', () => {
    const done = deriveFrame(m.teamDone)
    expect(done.team?.phase).toBe('done')
    expect(done.team?.tasks.every((state) => state === 'completed')).toBe(true)
    expect(done).toMatchObject({ teamReply: 'done', sheet: 'none', panel: 'team', charges: 2 })
    expect(deriveFrame(m.imageRowCue)).toMatchObject({ imageRowActive: true, activeSession: null })
  })

  it('counts messages up as the team works', () => {
    let last = 0
    for (const t of samples(CHAPTERS[1].start, CHAPTERS[1].end)) {
      const messages = deriveFrame(t).team!.messages
      expect(messages).toBeGreaterThanOrEqual(last)
      last = messages
    }
  })
})

describe('client preview timeline: images', () => {
  it('switches to the image studio, types, draws and charges a third time', () => {
    expect(deriveFrame(CHAPTERS[2].start)).toMatchObject({ scene: 'image', panel: 'none', team: null, imageRowActive: true })

    const typing = deriveFrame((m.typeStart + m.typeEnd) / 2)
    expect(typing.typed).toBeGreaterThan(0)
    expect(typing.typed).toBeLessThan(1)
    expect(typing.imageJob).toBe('idle')

    const drawing = deriveFrame(m.imageSubmit + 1000)
    expect(drawing).toMatchObject({ imageJob: 'drawing', submitted: true, charges: 2, drawSeconds: 1 })

    expect(deriveFrame(m.imageDone)).toMatchObject({ imageJob: 'done', charges: 3 })
  })
})

describe('formatTokens', () => {
  it('matches the client count format', () => {
    expect(formatTokens(0)).toBe('0')
    expect(formatTokens(950)).toBe('950')
    expect(formatTokens(1000)).toBe('1.0k')
    expect(formatTokens(3300)).toBe('3.3k')
  })
})
