<template>
  <nav class="navbar" :class="{ 'is-compact-mobile': compactMobile }">
    <router-link to="/library" class="brand">Alexandria</router-link>
    <div class="nav-links" :class="{ 'is-hidden-mobile': compactMobile }">
      <router-link to="/library" class="nav-link">Library</router-link>
      <router-link to="/lists" class="nav-link">Lists</router-link>
    </div>
    <div class="nav-actions">
      <button class="btn-ghost" @click="logout">Sign out</button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

withDefaults(defineProps<{ compactMobile?: boolean }>(), {
  compactMobile: false,
})

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem 1rem;
  flex-wrap: wrap;
  padding: 0 2rem;
  min-height: 56px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 100;
}

.navbar.is-compact-mobile {
  min-height: 52px;
  padding: 0 1.35rem;
}

.brand {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--accent);
  letter-spacing: 0.04em;
  text-decoration: none;
  flex-shrink: 0;
}

.nav-links {
  display: flex;
  gap: 0.25rem;
  margin-left: 1.5rem;
}

.nav-link {
  padding: 0.28rem 0.72rem;
  border-radius: var(--radius);
  font-size: 0.9rem;
  color: var(--text-muted);
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}
.nav-link:hover { color: var(--text); text-decoration: none; }
.nav-link.router-link-active { color: var(--accent); background: var(--surface-hover); }

.nav-actions {
  margin-left: auto;
}

.nav-actions .btn-ghost {
  white-space: nowrap;
  padding: 0.52rem 0.9rem;
}

@media (max-width: 980px) {
  .nav-links.is-hidden-mobile {
    display: none;
  }
}

@media (max-width: 720px) {
  .navbar {
    padding: 0.7rem 1rem;
  }

  .brand {
    font-size: 1.1rem;
  }

  .nav-links {
    order: 3;
    width: 100%;
    margin-left: 0;
    gap: 0.4rem;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .nav-links::-webkit-scrollbar {
    display: none;
  }

  .nav-link {
    white-space: nowrap;
  }

  .nav-actions .btn-ghost {
    padding: 0.5rem 0.85rem;
  }

  .navbar.is-compact-mobile {
    flex-wrap: nowrap;
    min-height: 48px;
    padding: 0.48rem 0.8rem;
  }

  .navbar.is-compact-mobile .brand {
    font-size: 1rem;
  }

  .nav-links.is-hidden-mobile {
    display: none;
  }

  .navbar.is-compact-mobile .nav-actions .btn-ghost {
    padding: 0.4rem 0.68rem;
    font-size: 0.8rem;
  }
}

@media (max-width: 480px) {
  .navbar:not(.is-compact-mobile) {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: center;
  }

  .navbar:not(.is-compact-mobile) .nav-links {
    grid-column: 1 / -1;
  }
}
</style>
