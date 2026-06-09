<template>
  <div class="space-y-6">
    <h2 class="text-lg font-semibold text-white">Settings</h2>

    <div class="card">
      <h3 class="text-md font-medium text-white mb-4">Server Information</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm text-opanel-text-muted">OPanel Version</label>
          <p class="text-sm text-white mt-1">0.1.0</p>
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted">API Base URL</label>
          <p class="text-sm text-white mt-1">/api</p>
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted">Logged in as</label>
          <p class="text-sm text-white mt-1">{{ auth.user?.username }} ({{ auth.user?.role }})</p>
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted">Token Expiry</label>
          <p class="text-sm text-white mt-1">24 hours</p>
        </div>
      </div>
    </div>

    <div class="card">
      <h3 class="text-md font-medium text-white mb-4">Change Password</h3>
      <form @submit.prevent="changePassword" class="space-y-4 max-w-md">
        <div>
          <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Current Password</label>
          <input v-model="currentPassword" type="password" class="input" placeholder="Current password" required />
        </div>
        <div>
          <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">New Password</label>
          <input v-model="newPassword" type="password" class="input" placeholder="New password" required />
        </div>
        <div>
          <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Confirm New Password</label>
          <input v-model="confirmPassword" type="password" class="input" placeholder="Confirm new password" required />
        </div>
        <div v-if="pwError" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
          {{ pwError }}
        </div>
        <div v-if="pwSuccess" class="bg-green-900/30 border border-green-800 text-green-400 text-sm rounded-lg px-4 py-3">
          {{ pwSuccess }}
        </div>
        <div>
          <button type="submit" class="btn-primary text-sm" :disabled="changingPw">
            {{ changingPw ? 'Saving...' : 'Change Password' }}
          </button>
        </div>
      </form>
    </div>

    <div class="card">
      <h3 class="text-md font-medium text-white mb-4">Quick Actions</h3>
      <div class="space-y-3">
        <button @click="handleLogout" class="btn-danger text-sm">
          Sign Out
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api'

const router = useRouter()
const auth = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const changingPw = ref(false)
const pwError = ref<string | null>(null)
const pwSuccess = ref<string | null>(null)

async function changePassword() {
  pwError.value = null
  pwSuccess.value = null

  if (newPassword.value !== confirmPassword.value) {
    pwError.value = 'New passwords do not match'
    return
  }

  if (newPassword.value.length < 6) {
    pwError.value = 'New password must be at least 6 characters'
    return
  }

  if (!auth.user) return

  changingPw.value = true
  try {
    await api.updateUser(auth.user.id, { password: newPassword.value })
    pwSuccess.value = 'Password changed successfully'
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e: unknown) {
    pwError.value = e instanceof Error ? e.message : 'Failed to change password'
  } finally {
    changingPw.value = false
  }
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>
