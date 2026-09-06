<template>
  <div>
    <div
      class="tree-node"
      :class="{ 'is-selected': isSelected, 'is-dir': entry.type === 'dir' }"
      :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
      @click="handleClick"
    >
      <!-- Expand Arrow for Directories -->
      <svg
        v-if="entry.type === 'dir'"
        class="expand-arrow"
        :class="{ expanded: isExpanded }"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <polyline points="9 18 15 12 9 6"></polyline>
      </svg>
      <span v-else class="file-spacer"></span>

      <!-- File/Folder Icon -->
      <span class="node-icon">{{ nodeIcon }}</span>

      <!-- Name -->
      <span class="node-name">{{ entry.name }}</span>
    </div>

    <!-- Recursive Children (if expanded directory) -->
    <template v-if="entry.type === 'dir' && isExpanded">
      <FileTreeNode
        v-for="child in children"
        :key="child.name"
        :entry="child"
        :path="path + '/' + child.name"
        :depth="depth + 1"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, type Ref } from 'vue'

interface FileEntry {
  name: string
  type: 'dir' | 'file'
  size?: number
}

interface Props {
  entry: FileEntry
  path: string
  depth: number
}

const props = defineProps<Props>()

// Inject shared state from FileExplorer parent
const expandedDirs = inject<Ref<Set<string>>>('expandedDirs')!
const selectedFile = inject<Ref<string | null>>('selectedFile')!
const toggleDir = inject<(path: string) => void>('toggleDir')!
const selectFile = inject<(path: string) => void>('selectFile')!
const getChildren = inject<(dirPath: string) => FileEntry[]>('getChildren')!

const isExpanded = computed(() => expandedDirs.value.has(props.path))
const isSelected = computed(() => selectedFile.value === props.path)

const children = computed(() => {
  if (props.entry.type !== 'dir' || !isExpanded.value) return []
  return getChildren(props.path)
})

// Icon based on entry type and file extension
const nodeIcon = computed(() => {
  if (props.entry.type === 'dir') {
    return isExpanded.value ? '\uD83D\uDCC2' : '\uD83D\uDCC1'
  }
  const ext = props.entry.name.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, string> = {
    vue: '\uD83D\uDC9A',
    ts: '\uD83D\uDCDC', tsx: '\uD83D\uDCDC', js: '\uD83D\uDCDC', jsx: '\uD83D\uDCDC',
    go: '\uD83D\uDD35',
    json: '\uD83D\uDCCB',
    md: '\uD83D\uDCDD',
    css: '\uD83C\uDFA8', scss: '\uD83C\uDFA8',
    png: '\uD83D\uDDBC\uFE0F', jpg: '\uD83D\uDDBC\uFE0F', jpeg: '\uD83D\uDDBC\uFE0F', gif: '\uD83D\uDDBC\uFE0F', svg: '\uD83D\uDDBC\uFE0F',
  }
  return iconMap[ext] || '\uD83D\uDCC4'
})

function handleClick() {
  if (props.entry.type === 'dir') {
    toggleDir(props.path)
  } else {
    selectFile(props.path)
  }
}
</script>

<style scoped>
.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary, #d4d4d4);
  transition: background 0.15s;
  user-select: none;
  white-space: nowrap;
}

.tree-node:hover {
  background: var(--overlay-bg);
}

.tree-node.is-selected {
  background: var(--overlay-bg-hover);
  color: var(--overlay-text-active);
}

.tree-node.is-dir {
  color: var(--overlay-text-hover);
}

.expand-arrow {
  flex-shrink: 0;
  transition: transform 0.15s ease;
  color: var(--overlay-text);
}

.expand-arrow.expanded {
  transform: rotate(90deg);
}

.file-spacer {
  display: inline-block;
  width: 14px;
  flex-shrink: 0;
}

.node-icon {
  flex-shrink: 0;
  font-size: 14px;
  line-height: 1;
}

.node-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
