/**
 * Slash command autocomplete management
 * Supports two-level completion: commands → argument values (for commands with required params)
 */

import { ref, watch, nextTick, type Ref } from 'vue'
import { findSlashCommand, type SlashCommand } from '~/composables/useSlashCommands'

export interface AutocompletePosition {
  top: number
  left: number
  width?: number
  bottom?: number
}

export function useSlashCommandAutocomplete(
  textareaRef: Ref<HTMLTextAreaElement | null>,
  inputMessage: Ref<string>,
  onCommandSelect: (command: SlashCommand) => void,
  onFeatureSelect?: (featureName: string) => void
) {
  const isVisible = ref(false)
  const selectedIndex = ref(0)
  const query = ref('')
  const position = ref<AutocompletePosition>({ top: 0, left: 0 })
  const isPasting = ref(false)

  // Sub-completion state (activated for any command whose first param is required)
  const isFeatureMode = ref(false)
  const availableFeatures = ref<string[]>([])

  // Shared textarea line-replace helper — avoids duplication between selectCommand / selectFeature
  const replaceCurrentLine = (textarea: HTMLTextAreaElement, newLine: string) => {
    const lines = textarea.value.split('\n')
    lines[lines.length - 1] = newLine
    const newText = lines.join('\n')
    textarea.value = newText
    textarea.selectionStart = textarea.selectionEnd = newText.length
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
  }

  // Pure function (not a computed) that evaluates the current input and updates
  // mode/query state as a side effect. Called from handleInput() only.
  const checkAutocomplete = (): boolean => {
    if (isPasting.value) return false

    const text = inputMessage.value
    if (!text) return false

    const currentLine = text.split('\n').at(-1) ?? ''

    // Second-level: /<cmd> <partial> — any command whose first param is required
    const subCmdMatch = currentLine.match(/^\s*\/(\w+)\s+(\w*)$/)
    if (subCmdMatch) {
      const cmd = findSlashCommand(subCmdMatch[1]!)
      if (cmd?.params?.[0]?.required) {
        isFeatureMode.value = true
        query.value = subCmdMatch[2]!
        return true
      }
    }

    isFeatureMode.value = false

    // First-level: /<partial>
    const slashMatch = currentLine.match(/^\s*\/(\w*)$/)
    if (slashMatch) {
      query.value = slashMatch[1]!
      return true
    }

    return false
  }

  // Calculate autocomplete position
  const updatePosition = () => {
    if (!textareaRef.value) return

    const textarea = textareaRef.value
    const inputArea = textarea.closest('.input-area')
    const inputAreaRect = inputArea?.getBoundingClientRect()

    if (inputAreaRect) {
      position.value = {
        top: 0,
        left: inputAreaRect.left + 16,
        width: inputAreaRect.width - 32,
        bottom: window.innerHeight - inputAreaRect.top + 8,
      }
    } else {
      const rect = textarea.getBoundingClientRect()
      position.value = { top: 0, left: rect.left, width: rect.width, bottom: window.innerHeight - rect.top + 8 }
    }
  }

  // Handle input changes — the only place checkAutocomplete() is called
  const handleInput = () => {
    if (isPasting.value) return
    isVisible.value = checkAutocomplete()
    if (isVisible.value) {
      nextTick(updatePosition)
    }
  }

  // Handle keydown events
  const handleKeydown = (event: KeyboardEvent) => {
    if (!isVisible.value) return

    if (['ArrowDown', 'ArrowUp', 'Enter', 'Escape'].includes(event.key)) {
      return
    }

    if (event.key === 'Backspace') {
      const currentLine = inputMessage.value.split('\n').at(-1) ?? ''

      // Exit sub-completion mode when user backspaces past the argument space
      if (isFeatureMode.value && /^\s*\/\w+\s*$/.test(currentLine)) {
        isFeatureMode.value = false
      }

      // Hide entirely when user deletes back past the slash
      if (/^\s*\/\s*$/.test(currentLine)) {
        isVisible.value = false
      }
    }
  }

  // Handle paste events to avoid false triggers
  const handlePaste = () => {
    isPasting.value = true
    isVisible.value = false
    setTimeout(() => { isPasting.value = false }, 100)
  }

  // Handle command selection from the first level
  const selectCommand = (command: SlashCommand) => {
    if (!textareaRef.value) return

    const hasRequiredParam = command.params?.[0]?.required === true
    // Commands with a required first param get a trailing space so the user
    // can immediately type the argument value
    const newLine = hasRequiredParam ? `/${command.name} ` : `/${command.name}`
    replaceCurrentLine(textareaRef.value, newLine)

    if (hasRequiredParam) {
      isFeatureMode.value = true
      query.value = ''
      selectedIndex.value = 0
      nextTick(updatePosition)
      return
    }

    isVisible.value = false
    onCommandSelect(command)
  }

  // Handle argument value selection (second-level completion)
  const selectFeature = (featureName: string) => {
    if (!textareaRef.value) return

    const cmd = inputMessage.value.split('\n').at(-1)?.match(/^\s*\/(\w+)\s/)?.[1] ?? 'feature'
    replaceCurrentLine(textareaRef.value, `/${cmd} ${featureName}`)

    isVisible.value = false
    isFeatureMode.value = false
    onFeatureSelect?.(featureName)
  }

  // Dismiss autocomplete
  const dismiss = () => {
    isVisible.value = false
    isFeatureMode.value = false
  }

  // Reset selection index when visibility or mode changes
  watch([isVisible, isFeatureMode], () => {
    selectedIndex.value = 0
  })

  // Auto-dismiss when clicking outside
  const handleClickOutside = (event: MouseEvent) => {
    if (!textareaRef.value?.contains(event.target as Node)) {
      dismiss()
    }
  }

  return {
    isVisible,
    query,
    position,
    selectedIndex,
    isFeatureMode,
    availableFeatures,
    handleInput,
    handleKeydown,
    handlePaste,
    selectCommand,
    selectFeature,
    dismiss,
    handleClickOutside
  }
}
