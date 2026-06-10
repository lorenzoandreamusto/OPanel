<template>
  <header class="h-16 bg-opanel-panel border-b border-opanel-border flex items-center justify-between px-6">
    <div class="flex items-center gap-4">
      <h1 class="text-lg font-semibold text-white">{{ pageTitle }}</h1>
    </div>
    <div class="flex items-center gap-4">
      <span class="text-sm text-opanel-text-muted">{{ currentTime }}</span>
      <button
        @click="handleLogout"
        class="btn-ghost text-sm"
      >
        Logout
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const currentTime = ref('')
let timer: ReturnType<typeof setInterval>

const pageTitle = computed(() => {
  const titles: Record<string, string> = {
    dashboard: 'Dashboard',
    domains: 'Domains',
    databases: 'Databases',
    mail: 'Mail',
    users: 'Users',
    settings: 'Settings',
  }
  return titles[route.name as string] || 'OPanel'
})

function updateTime() {
  currentTime.value = new Date().toLocaleTimeString()
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}

onMounted(() => {
  updateTime()
  timer = setInterval(updateTime, 1000)
})

onUnmounted(() => {
  clearInterval(timer)
})
</script>
