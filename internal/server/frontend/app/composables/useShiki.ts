import { ref, type Ref } from 'vue'
import { createHighlighter, type Highlighter } from 'shiki'

// Singleton highlighter instance
let highlighterPromise: Promise<Highlighter> | null = null
const highlighterReady = ref(false)

// Map from our language names to shiki language IDs
const langAliases: Record<string, string> = {
  typescript: 'typescript',
  javascript: 'javascript',
  python: 'python',
  go: 'go',
  rust: 'rust',
  ruby: 'ruby',
  vue: 'vue',
  html: 'html',
  css: 'css',
  scss: 'scss',
  json: 'json',
  yaml: 'yaml',
  toml: 'toml',
  markdown: 'markdown',
  bash: 'bash',
  sql: 'sql',
  graphql: 'graphql',
  dockerfile: 'dockerfile',
  makefile: 'makefile',
  protobuf: 'proto',
  text: 'text',
  svg: 'html',
}

// Core languages to pre-load (the rest load on demand)
const preloadLangs = [
  'typescript', 'javascript', 'python', 'go', 'json', 'html', 'css',
  'bash', 'yaml', 'markdown', 'vue', 'rust', 'sql', 'toml',
] as const

function getHighlighter(): Promise<Highlighter> {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: ['github-dark-default'],
      langs: [...preloadLangs],
    }).then((h) => {
      highlighterReady.value = true
      return h
    })
  }
  return highlighterPromise
}

export function useShiki() {
  const isReady: Ref<boolean> = highlighterReady

  // Initialize eagerly
  getHighlighter()

  async function highlight(code: string, language: string): Promise<string> {
    const highlighter = await getHighlighter()
    const lang = langAliases[language] || 'text'

    // Load language on demand if not already loaded
    const loadedLangs = highlighter.getLoadedLanguages()
    if (lang !== 'text' && !loadedLangs.includes(lang as any)) {
      try {
        await highlighter.loadLanguage(lang as any)
      } catch {
        // Fall back to text if language not supported
        return highlighter.codeToHtml(code, {
          lang: 'text',
          theme: 'github-dark-default',
        })
      }
    }

    return highlighter.codeToHtml(code, {
      lang: lang === 'text' ? 'text' : lang,
      theme: 'github-dark-default',
    })
  }

  return {
    isReady,
    highlight,
  }
}
