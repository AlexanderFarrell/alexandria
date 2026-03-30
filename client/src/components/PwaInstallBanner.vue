<template>
  <Transition name="install-slide">
    <div v-if="isInstallable && !dismissed" class="install-banner" role="banner">
      <div class="install-banner-content">
        <svg class="install-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M4 19h16M12 3v12m0 0-4-4m4 4 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <div class="install-text">
          <strong>Install Alexandria</strong>
          <span>Add to your home screen for offline access</span>
        </div>
        <button class="btn-primary install-btn" @click="handleInstall">Install</button>
        <button class="btn-ghost dismiss-btn" @click="dismissed = true" aria-label="Dismiss">✕</button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { usePwaInstall } from '@/composables/usePwaInstall'

const { isInstallable, promptInstall } = usePwaInstall()
const dismissed = ref(false)

async function handleInstall() {
  await promptInstall()
  dismissed.value = true
}
</script>

<style scoped>
.install-banner {
  position: fixed;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9000;
  width: min(480px, calc(100vw - 2rem));
}

.install-banner-content {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background: var(--surface);
  border: 1px solid var(--accent);
  border-radius: var(--radius);
  padding: 0.85rem 1rem;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.6);
}

.install-icon {
  width: 1.5rem;
  height: 1.5rem;
  flex-shrink: 0;
  color: var(--accent);
}

.install-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  gap: 0.1rem;
}

.install-text strong {
  color: var(--accent);
  font-size: 0.95rem;
}

.install-text span {
  font-size: 0.8rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.install-btn {
  flex-shrink: 0;
  padding: 0.4rem 0.9rem;
  font-size: 0.85rem;
}

.dismiss-btn {
  flex-shrink: 0;
  padding: 0.4rem 0.5rem;
  font-size: 0.8rem;
  border: none;
  color: var(--text-muted);
}

.install-slide-enter-active,
.install-slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.install-slide-enter-from,
.install-slide-leave-to {
  transform: translateX(-50%) translateY(120%);
  opacity: 0;
}
</style>
