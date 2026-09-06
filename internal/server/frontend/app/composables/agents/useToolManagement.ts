import { type Ref } from 'vue'
import type { TodoItem } from '~/utils/agents/todoParser'
import type { ActiveTool } from '~/types/agents'

interface ToolExecution {
  toolName: string
  filePath?: string
  command?: string
  pattern?: string
  detail?: string
  timestamp?: Date
}

interface ToolManagementParams {
  sessionTodos: Ref<Map<string, TodoItem[]>>
  sessionToolExecution: Ref<Map<string, ToolExecution | null>>
  activeTools: Ref<Map<string, ActiveTool[]>>
}

export function useToolManagement(params: ToolManagementParams) {
  const { sessionTodos, sessionToolExecution, activeTools } = params

  // Update session data methods
  const updateSessionTodos = (sessionId: string, todos: TodoItem[]) => {
    sessionTodos.value.set(sessionId, todos)
  }

  const updateSessionToolExecution = (sessionId: string, toolExecution: ToolExecution | null) => {
    sessionToolExecution.value.set(sessionId, toolExecution)
  }

  const clearSessionToolExecution = (sessionId: string) => {
    sessionToolExecution.value.delete(sessionId)
  }

  // Tool overlay management
  const addActiveTool = (sessionId: string, toolUse: any, messageId?: string) => {
    const tools = activeTools.value.get(sessionId) || []

    // For Edit tool, check if there's an existing tool for the same file
    // If so, update it instead of creating a new one
    if (toolUse.name === 'Edit' && toolUse.input?.file_path) {
      const existingToolIndex = tools.findIndex(
        t => t.name === 'Edit' &&
             t.input?.file_path === toolUse.input.file_path &&
             (t.status === 'running' || t.status === 'completed')
      )

      if (existingToolIndex !== -1) {
        // Update existing tool with new data
        tools[existingToolIndex] = {
          id: toolUse.id,
          name: toolUse.name,
          input: toolUse.input,
          status: 'running',
          startTime: Date.now(),
          sessionId,
          messageId
        }
        // Create a new Map to trigger Vue reactivity on the ref
        const newMap = new Map(activeTools.value)
        newMap.set(sessionId, [...tools])
        activeTools.value = newMap
        return
      }
    }

    // For TodoWrite tool, check if there's an existing TodoWrite tool
    // If so, update it instead of creating a new one (single persistent overlay)
    if (toolUse.name === 'TodoWrite') {
      const existingToolIndex = tools.findIndex(
        t => t.name === 'TodoWrite' && (t.status === 'running' || t.status === 'completed')
      )

      if (existingToolIndex !== -1) {
        // Update existing TodoWrite tool with new data
        tools[existingToolIndex] = {
          id: toolUse.id,
          name: toolUse.name,
          input: toolUse.input,
          status: 'running',
          startTime: Date.now(),
          sessionId,
          messageId
        }
        // Create a new Map to trigger Vue reactivity on the ref
        const newMap = new Map(activeTools.value)
        newMap.set(sessionId, [...tools])
        activeTools.value = newMap
        return
      }
    }

    // Otherwise, create a new tool
    const activeTool: ActiveTool = {
      id: toolUse.id,
      name: toolUse.name,
      input: toolUse.input,
      status: 'running',
      startTime: Date.now(),
      sessionId,
      messageId
    }
    tools.push(activeTool)
    // Create a new array to trigger Vue reactivity
    const newTools = [...tools]
    // Create a new Map to trigger Vue reactivity on the ref
    const newMap = new Map(activeTools.value)
    newMap.set(sessionId, newTools)
    activeTools.value = newMap
  }

  const completeActiveTool = (sessionId: string, toolUseId: string, isError: boolean = false) => {
    const tools = activeTools.value.get(sessionId) || []
    const tool = tools.find(t => t.id === toolUseId)
    if (tool) {
      tool.status = isError ? 'error' : 'completed'
      tool.endTime = Date.now()
      // Create a new Map to trigger Vue reactivity on the ref
      const newMap = new Map(activeTools.value)
      newMap.set(sessionId, [...tools])
      activeTools.value = newMap
    }
  }

  const removeActiveTool = (sessionId: string, toolId: string) => {
    const tools = activeTools.value.get(sessionId) || []
    const filtered = tools.filter(t => t.id !== toolId)
    // Create a new Map to trigger Vue reactivity on the ref
    const newMap = new Map(activeTools.value)
    newMap.set(sessionId, filtered)
    activeTools.value = newMap
  }

  const clearAllActiveTools = (sessionId: string) => {
    // Clear all tools for a session in a single operation
    const newMap = new Map(activeTools.value)
    newMap.set(sessionId, [])
    activeTools.value = newMap
  }

  return {
    updateSessionTodos,
    updateSessionToolExecution,
    clearSessionToolExecution,
    addActiveTool,
    completeActiveTool,
    removeActiveTool,
    clearAllActiveTools
  }
}
