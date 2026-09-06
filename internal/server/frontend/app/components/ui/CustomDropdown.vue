<template>
  <div class="custom-dropdown" :class="{ 'dropdown-open': isOpen, 'dropdown-disabled': disabled }">
    <!-- Dropdown Trigger -->
    <div
      class="dropdown-trigger"
      @click="toggleDropdown"
      @keydown="handleKeydown"
      :tabindex="disabled ? -1 : 0"
      role="combobox"
      :aria-expanded="isOpen"
      :aria-haspopup="true"
      :aria-labelledby="labelId"
      :aria-controls="menuId"
    >
      <slot name="trigger" :selectedOption="selectedOption">
        <div class="trigger-content">
          <div class="trigger-left">
            <span v-if="selectedOption?.icon" class="trigger-icon">{{ selectedOption.icon }}</span>
            <span class="selected-value">{{ selectedLabel }}</span>
          </div>
          <div class="dropdown-arrow" :class="{ 'arrow-open': isOpen }">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </div>
        </div>
      </slot>
    </div>

    <!-- Dropdown Menu -->
    <div
      v-if="isOpen"
      class="dropdown-menu"
      :id="menuId"
      role="listbox"
      :aria-labelledby="labelId"
    >
      <!-- Search Input (if searchable) -->
      <div v-if="searchable" class="dropdown-search">
        <input
          v-model="searchQuery"
          @input="filterOptions"
          @keydown="handleSearchKeydown"
          placeholder="Search..."
          ref="searchInput"
          type="text"
          class="search-input"
        />
      </div>

      <!-- Options List -->
      <div class="dropdown-options">
        <div
          v-for="(option, index) in filteredOptions"
          :key="option.value"
          class="dropdown-option"
          :class="{
            'option-selected': isSelected(option),
            'option-disabled': option.disabled,
            'option-hovered': hoveredIndex === index
          }"
          @click="selectOption(option)"
          @mouseenter="hoveredIndex = index"
          @mouseleave="hoveredIndex = -1"
          role="option"
          :aria-selected="isSelected(option)"
          :aria-disabled="option.disabled"
        >
          <slot name="option" :option="option">
            <div class="option-content">
              <div v-if="option.icon" class="option-icon">{{ option.icon }}</div>
              <div class="option-text">
                <div class="option-label">{{ option.label }}</div>
                <div v-if="option.description" class="option-description">{{ option.description }}</div>
              </div>
              <div v-if="isSelected(option)" class="option-checkmark">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
              </div>
            </div>
          </slot>
        </div>
      </div>

      <!-- No Results -->
      <div v-if="filteredOptions.length === 0" class="dropdown-no-results">
        No results found
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onMounted, onUnmounted } from 'vue'

interface DropdownOption {
  value: string | number
  label: string
  icon?: string
  description?: string
  disabled?: boolean
}

interface Props {
  modelValue: string | number
  options: DropdownOption[]
  searchable?: boolean
  disabled?: boolean
  placeholder?: string
  labelId?: string
}

const props = withDefaults(defineProps<Props>(), {
  searchable: false,
  disabled: false,
  placeholder: 'Select an option',
  labelId: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

// Reactive state
const isOpen = ref(false)
const searchQuery = ref('')
const hoveredIndex = ref(-1)

// Refs
const searchInput = ref<HTMLInputElement | null>(null)

// Computed properties
const selectedOption = computed(() => {
  return props.options.find(option => option.value === props.modelValue)
})

const selectedLabel = computed(() => {
  return selectedOption.value?.label || props.placeholder
})

const filteredOptions = computed(() => {
  if (!props.searchable || !searchQuery.value) {
    return props.options
  }

  const query = searchQuery.value.toLowerCase()
  return props.options.filter(option =>
    option.label.toLowerCase().includes(query) ||
    option.description?.toLowerCase().includes(query)
  )
})

const menuId = computed(() => `dropdown-menu-${Math.random().toString(36).substr(2, 9)}`)

// Methods
const toggleDropdown = () => {
  if (props.disabled) return

  isOpen.value = !isOpen.value

  if (isOpen.value && props.searchable) {
    nextTick(() => {
      searchInput.value?.focus()
    })
  }
}

const selectOption = (option: DropdownOption) => {
  if (option.disabled) return

  emit('update:modelValue', option.value)
  isOpen.value = false
  searchQuery.value = ''
  hoveredIndex.value = -1
}

const isSelected = (option: DropdownOption) => {
  return option.value === props.modelValue
}

const filterOptions = () => {
  hoveredIndex.value = -1
}

const handleKeydown = (event: KeyboardEvent) => {
  if (props.disabled) return

  switch (event.key) {
    case 'Enter':
    case ' ':
      event.preventDefault()
      toggleDropdown()
      break
    case 'ArrowDown':
      event.preventDefault()
      if (!isOpen.value) {
        toggleDropdown()
      } else {
        hoveredIndex.value = Math.min(hoveredIndex.value + 1, filteredOptions.value.length - 1)
      }
      break
    case 'ArrowUp':
      event.preventDefault()
      if (!isOpen.value) {
        toggleDropdown()
      } else {
        hoveredIndex.value = Math.max(hoveredIndex.value - 1, 0)
      }
      break
    case 'Escape':
      if (isOpen.value) {
        event.preventDefault()
        isOpen.value = false
        hoveredIndex.value = -1
      }
      break
  }
}

const handleSearchKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      hoveredIndex.value = Math.min(hoveredIndex.value + 1, filteredOptions.value.length - 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      hoveredIndex.value = Math.max(hoveredIndex.value - 1, 0)
      break
    case 'Enter':
      event.preventDefault()
      if (hoveredIndex.value >= 0 && hoveredIndex.value < filteredOptions.value.length) {
        selectOption(filteredOptions.value[hoveredIndex.value])
      }
      break
    case 'Escape':
      event.preventDefault()
      isOpen.value = false
      hoveredIndex.value = -1
      break
  }
}

// Watch for hovered index changes to scroll into view
watch(hoveredIndex, (newIndex) => {
  if (newIndex >= 0 && isOpen.value) {
    nextTick(() => {
      const optionElements = document.querySelectorAll('.dropdown-option')
      if (optionElements[newIndex]) {
        optionElements[newIndex].scrollIntoView({
          block: 'nearest',
          behavior: 'smooth'
        })
      }
    })
  }
})

// Close dropdown when clicking outside
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (!target.closest('.custom-dropdown')) {
    isOpen.value = false
    hoveredIndex.value = -1
    searchQuery.value = ''
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.custom-dropdown {
  position: relative;
  width: 100%;
}

.dropdown-trigger {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.95rem;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.dropdown-trigger:hover:not(.dropdown-disabled) {
  border-color: var(--accent-purple);
  background: var(--bg-secondary);
}

.dropdown-trigger:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.dropdown-disabled .dropdown-trigger {
  opacity: 0.6;
  cursor: not-allowed;
  background: var(--bg-secondary);
}

.dropdown-disabled .dropdown-trigger:hover {
  border-color: var(--border-color);
  background: var(--bg-secondary);
}

.trigger-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.trigger-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.trigger-icon {
  font-size: 1.1rem;
  flex-shrink: 0;
}

.selected-value {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s ease;
  color: var(--text-secondary);
}

.arrow-open {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
  z-index: 1000;
  margin-top: 4px;
  max-height: 300px;
  overflow: hidden;
  animation: dropdownSlideIn 0.2s ease;
}

@keyframes dropdownSlideIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.dropdown-search {
  padding: 12px;
  border-bottom: 1px solid var(--border-color);
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.9rem;
  transition: all 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.dropdown-options {
  max-height: 250px;
  overflow-y: auto;
  padding: 4px 0;
}

.dropdown-option {
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.15s ease;
  display: flex;
  align-items: center;
  gap: 12px;
}

.dropdown-option:hover:not(.option-disabled),
.dropdown-option.option-hovered:not(.option-disabled) {
  background: var(--bg-tertiary);
}

.dropdown-option.option-selected {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  color: var(--accent-purple);
}

.dropdown-option.option-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option-content {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.option-icon {
  font-size: 1.1rem;
  flex-shrink: 0;
  width: 20px;
  text-align: center;
}

.option-text {
  flex: 1;
  min-width: 0;
}

.option-label {
  font-weight: 500;
  font-size: 0.95rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.option-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.option-checkmark {
  color: var(--accent-purple);
  flex-shrink: 0;
}

.dropdown-no-results {
  padding: 20px 16px;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.9rem;
}

/* Scrollbar styling for dropdown options */
.dropdown-options::-webkit-scrollbar {
  width: 6px;
}

.dropdown-options::-webkit-scrollbar-track {
  background: var(--bg-secondary);
}

.dropdown-options::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.dropdown-options::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Responsive */
@media (max-width: 640px) {
  .dropdown-menu {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 90vw;
    max-width: 400px;
    max-height: 70vh;
  }
}
</style>