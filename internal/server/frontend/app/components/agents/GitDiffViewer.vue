<template>
  <div class="git-diff-viewer">
    <div v-if="parsedDiff.length === 0" class="no-diff">
      <p>No diff content available</p>
    </div>
    <div v-else class="diff-content">
      <div
        v-for="(chunk, index) in parsedDiff"
        :key="index"
        class="diff-chunk"
      >
        <!-- Chunk Header -->
        <div class="chunk-header">
          <span class="chunk-range">{{ chunk.header }}</span>
        </div>

        <!-- Diff Lines -->
        <div class="diff-lines">
          <div
            v-for="(line, lineIndex) in chunk.lines"
            :key="lineIndex"
            :class="['diff-line', line.type]"
          >
            <span class="line-number old-number">{{ line.oldLineNumber || '' }}</span>
            <span class="line-number new-number">{{ line.newLineNumber || '' }}</span>
            <span class="line-prefix">{{ line.prefix }}</span>
            <span class="line-content">{{ line.content }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface DiffLine {
  type: 'added' | 'removed' | 'context' | 'chunk-header'
  oldLineNumber: number | null
  newLineNumber: number | null
  prefix: string
  content: string
}

interface DiffChunk {
  header: string
  lines: DiffLine[]
}

interface Props {
  file: {
    path: string
    status: string
    additions: number
    deletions: number
    diff: string
  }
}

const props = defineProps<Props>()

// Parse unified diff format
const parsedDiff = computed((): DiffChunk[] => {
  if (!props.file.diff) return []

  const lines = props.file.diff.split('\n')
  const chunks: DiffChunk[] = []
  let currentChunk: DiffChunk | null = null
  let oldLineNum = 0
  let newLineNum = 0

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]

    // Chunk header (e.g., @@ -1,5 +1,7 @@)
    if (line.startsWith('@@')) {
      // Save previous chunk if exists
      if (currentChunk) {
        chunks.push(currentChunk)
      }

      // Parse line numbers from chunk header
      const match = line.match(/@@ -(\d+),?\d* \+(\d+),?\d* @@/)
      if (match) {
        oldLineNum = parseInt(match[1])
        newLineNum = parseInt(match[2])
      }

      // Start new chunk
      currentChunk = {
        header: line,
        lines: []
      }
      continue
    }

    // Skip diff metadata lines (---,  +++, etc.)
    if (line.startsWith('---') || line.startsWith('+++') || line.startsWith('diff --git')) {
      continue
    }

    if (!currentChunk) continue

    // Parse diff line
    if (line.startsWith('+')) {
      // Added line
      currentChunk.lines.push({
        type: 'added',
        oldLineNumber: null,
        newLineNumber: newLineNum++,
        prefix: '+',
        content: line.substring(1)
      })
    } else if (line.startsWith('-')) {
      // Removed line
      currentChunk.lines.push({
        type: 'removed',
        oldLineNumber: oldLineNum++,
        newLineNumber: null,
        prefix: '-',
        content: line.substring(1)
      })
    } else {
      // Context line (unchanged)
      currentChunk.lines.push({
        type: 'context',
        oldLineNumber: oldLineNum++,
        newLineNumber: newLineNum++,
        prefix: ' ',
        content: line.substring(1) || line
      })
    }
  }

  // Add last chunk
  if (currentChunk) {
    chunks.push(currentChunk)
  }

  return chunks
})
</script>

<style scoped>
.git-diff-viewer {
  font-family: 'Monaco', 'Menlo', 'Cascadia Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
}

.no-diff {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.diff-content {
  background: var(--bg-primary);
}

.diff-chunk {
  margin-bottom: 16px;
}

.chunk-header {
  padding: 8px 16px;
  background: rgba(100, 116, 139, 0.15);
  border-left: 3px solid var(--accent-cyan);
  color: var(--accent-cyan);
  font-weight: 600;
}

.chunk-range {
  user-select: none;
}

.diff-lines {
  display: flex;
  flex-direction: column;
}

.diff-line {
  display: flex;
  align-items: stretch;
  min-height: 20px;
  transition: background 0.1s;
}

.diff-line:hover {
  background: rgba(100, 116, 139, 0.1);
}

.diff-line.added {
  background: rgba(34, 197, 94, 0.1);
}

.diff-line.added:hover {
  background: rgba(34, 197, 94, 0.15);
}

.diff-line.removed {
  background: rgba(239, 68, 68, 0.1);
}

.diff-line.removed:hover {
  background: rgba(239, 68, 68, 0.15);
}

.line-number {
  display: inline-block;
  width: 50px;
  padding: 0 8px;
  text-align: right;
  color: var(--text-muted);
  background: rgba(100, 116, 139, 0.05);
  user-select: none;
  flex-shrink: 0;
  border-right: 1px solid var(--border-color);
}

.old-number {
  border-right: none;
}

.diff-line.added .line-number {
  background: rgba(34, 197, 94, 0.05);
}

.diff-line.removed .line-number {
  background: rgba(239, 68, 68, 0.05);
}

.line-prefix {
  display: inline-block;
  width: 20px;
  text-align: center;
  user-select: none;
  flex-shrink: 0;
  font-weight: 700;
}

.diff-line.added .line-prefix {
  color: var(--status-success);
}

.diff-line.removed .line-prefix {
  color: var(--status-error);
}

.diff-line.context .line-prefix {
  color: var(--text-muted);
}

.line-content {
  flex: 1;
  padding: 0 12px;
  white-space: pre;
  overflow-x: auto;
  color: var(--text-primary);
}

.diff-line.added .line-content {
  background: rgba(34, 197, 94, 0.05);
}

.diff-line.removed .line-content {
  background: rgba(239, 68, 68, 0.05);
}

/* Scrollbar styling */
.git-diff-viewer::-webkit-scrollbar {
  height: 8px;
}

.git-diff-viewer::-webkit-scrollbar-track {
  background: var(--bg-secondary);
}

.git-diff-viewer::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.git-diff-viewer::-webkit-scrollbar-thumb:hover {
  background: var(--accent-purple);
}
</style>
