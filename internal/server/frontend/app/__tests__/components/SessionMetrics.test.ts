import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SessionMetrics from '../../components/SessionMetrics.vue'
import { useUIStore } from '../../stores/ui/uiStore'

vi.mock('../../stores/projects/projectSubscriptionsStore', () => ({ useProjectSubscriptionsStore: () => ({ getProjectGitStatus: () => null, subscribeToProject: vi.fn(), unsubscribeFromProject: vi.fn() }) }))

beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()) })

it('defaults to compact sections and counts executions rather than tool names', async () => {
  const wrapper = mount(SessionMetrics, { props: { session: { id: 'test', status: 'idle', message_count: 51, options: {} }, toolExecutions: { Bash: 4, Read: 2 } }, global: { stubs: { Icon: true, ProjectPermissions: true, ContextUsageBar: true, GitStatus: true } } })
  const store = useUIStore()
  expect(store.sectionOrder).toEqual(['sessionInfo', 'gitStatus', 'toolsPermissions', 'activity'])
  expect(store.expandedSections).toMatchObject({ sessionInfo: true, gitStatus: true, toolsPermissions: false, activity: false })
  const activity = wrapper.findAll('button').find(button => button.text().includes('Activity'))!
  await activity.trigger('click')
  expect(activity.attributes('aria-expanded')).toBe('true')
  expect(wrapper.get('.tools-metric .metric-value').text()).toBe('6')
  wrapper.unmount()
})

it('preserves saved sections and supplies defaults for the new activity section', () => {
  localStorage.setItem('cct:ui', JSON.stringify({ version: 1, data: { sectionOrder: ['gitStatus', 'sessionInfo', 'toolsPermissions'], expandedSections: { sessionInfo: false, gitStatus: true } } }))
  const store = useUIStore()
  expect(store.expandedSections.sessionInfo).toBe(false)
  expect(store.expandedSections.activity).toBe(false)
  expect(store.expandedSections.toolsPermissions).toBe(false)
})

it('keeps approval skipping off when confirmation is cancelled', async () => {
  const send = vi.fn()
  const originalConfirm = window.confirm
  const confirm = vi.fn(() => false)
  window.confirm = confirm
  const wrapper = mount(SessionMetrics, { props: { session: { id: 'test', status: 'idle', message_count: 0, options: {} } }, global: { provide: { agentWs: { send, off: vi.fn() } }, stubs: { Icon: true, ProjectPermissions: true, ContextUsageBar: true, GitStatus: true } } })
  const checkbox = wrapper.get<HTMLInputElement>('input[aria-label="YOLO Mode"]')
  await checkbox.setValue(true)
  expect(confirm).toHaveBeenCalledOnce()
  expect(send).not.toHaveBeenCalled()
  expect(checkbox.element.checked).toBe(false)
  wrapper.unmount()
  window.confirm = originalConfirm
})

it('loads the session worktree after it changes instead of using project status', async () => {
  const { flushPromises } = await import('@vue/test-utils')
  let statusCalls = 0
  const fetchMock = vi.fn(async (url: string) => {
    const status = url.endsWith('git-status')
    if (status) statusCalls++
    return { ok: true, json: async () => status ? {
      branch: statusCalls === 1 ? 'main' : 'feature/live',
      worktree_path: statusCalls === 1 ? '' : '/repo/.worktrees/live',
      clean: false, staged: [], modified: ['changed.go'], untracked: [], deleted: []
    } : {} }
  })
  vi.stubGlobal('fetch', fetchMock)
  const session = { id: 'session', project_id: 'project', status: 'processing', message_count: 1, git_branch: 'main', options: { working_directory: '/repo' } }
  const wrapper = mount(SessionMetrics, { props: { session }, global: { stubs: { Icon: true, ProjectPermissions: true, ContextUsageBar: true, GitStatus: true } } })
  await flushPromises()
  expect(wrapper.findComponent({ name: 'GitStatus' }).props('status').branch).toBe('main')
  await wrapper.setProps({ session: { ...session, options: { ...session.options, workspace: { working_directory: '/repo/.worktrees/live', worktree_path: '/repo/.worktrees/live', branch: 'feature/live' } } } })
  await flushPromises()
  const git = wrapper.findComponent({ name: 'GitStatus' })
  expect(git.props('worktreePath')).toBe('/repo/.worktrees/live')
  expect(git.props('status')).toMatchObject({ branch: 'feature/live', modified: ['changed.go'], clean: false })
  wrapper.unmount()
  vi.unstubAllGlobals()
})

it('ignores late Git responses from the previous workspace and clears stale status on errors', async () => {
  const { flushPromises } = await import('@vue/test-utils')
  let finishOld: (value: unknown) => void = () => {}
  const oldResponse = new Promise(resolve => { finishOld = resolve })
  let statusCalls = 0
  vi.stubGlobal('fetch', vi.fn((url: string) => {
    if (!url.endsWith('git-status')) return Promise.resolve({ ok: true, json: async () => ({}) })
    if (++statusCalls === 1) return oldResponse
    if (statusCalls === 2) return Promise.resolve({ ok: true, json: async () => ({ branch: 'feature/new', worktree_path: '/new', clean: true }) })
    return Promise.resolve({ ok: false, json: async () => ({ error: 'Worktree no longer exists' }) })
  }))
  const session = { id: 'session', status: 'processing', message_count: 1, git_branch: 'main', options: { working_directory: '/repo' } }
  const wrapper = mount(SessionMetrics, { props: { session }, global: { stubs: { Icon: true, ProjectPermissions: true, ContextUsageBar: true, GitStatus: true } } })
  await wrapper.setProps({ session: { ...session, options: { ...session.options, workspace: { working_directory: '/new', worktree_path: '/new', branch: 'feature/new' } } } })
  await flushPromises()
  finishOld({ ok: true, json: async () => ({ branch: 'main', worktree_path: '', clean: true }) })
  await flushPromises()
  expect(wrapper.findComponent({ name: 'GitStatus' }).props('status').branch).toBe('feature/new')
  await wrapper.get('button[aria-label="Refresh Git status"]').trigger('click')
  await flushPromises()
  expect(wrapper.findComponent({ name: 'GitStatus' }).exists()).toBe(false)
  expect(wrapper.get('.git-error').text()).toContain('Worktree no longer exists')
  wrapper.unmount()
  vi.unstubAllGlobals()
})
