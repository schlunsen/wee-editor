import { overviewLog, changedFileCount } from '../../utils/agents/overview'

it('renders saved text and tool calls without showing image payloads or raw JSON blocks', () => {
  const entries = overviewLog([
    { id: 'a', role: 'assistant', created_at: '2026-09-16T10:00:00Z', content: 'Checking tests', tool_uses: [{ id: 'tool', name: 'Bash', input: { command: 'go test ./...' } }] },
    { id: 'b', role: 'user', content: JSON.stringify([{ type: 'image', source: { data: 'image-payload' } }, { type: 'text', text: 'Use this design' }]) }
  ])
  expect(entries.map(entry => entry.text)).toEqual(['Checking tests', 'go test ./...', 'Use this design'])
  expect(entries[1].kind).toBe('Bash')
})

it('bounds the live log and counts staged-and-modified files once', () => {
  expect(overviewLog(Array.from({ length: 80 }, (_, index) => ({ id: String(index), content: 'message', role: 'assistant' })))).toHaveLength(60)
  expect(changedFileCount({ staged: ['file.go'], modified: ['file.go', 'other.go'], untracked: ['new.go'] })).toBe(3)
})
