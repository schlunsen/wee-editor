<template>
  <div class="todo-write-preview">
    <div class="preview-header">
      <div class="header-icon">📋</div>
      <div class="header-title">Tasks</div>
      <div class="task-count">{{ todos.length }} {{ todos.length === 1 ? 'task' : 'tasks' }}</div>
    </div>
    <div class="todos-container">
      <div
        v-for="(todo, index) in todos"
        :key="index"
        class="todo-preview-item"
        :class="todo.status"
      >
        <div class="todo-status-indicator">
          <div v-if="todo.status === 'completed'" class="status-icon completed-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
          </div>
          <div v-else-if="todo.status === 'in_progress'" class="status-icon in-progress-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
            </svg>
          </div>
          <div v-else class="status-icon pending-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
            </svg>
          </div>
        </div>
        <div class="todo-details">
          <div class="todo-content-text">{{ todo.content }}</div>
          <div v-if="todo.activeForm && todo.status === 'in_progress'" class="todo-active-form-text">
            {{ todo.activeForm }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface TodoItem {
  content: string
  status: 'pending' | 'in_progress' | 'completed'
  activeForm?: string
}

interface Props {
  todos: TodoItem[]
}

defineProps<Props>()
</script>

<style scoped>
.todo-write-preview {
  background: var(--bg-secondary);
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.preview-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  background: linear-gradient(135deg, var(--accent-purple), var(--accent-purple-hover));
  color: white;
}

.header-icon {
  font-size: 1.3rem;
  line-height: 1;
}

.header-title {
  font-size: 1rem;
  font-weight: 600;
  flex: 1;
}

.task-count {
  font-size: 0.85rem;
  background: var(--overlay-bg-active);
  padding: 4px 10px;
  border-radius: 12px;
  font-weight: 500;
}

.todos-container {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 500px;
  overflow-y: auto;
}

.todo-preview-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background: var(--bg-primary);
  border-radius: 10px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.todo-preview-item:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  transform: translateX(-2px);
}

.todo-preview-item.completed {
  opacity: 0.75;
  background: var(--bg-secondary);
}

.todo-preview-item.in_progress {
  border-color: var(--accent-purple);
  background: linear-gradient(135deg, var(--bg-primary), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05));
}

.todo-status-indicator {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 2px;
}

.status-icon {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.completed-icon {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.in-progress-icon {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: var(--accent-purple);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.7;
    transform: scale(1.05);
  }
}

.pending-icon {
  background: rgba(148, 163, 184, 0.15);
  color: var(--text-secondary);
}

.todo-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.todo-content-text {
  font-size: 0.95rem;
  color: var(--text-primary);
  line-height: 1.5;
  word-wrap: break-word;
  font-weight: 500;
}

.todo-preview-item.completed .todo-content-text {
  text-decoration: line-through;
  color: var(--text-secondary);
  font-weight: 400;
}

.todo-active-form-text {
  font-size: 0.85rem;
  color: var(--accent-purple);
  font-style: italic;
  display: flex;
  align-items: center;
  gap: 6px;
}

.todo-active-form-text::before {
  content: '→';
  font-style: normal;
  font-weight: bold;
}

/* Empty state */
.todos-container:empty::after {
  content: 'No tasks found';
  display: block;
  text-align: center;
  padding: 24px;
  color: var(--text-secondary);
  font-style: italic;
}

/* Scrollbar styling */
.todos-container::-webkit-scrollbar {
  width: 8px;
}

.todos-container::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 4px;
}

.todos-container::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.todos-container::-webkit-scrollbar-thumb:hover {
  background: var(--accent-purple);
}

/* Mobile responsive */
@media (max-width: 768px) {
  .preview-header {
    padding: 12px 14px;
  }

  .header-icon {
    font-size: 1.1rem;
  }

  .header-title {
    font-size: 0.9rem;
  }

  .task-count {
    font-size: 0.8rem;
    padding: 3px 8px;
  }

  .todos-container {
    padding: 10px;
    gap: 8px;
  }

  .todo-preview-item {
    padding: 10px;
    gap: 10px;
  }

  .todo-content-text {
    font-size: 0.9rem;
  }

  .todo-active-form-text {
    font-size: 0.8rem;
  }
}
</style>
