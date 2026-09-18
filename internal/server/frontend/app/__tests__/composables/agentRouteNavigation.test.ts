import { computed, effectScope, reactive, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia, storeToRefs } from 'pinia'
import { useSessionStore } from '../../stores/session/sessionStore'
import { useSessionActions } from '../../composables/agents/useSessionActions'
import { useAgentRouteNavigation } from '../../composables/agents/useAgentRouteNavigation'

beforeEach(() => {
  localStorage.clear(); setActivePinia(createPinia())
  vi.stubGlobal('useAuthenticatedFetch', () => ({ fetchWithAuth: async () => ({ ok: true, json: async () => ({}) }) }))
})
afterEach(() => vi.unstubAllGlobals())

function setupNavigation(fetchProject?: (url: string) => Promise<Response>) {
  const store = useSessionStore()
  store.setSelectedProject({ id: 'old-project', name: 'Old project' } as any)
  const route = reactive({ path: '/agents', query: { project: 'target-project', session: 'target-agent' } })
  const connected = ref(true)
  const project = { id: 'target-project', name: 'Target project', path: '/target' }
  const fetchWithAuth = vi.fn(fetchProject || (async () => ({ ok: true, json: async () => ({ project }) }) as Response))
  const send = vi.fn(() => true)
  const selection = useSessionActions({
    agentWs: { get connected() { return connected.value }, send },
    ...storeToRefs(store),
    activeSessionId: computed({ get: () => store.activeSessionId, set: id => store.setActiveSession(id) }),
    isUserNearBottom: ref(true), scrollToBottom: vi.fn(), focusMessageInput: vi.fn()
  } as any)
  const selectSession = vi.fn(async (id: string) => {
    // This callback is the editor's subscription/history-loading entry point.
    expect(store.selectedProject?.id).toBe('target-project')
    await selection.selectSession(id)
  })
  const replace = vi.fn(async ({ query }: any) => { route.query = query })
  const scope = effectScope()
  scope.run(() => useAgentRouteNavigation({ route: route as any, router: { replace } as any,
    sessionStore: store, connected: () => connected.value, selectSession, fetchWithAuth }))
  return { store, route, connected, project, fetchWithAuth, selectSession, replace, scope, send }
}

it('unwraps the project API response and selects the linked conversation after sessions arrive', async () => {
  const state = setupNavigation()
  try {
    await flushPromises()
    expect(state.store.selectedProject).toEqual(state.project)
    expect(state.selectSession).not.toHaveBeenCalled()
    state.store.sessions = [{ id: 'target-agent', project_id: 'target-project', status: 'processing', options: {} }] as any
    await flushPromises()
    expect(state.selectSession).toHaveBeenCalledExactlyOnceWith('target-agent')
    expect(state.store.activeSessionId).toBe('target-agent')
    expect(state.send).toHaveBeenCalledWith({ type: 'subscribe_session', session_id: 'target-agent' })
    expect(state.send).toHaveBeenCalledWith({ type: 'load_messages', session_id: 'target-agent', limit: 200, offset: 0 })
    expect(state.route.query).toEqual({ project: 'target-project' })
    expect(JSON.parse(localStorage.getItem('selectedProjectObj')!)).toEqual(state.project)
  } finally { state.scope.stop() }
})

it('waits for the history connection before selecting and handles another click in the same project', async () => {
  const state = setupNavigation()
  try {
    state.connected.value = false
    state.store.sessions = ['target-agent', 'second-agent'].map(id => ({ id, project_id: 'target-project', status: 'processing', options: {} })) as any
    await flushPromises()
    expect(state.selectSession).not.toHaveBeenCalled()
    state.connected.value = true
    await flushPromises()
    expect(state.selectSession).toHaveBeenCalledExactlyOnceWith('target-agent')
    state.route.query = { project: 'target-project', session: 'second-agent' }
    await flushPromises()
    expect(state.store.activeSessionId).toBe('second-agent')
    expect(state.selectSession).toHaveBeenCalledTimes(2)
  } finally { state.scope.stop() }
})

it('preserves selected history and project when a session-list response arrives after navigation', async () => {
  const state = setupNavigation()
  try {
    state.store.sessions = [{ id: 'target-agent', project_id: 'target-project', status: 'processing', options: {} }] as any
    await flushPromises()
    state.store.messages['target-agent'] = [{ id: 'message', content: 'Existing conversation', role: 'assistant' }] as any
    state.store.messagesLoaded.add('target-agent')
    state.store.refreshSessions([{ ...state.store.sessions[0], message_count: 4 }])
    await flushPromises()
    expect(state.store.selectedProject?.id).toBe('target-project')
    expect(state.store.activeSessionId).toBe('target-agent')
    expect(state.store.messages['target-agent'][0].content).toBe('Existing conversation')
    expect(state.store.messagesLoaded.has('target-agent')).toBe(true)
    expect(state.selectSession).toHaveBeenCalledTimes(1)
  } finally { state.scope.stop() }
})

it('ignores a slow project response after the user navigates to another project', async () => {
  const responses = new Map<string, (response: Response) => void>()
  const state = setupNavigation(url => new Promise(resolve => responses.set(url, resolve)))
  try {
    state.route.query = { project: 'newer-project', session: 'newer-agent' }
    state.store.sessions = [{ id: 'newer-agent', project_id: 'newer-project', status: 'processing', options: {} }] as any
    // Keep selection pending so this test isolates out-of-order project replies.
    state.connected.value = false
    await flushPromises()
    responses.get('/api/projects/newer-project')!({ ok: true, json: async () => ({ project: { id: 'newer-project', name: 'Newer' } }) } as Response)
    await flushPromises()
    responses.get('/api/projects/target-project')!({ ok: true, json: async () => ({ project: state.project }) } as Response)
    await flushPromises()
    expect(state.store.selectedProject?.id).toBe('newer-project')
    expect(state.store.activeSessionId).toBe('newer-agent')
  } finally { state.scope.stop() }
})

it('switches back when the navbar changed projects but the URL still names the previous project', async () => {
  const state = setupNavigation()
  try {
    state.store.sessions = ['target-agent', 'second-agent'].map(id => ({ id, project_id: 'target-project', status: 'processing', options: {} })) as any
    await flushPromises()
    state.store.setSelectedProject({ id: 'manual-project', name: 'Manually selected' } as any)
    // The project query stays the same, only the overview's session link changes.
    state.route.query = { project: 'target-project', session: 'second-agent' }
    await flushPromises()
    expect(state.store.selectedProject?.id).toBe('target-project')
    expect(state.store.activeSessionId).toBe('second-agent')
    expect(state.send).toHaveBeenCalledWith({ type: 'load_messages', session_id: 'second-agent', limit: 200, offset: 0 })
  } finally { state.scope.stop() }
})
