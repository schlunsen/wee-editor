import { flushPromises, mount } from '@vue/test-utils'
import GitStatus from '../../components/agents/GitStatus.vue'

const baseStatus = {
  branch: 'authz/4-typed-api-errors',
  ahead: 0,
  behind: 0,
  staged: [],
  modified: ['desktop/frontend/src/api/http-client.ts'],
  untracked: [],
  deleted: [],
  clean: false
}

const mountGit = (props: Record<string, unknown>) =>
  mount(GitStatus, {
    props: { status: baseStatus, sessionId: 'session-1', ...props },
    global: { stubs: { NuxtLink: { template: '<a><slot /></a>' } } }
  })

describe('worktree indicator', () => {
  const worktreePath = '/Users/me/.claude/worktrees/authz+4-typed-api-errors'

  it('shows the worktree name prominently and the full path unabbreviated', () => {
    const wrapper = mountGit({ worktreePath })
    const badge = wrapper.get('[data-testid="worktree-indicator"]')
    expect(badge.get('.worktree-name').text()).toBe('authz+4-typed-api-errors')
    expect(badge.get('.worktree-path').text()).toBe(worktreePath)
    expect(badge.text()).not.toContain('...')
  })

  it('copies the full path to the clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.spyOn(navigator, 'clipboard', 'get').mockReturnValue({ writeText } as unknown as Clipboard)
    const wrapper = mountGit({ worktreePath })
    await wrapper.get('.worktree-copy').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith(worktreePath)
    expect(wrapper.get('.worktree-copy').attributes('aria-label')).toBe('Worktree path copied')
    vi.restoreAllMocks()
  })

  it('is hidden when the session is not in a worktree', () => {
    expect(mountGit({}).find('[data-testid="worktree-indicator"]').exists()).toBe(false)
  })
})

describe('GitHub branch links', () => {
  it('links to the branch and a compare view before any PR exists', () => {
    const wrapper = mountGit({ githubUrl: 'https://github.com/acme/wee/', status: { ...baseStatus, source_branch: 'origin/main' } })
    const links = wrapper.get('[data-testid="github-branch-links"]')
    expect(links.get('.github-branch-link').attributes('href')).toBe('https://github.com/acme/wee/tree/authz/4-typed-api-errors')
    expect(links.get('.github-compare-link').attributes('href')).toBe('https://github.com/acme/wee/compare/main...authz/4-typed-api-errors?expand=1')
    // The gh CLI hint is redundant once a GitHub link is available
    expect(wrapper.find('.gh-info').exists()).toBe(false)
  })

  it('drops the compare link once a PR is open, keeping the branch link', () => {
    const pr = { number: 7, title: 'Typed errors', url: 'https://github.com/acme/wee/pull/7', state: 'open' }
    const wrapper = mountGit({ githubUrl: 'https://github.com/acme/wee', status: { ...baseStatus, pr } })
    expect(wrapper.find('.github-branch-link').exists()).toBe(true)
    expect(wrapper.find('.github-compare-link').exists()).toBe(false)
    expect(wrapper.get('.pr-link').attributes('href')).toBe(pr.url)
  })

  it('offers no compare link on the default branch and no links without a GitHub remote', () => {
    const onMain = mountGit({ githubUrl: 'https://github.com/acme/wee', status: { ...baseStatus, branch: 'main' } })
    expect(onMain.get('.github-branch-link').attributes('href')).toBe('https://github.com/acme/wee/tree/main')
    expect(onMain.find('.github-compare-link').exists()).toBe(false)

    const noRemote = mountGit({})
    expect(noRemote.find('[data-testid="github-branch-links"]').exists()).toBe(false)
    expect(noRemote.find('.gh-info').exists()).toBe(true)
  })

  it('encodes branch segments without escaping the slashes GitHub expects', () => {
    const wrapper = mountGit({ githubUrl: 'https://github.com/acme/wee', status: { ...baseStatus, branch: 'feat/#12 space' } })
    expect(wrapper.get('.github-branch-link').attributes('href')).toBe('https://github.com/acme/wee/tree/feat/%2312%20space')
  })
})
