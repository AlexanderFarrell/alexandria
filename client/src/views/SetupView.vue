<template>
  <div class="setup-page">
    <div class="setup-box card">
      <h1 class="brand">Alexandria</h1>
      <p class="tagline">Connect to your library</p>
      <p class="hint">Enter the address of your Alexandria server.</p>

      <form @submit.prevent="submit">
        <div class="field">
          <label>Server URL</label>
          <input
            v-model="serverUrl"
            type="url"
            placeholder="https://books.example.com"
            autocomplete="url"
            autocorrect="off"
            autocapitalize="none"
            spellcheck="false"
            required
          />
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary submit-btn" :disabled="loading">
          {{ loading ? 'Connecting…' : 'Connect' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useServerConfigStore } from '@/stores/serverConfig'

const router = useRouter()
const serverConfig = useServerConfigStore()

const serverUrl = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true

  const url = serverUrl.value.replace(/\/$/, '')

  try {
    // Probe the server using native fetch so Tauri's HTTP plugin handles it
    // (bypasses CORS — XHR/axios would be blocked by the WebView)
    const res = await fetch(`${url}/api/v1/auth/status`)
    if (!res.ok) throw new Error(`status ${res.status}`)
    await serverConfig.save(url)
    router.push('/login')
  } catch {
    error.value = 'Could not reach the server. Check the URL and try again.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: var(--bg);
}

.setup-box {
  width: 100%;
  max-width: 400px;
  padding: 2.5rem 2rem;
}

.brand {
  font-size: 2rem;
  font-weight: 700;
  color: var(--accent);
  font-family: var(--font-body);
  text-align: center;
}

.tagline {
  text-align: center;
  color: var(--text-muted);
  font-size: 0.9rem;
  margin-top: 0.3rem;
  margin-bottom: 0.25rem;
}

.hint {
  text-align: center;
  color: var(--text-muted);
  font-size: 0.88rem;
  margin-bottom: 1.5rem;
}

form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.field label {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.submit-btn {
  width: 100%;
  margin-top: 0.5rem;
}
</style>
