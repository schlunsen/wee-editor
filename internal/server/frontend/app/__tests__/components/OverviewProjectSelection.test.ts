import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from '../../stores/session/sessionStore'
import { useJustRecipes } from '../../composables/useJustRecipes'
import { openOverviewAgent } from '../../utils/agents/openOverviewAgent'
import ProjectSelector from '../../components/ProjectSelector.vue'
import JustCommandPalette from '../../components/just/JustCommandPalette.vue'

beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()) })
afterEach(() => vi.unstubAllGlobals())

it('activates the project picker and loads Just commands for the overview agent’s project', async () => {
  const project = { id: 'target', name: 'Target project', path: '/target', is_active: true }
  const fetchWithAuth = vi.fn(async (url: string) => ({ ok: true, json: async () =>
    url.endsWith('/just/recipes') ? { recipes: [{ name: 'build', description: 'Build' }] } :
    url === '/api/projects' ? { projects: [project] } : { project }
  }) as Response)
  vi.stubGlobal('useAuthenticatedFetch', () => ({ fetchWithAuth }))
  vi.stubGlobal('useJustRecipes', useJustRecipes)
  const store = useSessionStore()
  store.setSelectedProject({ id: 'other', name: 'Other project', path: '/other' } as any)
  const picker = mount(ProjectSelector)
  const palette = mount(JustCommandPalette, { global: { stubs: { Teleport: true, Icon: true } } })
  const just = useJustRecipes()
  try {
    await flushPromises()
    const push = vi.fn(async () => {
      expect(store.selectedProject?.name).toBe('Target project')
      expect(store.selectedProject?.path).toBe('/target')
    })
    await openOverviewAgent({ id: 'agent', status: 'processing', project_id: 'target' }, { store, router: { push } as any, fetchWithAuth })
    await flushPromises()
    expect(picker.get('.current-project').text()).toBe('Target project')
    just.openPalette()
    await flushPromises()
    expect(fetchWithAuth).toHaveBeenCalledWith('/api/projects/target/just/recipes')
    expect(palette.text()).toContain('build')
  } finally { just.closePalette(); picker.unmount(); palette.unmount() }
})
