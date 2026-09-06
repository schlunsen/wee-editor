/**
 * Slash command definitions and registry
 */

export interface SlashCommand {
  name: string
  description: string
  category: 'general' | 'session' | 'tools' | 'help' | 'skill'
  icon?: string
  params?: SlashCommandParam[]
  handler?: (params: string[], fullCommand: string) => void | Promise<void>
  /** For skill-based commands: the scope (personal, project, plugin) */
  skillScope?: string
  /** For skill-based commands: argument hint (e.g. "[pr-number]") */
  argumentHint?: string
}

export interface SlashCommandParam {
  name: string
  description: string
  required?: boolean
  type: 'string' | 'number' | 'boolean'
}

/** Built-in slash commands */
export const BUILTIN_COMMANDS: SlashCommand[] = [
  {
    name: 'handoff',
    description: 'Create a session summary and start a new session',
    category: 'session',
    icon: '🔄'
  },
  {
    name: 'clear',
    description: 'Clear messages in the current conversation',
    category: 'session',
    icon: '🗑️'
  },
  {
    name: 'help',
    description: 'Show all available slash commands and their usage',
    category: 'help',
    icon: '❓'
  },
  {
    name: 'settings',
    description: 'Open application settings and preferences',
    category: 'general',
    icon: '⚙️'
  },
  {
    name: 'new',
    description: 'Start a fresh conversation session',
    category: 'session',
    icon: '✨'
  },
  {
    name: 'theme',
    description: 'Switch between light, dark, or auto theme',
    category: 'general',
    icon: '🎨',
    params: [
      {
        name: 'theme_name',
        description: 'Choose: light, dark, or auto',
        required: false,
        type: 'string'
      }
    ]
  },
  {
    name: 'feature',
    description: 'Load a feature context into this session',
    category: 'session',
    icon: '🎯',
    params: [
      {
        name: 'feature_name',
        description: 'Name of the feature to load (e.g. sharepad)',
        required: true,
        type: 'string'
      }
    ]
  }
]

/**
 * Dynamic skill commands, keyed by a caller-provided scope (e.g. projectId)
 * to avoid cross-component clobbering when multiple ChatAreas exist.
 */
const _skillCommandsByScope = new Map<string, SlashCommand[]>()

export function getSlashCommands(): SlashCommand[] {
  const skills = Array.from(_skillCommandsByScope.values()).flat()
  return [...BUILTIN_COMMANDS, ...skills]
}

/** Register skill-based slash commands for a given scope (e.g. projectId) */
export function registerSkillCommands(scope: string, skills: SlashCommand[]) {
  _skillCommandsByScope.set(scope, skills)
}

/** Clear skill commands for a given scope */
export function clearSkillCommands(scope: string) {
  _skillCommandsByScope.delete(scope)
}

export function findSlashCommand(query: string): SlashCommand | null {
  const normalizedQuery = query.toLowerCase().trim()
  const all = getSlashCommands()
  return all.find(cmd => cmd.name === normalizedQuery) || null
}

export function filterSlashCommands(query: string): SlashCommand[] {
  const normalizedQuery = query.toLowerCase().trim()
  const all = getSlashCommands()
  if (!normalizedQuery) return all

  return all.filter(cmd =>
    cmd.name.toLowerCase().includes(normalizedQuery) ||
    cmd.description.toLowerCase().includes(normalizedQuery)
  )
}

export function parseSlashCommand(input: string): { command: string; params: string[] } | null {
  const trimmed = input.trim()
  if (!trimmed.startsWith('/')) return null

  const parts = trimmed.slice(1).split(/\s+/)
  const command = parts[0]
  const params = parts.slice(1)

  return { command, params }
}