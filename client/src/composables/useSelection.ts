import { ref, computed } from 'vue'
import type { Book } from '@/types'

export function useSelection(getBooks: () => Book[]) {
  const selectedIds = ref<Set<string>>(new Set())

  const isSelected = (id: string) => selectedIds.value.has(id)
  const selectedCount = computed(() => selectedIds.value.size)
  const hasSelection = computed(() => selectedIds.value.size > 0)
  const allSelected = computed(
    () => getBooks().length > 0 && getBooks().every((b) => selectedIds.value.has(b.id)),
  )
  const selectedBooks = computed(() => getBooks().filter((b) => selectedIds.value.has(b.id)))

  function toggle(id: string) {
    // Always create a new Set — Vue reactivity doesn't track in-place Set mutation
    const next = new Set(selectedIds.value)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    selectedIds.value = next
  }

  function selectAll() {
    selectedIds.value = new Set(getBooks().map((b) => b.id))
  }

  function clearSelection() {
    selectedIds.value = new Set()
  }

  return {
    selectedIds,
    isSelected,
    selectedCount,
    hasSelection,
    allSelected,
    selectedBooks,
    toggle,
    selectAll,
    clearSelection,
  }
}
