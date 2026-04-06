<template>
  <div class="batch-bar">
    <span class="batch-count">{{ count }} selected</span>

    <div class="batch-actions">
      <template v-if="lists !== undefined">
        <div class="list-picker-wrap">
          <select v-model="selectedListId" class="list-select">
            <option value="" disabled>Add to list…</option>
            <option v-for="list in lists" :key="list.id" :value="list.id">{{ list.name }}</option>
          </select>
          <button
            class="btn-primary batch-btn"
            :disabled="!selectedListId"
            @click="onAddToList"
          >Add to list</button>
        </div>
      </template>

      <template v-if="showRemove">
        <button class="btn-ghost batch-btn" @click="$emit('remove')">Remove from list</button>
      </template>

      <template v-if="!confirmingDelete">
        <button class="btn-danger batch-btn" @click="confirmingDelete = true">Delete</button>
      </template>
      <template v-else>
        <span class="confirm-msg">Delete {{ count }} book{{ count !== 1 ? 's' : '' }}?</span>
        <button class="btn-danger batch-btn" @click="onConfirmDelete">Confirm</button>
        <button class="btn-ghost batch-btn" @click="confirmingDelete = false">Cancel</button>
      </template>
    </div>

    <button class="batch-clear btn-ghost" @click="$emit('clear')" title="Clear selection">✕</button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { BookList } from '@/types'

const props = defineProps<{
  count: number
  lists?: BookList[]
  showRemove?: boolean
}>()

const emit = defineEmits<{
  (e: 'clear'): void
  (e: 'delete'): void
  (e: 'add-to-list', listId: string): void
  (e: 'remove'): void
}>()

const selectedListId = ref('')
const confirmingDelete = ref(false)

function onAddToList() {
  if (selectedListId.value) {
    emit('add-to-list', selectedListId.value)
    selectedListId.value = ''
  }
}

function onConfirmDelete() {
  confirmingDelete.value = false
  emit('delete')
}

// Reset confirm state when count changes (e.g. after partial delete)
watch(() => props.count, () => {
  confirmingDelete.value = false
})
</script>

<style scoped>
.batch-bar {
  position: fixed;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 150;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 0.75rem;
  padding: 0.6rem 1rem;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
  white-space: nowrap;
}

.batch-count {
  font-size: 0.85rem;
  color: var(--text-muted);
  padding-right: 0.25rem;
  border-right: 1px solid var(--border);
  margin-right: 0.25rem;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.batch-btn {
  font-size: 0.8rem;
  padding: 0.3rem 0.7rem;
}

.list-picker-wrap {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.list-select {
  background: var(--surface-hover);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 0.4rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
  cursor: pointer;
}

.confirm-msg {
  font-size: 0.82rem;
  color: var(--danger);
}

.btn-danger {
  background: var(--danger);
  color: #fff;
  border: none;
  border-radius: 0.4rem;
  cursor: pointer;
  font-size: 0.8rem;
  padding: 0.3rem 0.7rem;
  transition: opacity 0.15s;
}
.btn-danger:hover {
  opacity: 0.85;
}

.batch-clear {
  font-size: 0.75rem;
  padding: 0.2rem 0.4rem;
  margin-left: 0.25rem;
  color: var(--text-muted);
}
</style>
