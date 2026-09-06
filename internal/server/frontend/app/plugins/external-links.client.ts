/**
 * Nuxt client plugin: External Link Handler for Tauri
 *
 * In the Tauri desktop app, `target="_blank"` links don't automatically open
 * in the user's default browser — the webview blocks or ignores them.
 *
 * This plugin adds a global click interceptor that:
 * - Detects if we're running inside Tauri (via __TAURI_INTERNALS__)
 * - Intercepts clicks on links with `target="_blank"` or external hrefs
 * - Opens them in the OS default browser via Tauri's shell plugin
 * - Does nothing in a regular browser — links work normally
 */
export default defineNuxtPlugin(() => {
  // Only activate inside Tauri
  const tauriInternals = (window as any).__TAURI_INTERNALS__
  if (!tauriInternals) return

  /**
   * Open a URL in the default browser via Tauri's shell:open IPC command.
   * This requires `shell:allow-open` in src-tauri/capabilities/default.json.
   */
  async function openInBrowser(url: string) {
    try {
      await tauriInternals.invoke('plugin:shell|open', {
        path: url,
        with: null,
      })
    }
    catch (err) {
      console.error('[external-links] Failed to open URL in browser:', url, err)
      // Fallback: try window.open (may or may not work in webview)
      window.open(url, '_blank')
    }
  }

  /**
   * Determine if a URL is "external" — i.e. should open in the browser,
   * not navigate the webview.
   */
  function isExternalUrl(href: string): boolean {
    if (!href) return false
    // Explicit external protocols
    if (href.startsWith('http://') || href.startsWith('https://')) {
      // Internal app URLs (localhost on the app port) stay in the webview
      try {
        const url = new URL(href)
        if (url.hostname === 'localhost' || url.hostname === '127.0.0.1') {
          return false
        }
      }
      catch {
        // If URL parsing fails, treat as external to be safe
      }
      return true
    }
    // mailto:, tel:, etc.
    if (href.startsWith('mailto:') || href.startsWith('tel:')) {
      return true
    }
    return false
  }

  // Global click interceptor on the document
  document.addEventListener('click', (event) => {
    // Walk up the DOM to find the nearest <a> element
    const target = (event.target as HTMLElement)?.closest?.('a')
    if (!target) return

    const href = target.getAttribute('href')
    if (!href) return

    // Only open links that point to actual external URLs
    // (ignore relative paths like "/some/page" even if they have target="_blank")
    if (isExternalUrl(href)) {
      event.preventDefault()
      openInBrowser(href)
    }
  }, true) // Use capture phase to intercept before Vue handlers
})
