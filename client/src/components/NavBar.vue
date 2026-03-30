<template>
  <nav class="navbar">
    <router-link to="/library" class="brand">Alexandria</router-link>
    <div class="nav-links">
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
  padding: 0 2rem;
  height: 56px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 100;
}
.brand {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--accent);
  letter-spacing: 0.04em;
  text-decoration: none;
}
.nav-links {
  display: flex;
  gap: 0.25rem;
  margin-left: 1.5rem;
}
.nav-link {
  padding: 0.3rem 0.75rem;
  border-radius: var(--radius);
  font-size: 0.9rem;
  color: var(--text-muted);
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}
.nav-link:hover { color: var(--text); text-decoration: none; }
.nav-link.router-link-active { color: var(--accent); background: var(--surface-hover); }
</style>
