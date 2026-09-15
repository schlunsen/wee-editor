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
