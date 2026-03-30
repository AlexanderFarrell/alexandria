<template>
  <div class="login-page">
    <div class="login-box card">
      <h1 class="brand">Alexandria</h1>
      <p class="tagline">Your personal library</p>
      <p class="status-msg" :class="{ 'error-msg': !!authStatusError }">{{ authHint }}</p>

      <div class="tabs">
        <button :class="['tab', { active: mode === 'login' }]" @click="mode = 'login'">Sign in</button>
        <button v-if="canRegister" :class="['tab', { active: mode === 'register' }]" @click="mode = 'register'">
          Register
        </button>
      </div>

      <form @submit.prevent="submit">
        <div class="field">
          <label>Username</label>
          <input v-model="username" type="text" placeholder="your username" autocomplete="username" required />
        </div>
        <div class="field">
          <label>Password</label>
          <input
            v-model="password"
            type="password"
            placeholder="••••••••"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            required
          />
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary submit-btn" :disabled="loading">
          {{ loading ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Create account' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import * as authApi from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import type { AuthStatus } from '@/types'

const auth = useAuthStore()
const router = useRouter()

const mode = ref<'login' | 'register'>('login')
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const canRegister = ref(false)
const statusLoading = ref(true)
const authStatusError = ref('')
const registrationMode = ref<AuthStatus['registration_mode']>('single')

const authHint = computed(() => {
  if (statusLoading.value) return 'Checking registration availability…'
  if (authStatusError.value) return authStatusError.value
  if (registrationMode.value === 'single') {
    return canRegister.value
      ? 'Create the owner account to bootstrap this deployment.'
      : 'The owner account already exists. Sign in with that account.'
  }
  if (registrationMode.value === 'disable') {
    return 'Account registration is disabled. Sign in with an existing account.'
  }
  return 'Multi-user mode shares one library across all accounts.'
})

onMounted(async () => {
  try {
    const status = await authApi.getStatus()
    registrationMode.value = status.registration_mode
    canRegister.value = status.can_register
    if (!status.can_register && mode.value === 'register') {
      mode.value = 'login'
    }
  } catch {
    authStatusError.value = 'Failed to determine registration availability.'
    canRegister.value = false
    mode.value = 'login'
  } finally {
    statusLoading.value = false
  }
})

async function submit() {
  error.value = ''
  if (mode.value === 'register' && !canRegister.value) {
    error.value = 'Registration is currently unavailable.'
    return
  }
  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(username.value, password.value)
    } else {
      await auth.register(username.value, password.value)
    }
    router.push('/library')
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg ?? 'Something went wrong. Try again.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: var(--bg);
}

.login-box {
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
  margin-bottom: 0.75rem;
}

.status-msg {
  min-height: 2.5rem;
  margin-bottom: 1.25rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.88rem;
  line-height: 1.4;
}

.tabs {
  display: flex;
  gap: 0;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--border);
}
.tab {
  background: transparent;
  color: var(--text-muted);
  border-radius: 0;
  padding: 0.5rem 1rem;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
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
