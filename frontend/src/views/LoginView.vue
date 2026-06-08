<template>
  <div class="min-h-screen flex items-center justify-center bg-opanel-bg">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="w-16 h-16 bg-opanel-primary rounded-2xl flex items-center justify-center mx-auto mb-4">
          <span class="text-white font-bold text-2xl">O</span>
        </div>
        <h1 class="text-2xl font-bold text-white">OPanel</h1>
        <p class="text-opanel-text-muted mt-1">Server Control Panel</p>
      </div>

      <div class="card">
        <form @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Username</label>
            <input
              v-model="username"
              type="text"
              class="input"
              placeholder="admin"
              autocomplete="username"
              required
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Password</label>
            <input
              v-model="password"
              type="password"
              class="input"
              placeholder="Password"
              autocomplete="current-password"
              required
            />
          </div>

          <div v-if="auth.error" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
            {{ auth.error }}
          </div>

          <button
            type="submit"
            class="btn-primary w-full"
            :disabled="auth.loading"
          >
            <span v-if="auth.loading" class="flex items-center justify-center gap-2">
              <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              Signing in...
            </span>
            <span v-else>Sign In</span>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')

async function handleLogin() {
  try {
    await auth.login(username.value, password.value)
    router.push('/')
  } catch {
    // error is set in store
  }
}
</script>
