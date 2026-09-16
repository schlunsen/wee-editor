/** A zero placeholder after messages exist is not a measured context reading. */
export function hasMeasuredContext(usage: { total_tokens?: number; context_window?: number; percentage?: number } | null | undefined, messageCount = 0): boolean {
  return !!usage && Number.isFinite(usage.total_tokens) && usage.total_tokens! >= 0
    && Number.isFinite(usage.context_window) && usage.context_window! > 0
    && Number.isFinite(usage.percentage) && usage.percentage! >= 0
    && (usage.total_tokens! > 0 || messageCount === 0)
}
