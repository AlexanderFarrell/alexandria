<template>
  <ul class="reader-nav-tree">
    <li v-for="item in items" :key="itemKey(item)" class="reader-nav-item">
      <button
        v-if="item.section_id"
        class="reader-nav-link"
        :class="{ 'is-active': isActive(item) }"
        @click="$emit('navigate', item.section_id, item.fragment)"
      >
        <span class="reader-nav-text">{{ item.label }}</span>
        <span v-if="item.fragment" class="reader-nav-fragment">#</span>
      </button>
      <div v-else class="reader-nav-group">{{ item.label }}</div>

      <ReaderNavTree
        v-if="item.children?.length"
        :items="item.children"
        :active-section-id="activeSectionId"
        :active-fragment="activeFragment"
        @navigate="emitNavigate"
      />
    </li>
  </ul>
</template>

<script setup lang="ts">
import type { ReaderNavItem } from '@/types'

const props = defineProps<{
  items: ReaderNavItem[]
  activeSectionId: string
  activeFragment: string
}>()

const emit = defineEmits<{
  (e: 'navigate', sectionId: string, fragment?: string): void
}>()

function emitNavigate(sectionId: string, fragment?: string) {
  emit('navigate', sectionId, fragment)
}

function itemKey(item: ReaderNavItem): string {
  return `${item.section_id ?? 'group'}:${item.fragment ?? ''}:${item.label}`
}

function isActive(item: ReaderNavItem): boolean {
  if (!item.section_id || item.section_id !== props.activeSectionId) return false
  if (!item.fragment) return true
  return item.fragment === props.activeFragment
}
</script>

<style scoped>
.reader-nav-tree {
  list-style: none;
  margin: 0;
  padding: 0;
}

.reader-nav-item + .reader-nav-item {
  margin-top: 0.25rem;
}

.reader-nav-item > .reader-nav-tree {
  margin-top: 0.2rem;
  margin-left: 0.8rem;
  padding-left: 0.8rem;
  border-left: 1px solid var(--reader-nav-border, rgba(130, 130, 130, 0.2));
}

.reader-nav-link,
.reader-nav-group {
  width: 100%;
  text-align: left;
  border: 0;
  border-radius: 0.8rem;
  padding: 0.5rem 0.65rem;
  font: inherit;
}

.reader-nav-link {
  background: transparent;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.reader-nav-link:hover {
  background: rgba(127, 127, 127, 0.08);
}

.reader-nav-link.is-active {
  background: var(--reader-nav-active-bg, rgba(200, 169, 110, 0.14));
  color: var(--reader-nav-active-text, inherit);
}

.reader-nav-group {
  color: var(--text-muted);
  font-weight: 600;
}

.reader-nav-text {
  flex: 1;
}

.reader-nav-fragment {
  opacity: 0.6;
  font-size: 0.82rem;
}
</style>
