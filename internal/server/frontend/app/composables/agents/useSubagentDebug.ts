/**
 * useSubagentDebug - Composable for tracking and displaying individual subagent messages.
 *
 * Receives `subagent_message` WebSocket events and stores them per-agent so the
 * frontend can render a live stream of what each subagent is doing. This is the
 * primary debugging tool for understanding why messages might not appear in the
 * main chat view.
 */

export interface SubagentMessage {
  agent_id: string
  agent_type: string
  description: string
  message_type: 'text' | 'tool_use' | 'tool_result' | 'thinking'
  content: string
  tool_name?: string
  tool_input?: Record<string, any>
  is_error: boolean
  sequence_num: number
  nesting_depth: number
  timestamp: string
  session_id: string
}

export interface SubagentInfo {
  agent_id: string
  agent_type: string
  description: string
  session_id: string
  nesting_depth: number
  first_seen: number
  last_seen: number
  message_count: number
}

// Singleton state shared across all components
const subagentMessages = ref<Map<string, SubagentMessage[]>>(new Map())
const subagentInfo = ref<Map<string, SubagentInfo>>(new Map())
const debugEnabled = ref(true)
const maxMessagesPerAgent = ref(500)

export const useSubagentDebug = () => {
  /**
   * Process an incoming subagent_message WebSocket event.
   */
  const processSubagentMessage = (event: any) => {
    if (!debugEnabled.value) return

    const msg: SubagentMessage = {
      agent_id: event.agent_id,
      agent_type: event.agent_type || 'unknown',
      description: event.description || '',
      message_type: event.message_type,
      content: event.content || '',
      tool_name: event.tool_name,
      tool_input: event.tool_input,
      is_error: event.is_error || false,
      sequence_num: event.sequence_num || 0,
      nesting_depth: event.nesting_depth || 0,
      timestamp: event.timestamp || new Date().toISOString(),
      session_id: event.session_id,
    }

    // Update agent info
    const existing = subagentInfo.value.get(msg.agent_id)
    if (existing) {
      existing.last_seen = Date.now()
      existing.message_count++
    } else {
      subagentInfo.value.set(msg.agent_id, {
        agent_id: msg.agent_id,
        agent_type: msg.agent_type,
        description: msg.description,
        session_id: msg.session_id,
        nesting_depth: msg.nesting_depth,
        first_seen: Date.now(),
        last_seen: Date.now(),
        message_count: 1,
      })
    }

    // Append message to agent's history
    const messages = subagentMessages.value.get(msg.agent_id) || []
    messages.push(msg)

    // Cap at max messages to prevent memory bloat
    if (messages.length > maxMessagesPerAgent.value) {
      messages.splice(0, messages.length - maxMessagesPerAgent.value)
    }

    subagentMessages.value.set(msg.agent_id, messages)

    // Console log for immediate debugging (always, regardless of panel state)
    const prefix = '  '.repeat(msg.nesting_depth)
    const typeIcon = {
      text: '\u{1f4ac}',
      tool_use: '\u{1f527}',
      tool_result: '\u{1f4e6}',
      thinking: '\u{1f914}',
    }[msg.message_type] || '\u{2753}'

    const preview = msg.content
      ? msg.content.substring(0, 120).replace(/\n/g, '\u21b5')
      : msg.tool_name || ''

    console.log(
      `%c[Subagent] ${prefix}${typeIcon} ${msg.agent_type}/${msg.description} #${msg.sequence_num}: ${msg.message_type} ${preview}`,
      msg.is_error ? 'color: red' : 'color: #888'
    )
  }

  /**
   * Get all messages for a specific subagent.
   */
  const getAgentMessages = (agentId: string): SubagentMessage[] => {
    return subagentMessages.value.get(agentId) || []
  }

  /**
   * Get all messages for a session (across all subagents).
   */
  const getSessionMessages = (sessionId: string): SubagentMessage[] => {
    const all: SubagentMessage[] = []
    for (const [agentId, info] of subagentInfo.value) {
      if (info.session_id === sessionId) {
        const msgs = subagentMessages.value.get(agentId) || []
        all.push(...msgs)
      }
    }
    // Sort by timestamp
    all.sort((a, b) => a.timestamp.localeCompare(b.timestamp))
    return all
  }

  /**
   * Get all tracked subagents for a session.
   */
  const getSessionAgents = (sessionId: string): SubagentInfo[] => {
    const agents: SubagentInfo[] = []
    for (const info of subagentInfo.value.values()) {
      if (info.session_id === sessionId) {
        agents.push(info)
      }
    }
    // Sort by first_seen
    agents.sort((a, b) => a.first_seen - b.first_seen)
    return agents
  }

  /**
   * Clear all tracked subagent data.
   */
  const clearAll = () => {
    subagentMessages.value.clear()
    subagentInfo.value.clear()
  }

  /**
   * Clear data for a specific session.
   */
  const clearSession = (sessionId: string) => {
    for (const [agentId, info] of subagentInfo.value) {
      if (info.session_id === sessionId) {
        subagentMessages.value.delete(agentId)
        subagentInfo.value.delete(agentId)
      }
    }
  }

  return {
    // State
    subagentMessages: readonly(subagentMessages),
    subagentInfo: readonly(subagentInfo),
    debugEnabled,

    // Methods
    processSubagentMessage,
    getAgentMessages,
    getSessionMessages,
    getSessionAgents,
    clearAll,
    clearSession,
  }
}
