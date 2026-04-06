<template>
  <ul class="reader-nav-tree" :id="listId">
    <li v-for="(item, index) in items" :key="nodeKey(index)" class="reader-nav-item">
      <div class="reader-nav-row">
        <button
          v-if="hasChildren(item) && item.section_id"
          class="reader-nav-disclosure"
          type="button"
          :aria-controls="childListId(nodeKey(index))"
          :aria-expanded="isExpanded(nodeKey(index))"
          :aria-label="disclosureLabel(item, isExpanded(nodeKey(index)))"
          @click="toggleBranch(nodeKey(index))"
        >
          <span
            class="reader-nav-disclosure-icon"
            :class="{ 'is-expanded': isExpanded(nodeKey(index)) }"
            aria-hidden="true"
          ></span>
        </button>

        <button
          v-if="item.section_id"
          class="reader-nav-link"
          :class="{ 'is-active': isActive(item) }"
          type="button"
          @click="$emit('navigate', item.section_id, item.fragment)"
        >
          <span class="reader-nav-text">{{ item.label }}</span>
          <span v-if="item.fragment" class="reader-nav-fragment">#</span>
        </button>

        <button
          v-else-if="hasChildren(item)"
          class="reader-nav-group-toggle"
          type="button"
          :aria-controls="childListId(nodeKey(index))"
          :aria-expanded="isExpanded(nodeKey(index))"
          :aria-label="disclosureLabel(item, isExpanded(nodeKey(index)))"
          @click="toggleBranch(nodeKey(index))"
        >
          <span
            class="reader-nav-disclosure-icon"
            :class="{ 'is-expanded': isExpanded(nodeKey(index)) }"
            aria-hidden="true"
          ></span>
          <span class="reader-nav-text">{{ item.label }}</span>
        </button>

        <div v-else class="reader-nav-group">{{ item.label }}</div>
      </div>

      <ReaderNavTree
        v-if="hasChildren(item) && isExpanded(nodeKey(index))"
        :items="item.children ?? []"
        :active-section-id="activeSectionId"
        :active-fragment="activeFragment"
        :path-prefix="nodeKey(index)"
        :list-id="childListId(nodeKey(index))"
        @navigate="emitNavigate"
      />
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed, inject, provide, ref, watch } from 'vue'
import type { InjectionKey } from 'vue'
import type { ReaderNavItem } from '@/types'

interface ReaderNavTreeController {
  expand(keys: string[]): void
  isExpanded(key: string): boolean
  toggle(key: string): void
}

const readerNavTreeControllerKey: InjectionKey<ReaderNavTreeController> = Symbol('reader-nav-tree-controller')

const props = withDefaults(defineProps<{
  items: ReaderNavItem[]
  activeSectionId: string
  activeFragment: string
  pathPrefix?: string
  listId?: string
}>(), {
  pathPrefix: '',
})

const emit = defineEmits<{
  (e: 'navigate', sectionId: string, fragment?: string): void
}>()

const parentController = inject(readerNavTreeControllerKey, null)
const controller = parentController ?? createController()

if (!parentController) {
  provide(readerNavTreeControllerKey, controller)
}

const activeBranchKeys = computed(() => collectActiveBranchKeys(props.items, props.pathPrefix))

if (!parentController) {
  watch(activeBranchKeys, (keys) => {
    controller.expand(keys)
  }, { immediate: true })
}

function createController(): ReaderNavTreeController {
  const expandedKeys = ref(new Set<string>())

  return {
    expand(keys: string[]) {
      if (keys.length === 0) return
      const next = new Set(expandedKeys.value)
      let changed = false

      keys.forEach((key) => {
        if (next.has(key)) return
        next.add(key)
        changed = true
      })

      if (changed) expandedKeys.value = next
    },
    isExpanded(key: string) {
      return expandedKeys.value.has(key)
    },
    toggle(key: string) {
      const next = new Set(expandedKeys.value)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      expandedKeys.value = next
    },
  }
}

function emitNavigate(sectionId: string, fragment?: string) {
  emit('navigate', sectionId, fragment)
}

function hasChildren(item: ReaderNavItem): boolean {
  return Boolean(item.children && item.children.length > 0)
}

function nodeKey(index: number): string {
  return props.pathPrefix ? `${props.pathPrefix}.${index}` : String(index)
}

function childListId(key: string): string {
  return `reader-nav-branch-${key.replace(/[^a-zA-Z0-9_-]+/g, '-')}`
}

function toggleBranch(key: string) {
  controller.toggle(key)
}

function isExpanded(key: string): boolean {
  return controller.isExpanded(key)
}

function disclosureLabel(item: ReaderNavItem, expanded: boolean): string {
  const action = expanded ? 'Collapse' : 'Expand'
  if (item.section_id) return `${action} subsections for ${item.label}`
  return `${action} ${item.label}`
}

function isActive(item: ReaderNavItem): boolean {
  if (!item.section_id || item.section_id !== props.activeSectionId) return false
  if (!item.fragment) return true
  return item.fragment === props.activeFragment
}

function collectActiveBranchKeys(items: ReaderNavItem[], parentPath: string): string[] {
  if (!props.activeSectionId) return []

  if (props.activeFragment) {
    const exactFragmentBranch = collectMatchingBranchKeys(items, parentPath, (item) => (
      item.section_id === props.activeSectionId && item.fragment === props.activeFragment
    ))
    if (exactFragmentBranch.length > 0) return exactFragmentBranch
  }

  const sectionBranch = collectMatchingBranchKeys(items, parentPath, (item) => (
    item.section_id === props.activeSectionId && !item.fragment
  ))
  if (sectionBranch.length > 0) return sectionBranch

  return collectMatchingBranchKeys(items, parentPath, (item) => item.section_id === props.activeSectionId)
}

function collectMatchingBranchKeys(
  items: ReaderNavItem[],
  parentPath: string,
  matcher: (item: ReaderNavItem) => boolean,
): string[] {
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index]
    const key = parentPath ? `${parentPath}.${index}` : String(index)

    if (item.children?.length) {
      const childMatch = collectMatchingBranchKeys(item.children, key, matcher)
      if (childMatch.length > 0) return [key, ...childMatch]
    }

    if (matcher(item)) {
      return [key]
    }
  }

  return []
}
</script>

<style scoped>
.reader-nav-tree {
  list-style: none;
  margin: 0;
  padding: 0;
}

.reader-nav-item + .reader-nav-item {
  margin-top: 0.28rem;
}

.reader-nav-item > .reader-nav-tree {
  margin-top: 0.22rem;
  margin-left: 0.8rem;
  padding-left: 0.78rem;
  border-left: 1px solid var(--reader-nav-border, rgba(130, 130, 130, 0.2));
}

.reader-nav-row {
  display: flex;
  align-items: stretch;
  gap: 0.35rem;
}

.reader-nav-disclosure,
.reader-nav-link,
.reader-nav-group-toggle,
.reader-nav-group {
  border: 0;
  border-radius: 0.8rem;
  font: inherit;
}

.reader-nav-disclosure {
  flex-shrink: 0;
  width: 2rem;
  min-height: 2.25rem;
  padding: 0;
  background: transparent;
  color: var(--reader-muted, inherit);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.reader-nav-link,
.reader-nav-group-toggle,
.reader-nav-group {
  flex: 1 1 auto;
  min-width: 0;
  text-align: left;
  padding: 0.5rem 0.65rem;
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.reader-nav-link,
.reader-nav-group-toggle {
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.reader-nav-link:hover,
.reader-nav-group-toggle:hover,
.reader-nav-disclosure:hover {
  background: rgba(127, 127, 127, 0.08);
}

.reader-nav-link.is-active {
  background: var(--reader-nav-active-bg, rgba(200, 169, 110, 0.14));
  color: var(--reader-nav-active-text, inherit);
}

.reader-nav-group-toggle,
.reader-nav-group {
  color: var(--reader-muted, var(--text-muted));
  font-weight: 600;
}

.reader-nav-text {
  flex: 1;
  min-width: 0;
}

.reader-nav-fragment {
  opacity: 0.6;
  font-size: 0.82rem;
}

.reader-nav-disclosure-icon {
  width: 0.5rem;
  height: 0.5rem;
  border-right: 1.5px solid currentColor;
  border-bottom: 1.5px solid currentColor;
  transform: translateY(-0.05rem) rotate(-45deg);
  transition: transform 0.16s ease;
}

.reader-nav-disclosure-icon.is-expanded {
  transform: translateY(-0.14rem) rotate(45deg);
}
</style>
