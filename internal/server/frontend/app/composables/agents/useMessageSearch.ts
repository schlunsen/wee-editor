import { ref, computed } from 'vue'
import type { Message } from '~/types/agents'

export interface SearchResult {
  message: Message
  matchPositions: Array<{
    start: number
    end: number
  }>
  context: string
}

export function useMessageSearch() {
  const searchQuery = ref('')
  const isCaseSensitive = ref(false)

  /**
   * Extract text content from message
   */
  const extractMessageText = (message: Message): string => {
    if (typeof message.content === 'string') {
      return message.content
    }

    if (Array.isArray(message.content)) {
      return message.content
        .filter(block => typeof block === 'object' && block.type === 'text')
        .map(block => (block as any).text || '')
        .join(' ')
    }

    return ''
  }

  /**
   * Find all positions of search query in text
   */
  const findMatchPositions = (text: string, query: string): Array<{ start: number; end: number }> => {
    const positions: Array<{ start: number; end: number }> = []
    const searchText = isCaseSensitive.value ? text : text.toLowerCase()
    const searchQuery = isCaseSensitive.value ? query : query.toLowerCase()

    if (!searchQuery) return positions

    let startIndex = 0
    while (true) {
      const index = searchText.indexOf(searchQuery, startIndex)
      if (index === -1) break

      positions.push({
        start: index,
        end: index + searchQuery.length
      })

      startIndex = index + 1
    }

    return positions
  }

  /**
   * Get context around matched text
   */
  const getMatchContext = (text: string, position: { start: number; end: number }, contextLength: number = 50): string => {
    const start = Math.max(0, position.start - contextLength)
    const end = Math.min(text.length, position.end + contextLength)

    let context = text.substring(start, end)

    // Add ellipsis if truncated
    if (start > 0) context = '...' + context
    if (end < text.length) context = context + '...'

    return context
  }

  /**
   * Search through messages
   */
  const searchMessages = (messages: Message[]): SearchResult[] => {
    if (!searchQuery.value.trim()) return []

    const results: SearchResult[] = []

    for (const message of messages) {
      const text = extractMessageText(message)
      const matchPositions = findMatchPositions(text, searchQuery.value)

      if (matchPositions.length > 0) {
        // Use the first match for context
        const firstMatch = matchPositions[0]
        const context = getMatchContext(text, firstMatch)

        results.push({
          message,
          matchPositions,
          context
        })
      }
    }

    return results
  }

  /**
   * Filter messages based on search query
   */
  const filterMessages = (messages: Message[]): Message[] => {
    if (!searchQuery.value.trim()) return messages

    return messages.filter(message => {
      const text = extractMessageText(message)
      const searchText = isCaseSensitive.value ? text : text.toLowerCase()
      const query = isCaseSensitive.value ? searchQuery.value : searchQuery.value.toLowerCase()
      return searchText.includes(query)
    })
  }

  /**
   * Clear search
   */
  const clearSearch = () => {
    searchQuery.value = ''
  }

  /**
   * Highlight search results in text
   */
  const highlightMatches = (text: string): string => {
    if (!searchQuery.value.trim()) return text

    const searchText = isCaseSensitive.value ? text : text.toLowerCase()
    const query = isCaseSensitive.value ? searchQuery.value : searchQuery.value.toLowerCase()

    if (!searchText.includes(query)) return text

    const parts = text.split(new RegExp(`(${escapeRegex(query)})`, isCaseSensitive.value ? 'g' : 'gi'))

    return parts
      .map((part, idx) => {
        const isMatch = isCaseSensitive.value
          ? part === searchQuery.value
          : part.toLowerCase() === searchQuery.value.toLowerCase()

        return isMatch ? `<mark>${escapeHtml(part)}</mark>` : escapeHtml(part)
      })
      .join('')
  }

  /**
   * Escape regex special characters
   */
  const escapeRegex = (str: string): string => {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  }

  /**
   * Escape HTML special characters
   */
  const escapeHtml = (str: string): string => {
    const div = document.createElement('div')
    div.textContent = str
    return div.innerHTML
  }

  // Computed properties
  const hasQuery = computed(() => searchQuery.value.trim().length > 0)

  return {
    searchQuery,
    isCaseSensitive,
    hasQuery,
    searchMessages,
    filterMessages,
    clearSearch,
    highlightMatches,
    extractMessageText,
    getMatchContext
  }
}
