import type { Router } from 'vue-router'
import type { useSessionStore } from '../../stores/session/sessionStore'
import type { OverviewAgent } from '../../composables/agents/useAgentOverview'

export async function openOverviewAgent(agent: OverviewAgent, {
  store, router, fetchWithAuth
}: {
  store: ReturnType<typeof useSessionStore>
  router: Pick<Router, 'push'>
  fetchWithAuth: (url: string) => Promise<Response>
}) {
  // Activate the complete project exactly as the project picker does, before
  // leaving the overview. URL watchers alone cannot establish editor state.
  if (agent.project_id) {
    const response = await fetchWithAuth(`/api/projects/${agent.project_id}`)
    if (!response.ok) throw new Error('Could not load this agent’s project. Please try again.')
    const { project } = await response.json()
    if (project?.id !== agent.project_id || !project.name || !project.path) {
      throw new Error('This agent’s project is unavailable. Please refresh and try again.')
    }
    store.setSelectedProject(project)
  } else {
    store.setSelectedProject(null)
  }
  await router.push({ path: '/agents', query: { ...(agent.project_id ? { project: agent.project_id } : {}), session: agent.id } })
}
