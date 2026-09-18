import { watch } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import type { useSessionStore } from '../../stores/session/sessionStore'

// Resolve the project before selecting a linked session, using the editor's
// normal selection callback to subscribe and load its conversation history.
export function useAgentRouteNavigation({ route, router, sessionStore, connected, selectSession, fetchWithAuth }: {
  route: RouteLocationNormalizedLoaded
  router: Pick<Router, 'replace'>
  sessionStore: ReturnType<typeof useSessionStore>
  connected: () => boolean
  selectSession: (id: string) => Promise<void>
  fetchWithAuth: (url: string) => Promise<Response>
}) {
  // Watch for project query parameter changes in URL
  // Only react when the param actively changes while on the agents page
  // (not when navigating away to a different route)
  watch([() => route.query.project, () => route.query.session], async ([projectId], previous) => {
    const oldProjectId = previous?.[0]
    // Skip if we've navigated away from the agents page — the watcher can fire
    // during route transitions before the component is unmounted
    if (route.path !== '/agents') return

    if (projectId && typeof projectId === 'string') {
      // If the store already has this project selected, skip
      if (sessionStore.selectedProject?.id === projectId) return

      // Fetch the full project object before setting it
      try {
        const response = await fetchWithAuth(`/api/projects/${projectId}`)
        if (response.ok) {
          const { project } = await response.json()
          // Ignore a late response if another navigation has already started.
          if (route.query.project !== projectId || route.path !== '/agents') return
          if (!project || project.id !== projectId) throw new Error('Invalid project response')
          sessionStore.setSelectedProject(project)
        }
      } catch (err) {
        console.error('Failed to fetch project for selector:', err)
      }
    } else if (projectId === undefined && oldProjectId !== undefined) {
      // Only clear if the project param was explicitly removed (had a value before)
      sessionStore.setSelectedProject(null)
    }
  }, { immediate: true })

  // Overview links also work when arriving from another page. Wait until both
  // the session list and its project are ready before using normal selection.
  watch([
    () => route.path === '/agents' ? route.query.session : undefined,
    () => connected(),
    () => sessionStore.sessions.some(s => s.id === route.query.session),
    () => !route.query.project || sessionStore.selectedProject?.id === route.query.project
  ] as const, async ([sessionId, connected, exists, projectReady]) => {
    if (typeof sessionId === 'string' && connected && exists && projectReady) {
      await selectSession(sessionId)
      if (route.query.session === sessionId) {
        const { session: _selected, ...query } = route.query
        await router.replace({ path: route.path, query })
      }
    }
  }, { immediate: true })
}
