/**
 * Composable for Tauri desktop global hotkey integration.
 * Manages keyboard shortcut bindings for skill invocation
 * when running inside the Tauri desktop app.
 */

interface HotkeyBinding {
  shortcut: string
  skill_name: string
  description?: string
}

interface TauriInternals {
  invoke: (cmd: string, args?: Record<string, unknown>) => Promise<unknown>
}

function getTauriInternals(): TauriInternals | null {
  if (typeof window !== 'undefined' && (window as any).__TAURI_INTERNALS__) {
    return (window as any).__TAURI_INTERNALS__ as TauriInternals
  }
  return null
}

export function useTauriHotkeys() {
  const isTauri = computed(() => getTauriInternals() !== null)
  const bindings = ref<HotkeyBinding[]>([])

  /**
   * Load hotkey bindings from the Tauri settings
   */
  async function loadBindings(): Promise<HotkeyBinding[]> {
    const tauri = getTauriInternals()
    if (!tauri) return []

    try {
      const result = (await tauri.invoke('get_hotkey_bindings')) as HotkeyBinding[]
      bindings.value = result
      return result
    } catch {
      return []
    }
  }

  /**
   * Save hotkey bindings to the Tauri settings
   */
  async function saveBindings(newBindings: HotkeyBinding[]): Promise<void> {
    const tauri = getTauriInternals()
    if (!tauri) return

    await tauri.invoke('save_hotkey_bindings', { bindings: newBindings })
    bindings.value = newBindings
  }

  /**
   * Add a new hotkey binding
   */
  async function addBinding(shortcut: string, skillName: string, description?: string): Promise<void> {
    const newBinding: HotkeyBinding = {
      shortcut,
      skill_name: skillName,
      description,
    }
    const updated = [...bindings.value, newBinding]
    await saveBindings(updated)
  }

  /**
   * Remove a hotkey binding by shortcut
   */
  async function removeBinding(shortcut: string): Promise<void> {
    const updated = bindings.value.filter((b) => b.shortcut !== shortcut)
    await saveBindings(updated)
  }

  /**
   * Invoke a skill by name via the Tauri backend
   */
  async function invokeSkill(skillName: string, sessionId?: string): Promise<string | null> {
    const tauri = getTauriInternals()
    if (!tauri) return null

    try {
      return (await tauri.invoke('invoke_skill', {
        skillName,
        sessionId: sessionId || null,
      })) as string
    } catch (e) {
      console.error('Failed to invoke skill:', e)
      return null
    }
  }

  return {
    isTauri,
    bindings,
    loadBindings,
    saveBindings,
    addBinding,
    removeBinding,
    invokeSkill,
  }
}
