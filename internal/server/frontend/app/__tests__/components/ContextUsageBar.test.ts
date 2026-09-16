import { mount } from '@vue/test-utils'
import ContextUsageBar from '../../components/agents/ContextUsageBar.vue'
import { hasMeasuredContext } from '../../utils/contextUsage'

const usage = { total_tokens: 0, context_window: 200000, percentage: 0, model: 'test', categories: [], lastUpdateTime: 1, messageCountAtUpdate: 51 }
const mountBar = (props = {}) => mount(ContextUsageBar, { props, global: { stubs: { Icon: true } } })

describe('context reporting', () => {
  it('does not present placeholder zeros as free context for an existing conversation', () => {
    const wrapper = mountBar({ usage, messageCount: 51 })
    expect(wrapper.text()).toContain('Usage unavailable')
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(false)
  })
  it('shows a measured reading and supports refresh', async () => {
    const wrapper = mountBar({ usage: { ...usage, total_tokens: 50000, percentage: 25 }, messageCount: 51 })
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('25')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
  it('keeps the refresh action available without data', () => {
    expect(mountBar().text()).toContain('Usage unavailable')
  })
  it('rejects incomplete, invalid, and zero-window readings while allowing an empty new session', () => {
    expect(hasMeasuredContext(usage, 0)).toBe(true)
    expect(hasMeasuredContext({ ...usage, context_window: 0 })).toBe(false)
    expect(hasMeasuredContext({ ...usage, total_tokens: NaN })).toBe(false)
    expect(hasMeasuredContext({ context_window: 200000 })).toBe(false)
  })
})
