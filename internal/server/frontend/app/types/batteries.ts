/**
 * Battery component types for MCPs, Agents, and Commands
 */

export type ComponentType = 'mcp' | 'agent' | 'command'

export interface BatteryComponent {
  name: string
  path: string
  category: string
  type: ComponentType
  installed: boolean
  description?: string
}

export interface ComponentPreview {
  name: string
  content: string
  path: string
  type: ComponentType
  installed?: boolean
}

export interface ComponentsResponse {
  components: BatteryComponent[]
}

export interface InstallResponse {
  message: string
  name: string
}
