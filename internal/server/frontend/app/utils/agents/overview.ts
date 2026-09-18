export interface OverviewLogEntry {
  id: string
  time: string
  kind: string
  text: string
}

function textContent(value: unknown): string {
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (Array.isArray(parsed)) return textContent(parsed)
    } catch { /* Plain message text. */ }
    return value
  }
  if (Array.isArray(value)) return value.map(block => {
    if (block?.type === 'text') return block.text || ''
    if (block?.type === 'tool_result') return textContent(block.content)
    return ''
  }).filter(Boolean).join('\n')
  return ''
}

export function overviewLog(messages: any[]): OverviewLogEntry[] {
  const entries: OverviewLogEntry[] = []
  for (const message of messages) {
    const time = message.created_at || message.timestamp || ''
    const id = message.id || `${time}-${entries.length}`
    const text = textContent(message.content)
    if (text.trim()) entries.push({ id, time, kind: message.role === 'user' ? 'User' : message.role === 'system' ? 'System' : 'Agent', text: text.slice(0, 3000) })
    for (const tool of message.tool_uses || message.toolUses || []) {
      const input = tool.input || {}
      const detail = input.command || input.cmd || input.file_path || input.path || input.query || input.description || ''
      entries.push({ id: `${id}-${tool.id || entries.length}`, time, kind: tool.name || 'Tool', text: String(detail).slice(0, 1500) || 'Running tool' })
    }
  }
  return entries.slice(-60)
}

export function changedFileCount(status: any): number {
  if (!status) return 0
  return new Set([...(status.staged || []), ...(status.modified || []), ...(status.untracked || []), ...(status.deleted || [])]).size
}
