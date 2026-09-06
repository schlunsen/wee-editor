/**
 * Metrics Store
 *
 * Tracks tool usage, permission approvals, and context usage metrics
 * across sessions with comprehensive analytics.
 */

import { defineStore } from 'pinia'
import type { ToolStats, PermissionStats, ContextUsage, MetricsState } from './types'

export const useMetricsStore = defineStore('metrics', {
  state: (): MetricsState => ({
    sessionToolStats: new Map(),
    sessionPermissionStats: new Map(),
    sessionContextUsage: new Map(),
    globalToolStats: {},
    totalPermissions: {
      approved: 0,
      denied: 0,
      total: 0
    }
  }),

  getters: {
    /**
     * Get top tools by usage count
     */
    topTools: (state) => {
      return (limit = 10) => {
        const sorted = Object.entries(state.globalToolStats)
          .sort(([, a], [, b]) => b - a)
          .slice(0, limit)
        return Object.fromEntries(sorted)
      }
    },

    /**
     * Get permission approval rate
     */
    permissionApprovalRate: (state) => {
      if (state.totalPermissions.total === 0) return 0
      return (state.totalPermissions.approved / state.totalPermissions.total) * 100
    },

    /**
     * Get total context usage across all sessions
     */
    totalContextUsage: (state) => {
      let totalTokens = 0
      let totalCost = 0

      state.sessionContextUsage.forEach((usage) => {
        totalTokens += usage.totalTokens
        totalCost += usage.cost
      })

      return {
        totalTokens,
        totalCost,
        sessionCount: state.sessionContextUsage.size
      }
    },

    /**
     * Get average tokens per session
     */
    averageTokensPerSession: (state) => {
      if (state.sessionContextUsage.size === 0) return 0
      const total = Array.from(state.sessionContextUsage.values()).reduce(
        (sum, usage) => sum + usage.totalTokens,
        0
      )
      return Math.round(total / state.sessionContextUsage.size)
    },

    /**
     * Get tool stats for a specific session
     */
    getSessionToolStats: (state) => {
      return (sessionId: string) => {
        return state.sessionToolStats.get(sessionId) || {}
      }
    },

    /**
     * Get permission stats for a specific session
     */
    getSessionPermissionStats: (state) => {
      return (sessionId: string) => {
        return (
          state.sessionPermissionStats.get(sessionId) || {
            approved: 0,
            denied: 0,
            total: 0
          }
        )
      }
    },

    /**
     * Get context usage for a specific session
     */
    getSessionContextUsage: (state) => {
      return (sessionId: string) => {
        return state.sessionContextUsage.get(sessionId) || null
      }
    },

  },

  actions: {
    /**
     * Record a tool usage
     */
    recordToolUse(sessionId: string, toolName: string) {
      // Update session tool stats
      if (!this.sessionToolStats.has(sessionId)) {
        this.sessionToolStats.set(sessionId, {})
      }
      const sessionStats = this.sessionToolStats.get(sessionId)!
      sessionStats[toolName] = (sessionStats[toolName] || 0) + 1

      // Update global tool stats
      this.globalToolStats[toolName] = (this.globalToolStats[toolName] || 0) + 1
    },

    /**
     * Record a permission decision
     */
    recordPermission(sessionId: string, approved: boolean) {
      // Update session permission stats
      if (!this.sessionPermissionStats.has(sessionId)) {
        this.sessionPermissionStats.set(sessionId, {
          approved: 0,
          denied: 0,
          total: 0
        })
      }
      const sessionStats = this.sessionPermissionStats.get(sessionId)!
      if (approved) {
        sessionStats.approved++
      } else {
        sessionStats.denied++
      }
      sessionStats.total++

      // Update total permissions
      if (approved) {
        this.totalPermissions.approved++
      } else {
        this.totalPermissions.denied++
      }
      this.totalPermissions.total++
    },

    /**
     * Update context usage for a session with actual SDK data
     */
    updateContextUsage(sessionId: string, usage: ContextUsage, messageCount: number = 0) {
      this.sessionContextUsage.set(sessionId, {
        ...usage,
        lastUpdateTime: Date.now(),
        messageCountAtUpdate: messageCount,
      })
    },

    /**
     * Restore context usage from cache (e.g., localStorage on page reload)
     */
    restoreContextUsage(sessionId: string, usage: ContextUsage) {
      this.sessionContextUsage.set(sessionId, {
        ...usage,
      })
    },


    /**
     * Clear metrics for a specific session
     */
    clearSessionMetrics(sessionId: string) {
      this.sessionToolStats.delete(sessionId)
      this.sessionPermissionStats.delete(sessionId)
      this.sessionContextUsage.delete(sessionId)
    },

    /**
     * Clear all metrics
     */
    clearAll() {
      this.sessionToolStats.clear()
      this.sessionPermissionStats.clear()
      this.sessionContextUsage.clear()
      this.globalToolStats = {}
      this.totalPermissions = {
        approved: 0,
        denied: 0,
        total: 0
      }
    }
  }
})
