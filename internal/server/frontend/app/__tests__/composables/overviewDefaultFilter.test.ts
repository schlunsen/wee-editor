import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { useAgentOverview } from '../../composables/agents/useAgentOverview'

const auth = vi.hoisted(() => ({ fetchWithAuth: vi.fn() }))
vi.mock('../../composables/useAuthenticatedFetch', () => ({ useAuthenticatedFetch: () => auth }))

beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()) })
afterEach(() => vi.unstubAllGlobals())

const sessions = [
  { id: 'running', status: 'processing', project_id: 'p', updated_at: new Date().toISOString(), message_count: 4 },
  { id: 'finished', status: 'idle', project_id: 'p', updated_at: new Date().toISOString(), message_count: 6, num_turns: 2 },
  { id: 'blocked', status: 'idle', project_id: 'p', updated_at: new Date().toISOString(), message_count: 2, pending_permissions: 1 }
]

function mountOverview() {
  auth.fetchWithAuth.mockImplementation(async (url: string) => ({ ok: true, json: async () =>
    url.startsWith('/api/agent/sessions') ? { sessions } :
    url === '/api/projects' ? { projects: [{ id: 'p', name: 'Project', path: '/p' }] } : {}
  }) as Response)
  const api: any = {}
  const wrapper = mount(defineComponent({
    setup() { Object.assign(api, useAgentOverview()); return () => h('div') }
  }))
  return { wrapper, api }
}

it('opens on the running agents instead of every recent session', async () => {
  const { wrapper, api } = mountOverview()
  try {
    await flushPromises()
    expect(api.filter.value).toBe('working')
    expect(api.agents.value.map((a: any) => a.id)).toEqual(['running'])
    expect(api.counts.value).toMatchObject({ all: 3, working: 1, attention: 1, recent: 1 })
  } finally { wrapper.unmount() }
})

it('still reaches the other agents when the filter changes', async () => {
  const { wrapper, api } = mountOverview()
  try {
    await flushPromises()
    api.filter.value = 'attention'
    await flushPromises()
    expect(api.agents.value.map((a: any) => a.id)).toEqual(['blocked'])
    api.filter.value = 'all'
    await flushPromises()
    expect(api.agents.value.map((a: any) => a.id).sort()).toEqual(['blocked', 'finished', 'running'])
  } finally { wrapper.unmount() }
})
