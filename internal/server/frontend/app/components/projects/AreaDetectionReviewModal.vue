<template>
  <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <div class="header-content">
          <h2>Review Detected Areas</h2>
          <p class="subtitle">
            {{ selectedAreas.length }} of {{ allAreas.length }} areas selected
            <span v-if="allAreas.length > selectedAreas.length" class="confidence-hint">
              (tip: uncheck low-confidence detections)
            </span>
          </p>
        </div>
        <button @click="$emit('close')" class="modal-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <!-- Quick Filters -->
      <div class="filter-bar">
        <label class="checkbox-label">
          <input type="checkbox" v-model="showLowConfidence" />
          Show all (including low confidence)
        </label>
        <div class="spacer"></div>
        <button @click="selectAll" class="btn-link">Select All</button>
        <button @click="deselectAll" class="btn-link">Deselect All</button>
      </div>

      <div class="modal-body">
        <div v-if="visibleAreas.length === 0" class="no-areas">
          <p>No areas to display</p>
        </div>

        <div v-else class="areas-list">
          <div
            v-for="area in visibleAreas"
            :key="area.name"
            class="area-item"
            :class="{ selected: isSelected(area.name) }"
          >
            <div class="area-checkbox">
              <input
                type="checkbox"
                :checked="isSelected(area.name)"
                @change="toggleSelection(area.name)"
              />
            </div>

            <div class="area-content">
              <div class="area-header">
                <div class="area-title">
                  <span class="area-icon" :style="{ backgroundColor: area.color }">
                    {{ area.icon }}
                  </span>
                  <div class="area-name-block">
                    <h3>{{ area.name }}</h3>
                    <p class="area-path">{{ area.relative_path }}</p>
                  </div>
                </div>

                <div class="area-meta">
                  <span class="confidence-badge" :class="getConfidenceClass(area.confidence)">
                    {{ Math.round(area.confidence * 100) }}%
                  </span>
                  <span class="type-badge">{{ area.subdomain_type }}</span>
                </div>
              </div>

              <p v-if="area.description" class="area-description">{{ area.description }}</p>

              <div class="area-footer">
                <span class="detected-type">{{ area.detected_type }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="modal-actions">
        <div class="action-info">
          <p v-if="selectedAreas.length > 0" class="info-text">
            ✅ {{ selectedAreas.length }} area{{ selectedAreas.length !== 1 ? 's' : '' }} will be added to the database
          </p>
          <p v-else class="warning-text">
            ⚠️ No areas selected. Please select areas to add.
          </p>
        </div>
        <div class="spacer"></div>
        <button @click="$emit('close')" class="btn-cancel">Cancel</button>
        <button
          @click="handleSave"
          class="btn-save"
          :disabled="selectedAreas.length === 0 || saving"
        >
          <div v-if="saving" class="btn-spinner"></div>
          <span v-else>✓ Add Selected Areas</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { DetectedArea } from '~/types/projects'

interface Props {
  show: boolean
  areas: DetectedArea[]
  saving?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  saving: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', areas: DetectedArea[]): void
}>()

// State
const selectedAreaNames = ref<Set<string>>(new Set())
const showLowConfidence = ref(false)

// Computed
const allAreas = computed(() => props.areas)

const visibleAreas = computed(() => {
  if (showLowConfidence.value) {
    return allAreas.value
  }
  // Filter out low confidence areas (< 70%)
  return allAreas.value.filter(area => area.confidence >= 0.7)
})

const selectedAreas = computed(() => {
  return allAreas.value.filter(area => selectedAreaNames.value.has(area.name))
})

// Methods
const isSelected = (areaName: string) => {
  return selectedAreaNames.value.has(areaName)
}

const toggleSelection = (areaName: string) => {
  if (selectedAreaNames.value.has(areaName)) {
    selectedAreaNames.value.delete(areaName)
  } else {
    selectedAreaNames.value.add(areaName)
  }
}

const selectAll = () => {
  visibleAreas.value.forEach(area => {
    selectedAreaNames.value.add(area.name)
  })
}

const deselectAll = () => {
  selectedAreaNames.value.clear()
}

const getConfidenceClass = (confidence: number) => {
  if (confidence >= 0.9) return 'high'
  if (confidence >= 0.7) return 'medium'
  return 'low'
}

const handleSave = () => {
  if (selectedAreas.value.length === 0) return
  emit('save', selectedAreas.value)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.header-content {
  flex: 1;
}

.modal-header h2 {
  margin: 0 0 0.25rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.subtitle {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.confidence-hint {
  font-size: 0.85rem;
  font-style: italic;
}

.modal-close {
  padding: 0.5rem;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
  flex-shrink: 0;
}

.modal-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

/* Filter Bar */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 0.9rem;
  user-select: none;
}

.checkbox-label input[type="checkbox"] {
  cursor: pointer;
  width: 16px;
  height: 16px;
}

.spacer {
  flex: 1;
}

.btn-link {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  transition: all 0.2s;
}

.btn-link:hover {
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

/* Modal Body */
.modal-body {
  padding: 1.5rem;
  flex: 1;
  overflow-y: auto;
}

.no-areas {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary);
}

.areas-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.area-item {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 8px;
  transition: all 0.2s;
  cursor: pointer;
}

.area-item:hover {
  border-color: var(--accent-purple);
}

.area-item.selected {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
}

.area-checkbox {
  display: flex;
  align-items: flex-start;
  padding-top: 0.25rem;
  flex-shrink: 0;
}

.area-checkbox input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: var(--accent-purple);
}

.area-content {
  flex: 1;
}

.area-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

.area-title {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  flex: 1;
}

.area-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  flex-shrink: 0;
}

.area-name-block {
  flex: 1;
}

.area-name-block h3 {
  margin: 0 0 0.25rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.area-path {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
}

.area-meta {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.confidence-badge,
.type-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
}

.confidence-badge {
  background: var(--bg-primary);
  color: var(--text-secondary);
}

.confidence-badge.high {
  background: #d4edda;
  color: #155724;
}

.confidence-badge.medium {
  background: #fff3cd;
  color: #856404;
}

.confidence-badge.low {
  background: #f8d7da;
  color: #721c24;
}

.type-badge {
  background: var(--bg-primary);
  color: var(--text-secondary);
  text-transform: capitalize;
}

.area-description {
  margin: 0.5rem 0 0.5rem 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.area-footer {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.detected-type {
  font-size: 0.75rem;
  color: var(--text-secondary);
  background: var(--bg-primary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  text-transform: capitalize;
}

/* Modal Actions */
.modal-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.action-info {
  flex: 1;
}

.info-text,
.warning-text {
  margin: 0;
  font-size: 0.9rem;
}

.info-text {
  color: #28a745;
}

.warning-text {
  color: #ffc107;
}

.btn-cancel,
.btn-save {
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: none;
}

.btn-cancel {
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.btn-save {
  background: var(--accent-purple);
  color: white;
}

.btn-save:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

.btn-save:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--overlay-text);
  border-top-color: var(--overlay-text-active);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 640px) {
  .modal-content {
    width: 95%;
    max-height: 95vh;
  }

  .filter-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .btn-link {
    flex: 1;
  }

  .area-header {
    flex-direction: column;
  }

  .area-meta {
    width: 100%;
  }
}
</style>
