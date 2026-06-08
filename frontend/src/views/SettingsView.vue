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
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>
