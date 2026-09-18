import { overviewState, overviewCandidates, markOverviewReviewed, readOverviewReviews, reviewVersion } from '../../utils/agents/overviewState'
const now = Date.parse('2026-09-16T14:00:00Z')
const agent = { id: 'one', status: 'processing', message_count: 10, num_turns: 1, updated_at: new Date(now).toISOString() }
beforeEach(() => localStorage.clear())
it('keeps blockers ahead of working status and distinguishes finished turns from idle sessions', () => {
  expect(overviewState({ ...agent, pending_questions: 1 })).toEqual({ category: 'attention', label: 'Question waiting' })
  expect(overviewState({ ...agent, pending_permissions: 1 }).category).toBe('attention')
  expect(overviewState({ ...agent, status: 'error' }).category).toBe('attention')
  expect(overviewState({ ...agent, status: 'idle' }).label).toBe('Turn finished')
  expect(overviewState({ ...agent, status: 'idle', num_turns: 0 }).label).toBe('Idle')
  expect(overviewState({ ...agent, status: 'ended' }).label).toBe('Stopped')
})
it('retains recent results and every running or blocked agent, but omits old history', () => {
  const old = new Date(now - 48 * 3600000).toISOString()
  const sessions = [agent, { ...agent, id: 'done', status: 'idle' }, { ...agent, id: 'old', status: 'idle', updated_at: old },
    { ...agent, id: 'blocked', pending_permissions: 1, updated_at: old }, { ...agent, id: 'failed', status: 'error', message_count: 0 }]
  expect(overviewCandidates(sessions, now, 24).map(a => a.id)).toEqual(['one', 'done', 'blocked', 'failed'])
  expect(overviewCandidates(sessions, now, 168)).toHaveLength(5)
})
it('marks the current result reviewed locally and flags the next finished turn again', () => {
  const done = { ...agent, status: 'idle' }
  markOverviewReviewed(done)
  expect(readOverviewReviews()[done.id]).toBe(reviewVersion(done))
  expect(readOverviewReviews()[done.id]).not.toBe(reviewVersion({ ...done, num_turns: 2, message_count: 14 }))
  expect(overviewState({ ...agent, pending_permissions: 1 }).category).toBe('attention')
})
