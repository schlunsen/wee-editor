import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import AgentOverview from '../../components/agents/AgentOverview.vue'

vi.mock('../../composables/agents/useAgentOverview', () => ({
  useAgentOverview: () => ({
    agents: ref([
      { id: 'agent-one', status: 'processing', project_id: 'one', selected_avatar: { name: 'First' } },
      { id: 'agent-two', status: 'processing', project_id: 'two', selected_avatar: { name: 'Second' } }
    ]),
    projects: ref({ one: { name: 'Project one' }, two: { name: 'Project two' } }),
    details: ref({}), loading: ref(false), error: ref(''), projectCount: ref(2), contextFor: () => undefined,
    filter: ref('all'), recentHours: ref(24), recentLimit: ref(12),
    counts: ref({ all: 2, working: 2, attention: 0, recent: 0, unreviewed: 0 }),
    needsReview: () => false, markReviewed: vi.fn(), hiddenRecentCount: ref(0)
  })
}))

it('shows both projects and opens the chosen agent without treating log clicks as navigation', async () => {
  const wrapper = mount(AgentOverview)
  expect(wrapper.findAll('.agent-column')).toHaveLength(2)
  expect(wrapper.text()).toContain('Project two')
  await wrapper.findAll('.activity-log')[1].trigger('click')
  expect(wrapper.emitted('select')).toBeUndefined()
  await wrapper.get('button[aria-label="Open agent Second"]').trigger('click')
  expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ id: 'agent-two', status: 'processing', project_id: 'two' })
  wrapper.unmount()
})
