<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="card">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-opanel-text-muted">Domains</p>
            <p class="text-2xl font-bold text-white mt-1">{{ domains.length }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-opanel-primary/10 flex items-center justify-center">
            <svg class="w-6 h-6 text-opanel-primary" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
            </svg>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-opanel-text-muted">Databases</p>
            <p class="text-2xl font-bold text-white mt-1">{{ databases.length }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-green-900/30 flex items-center justify-center">
            <svg class="w-6 h-6 text-green-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
            </svg>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-opanel-text-muted">Active Domains</p>
            <p class="text-2xl font-bold text-white mt-1">{{ activeDomains }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-yellow-900/30 flex items-center justify-center">
            <svg class="w-6 h-6 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
            </svg>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-opanel-text-muted">Users</p>
            <p class="text-2xl font-bold text-white mt-1">{{ userCount }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl bg-purple-900/30 flex items-center justify-center">
            <svg class="w-6 h-6 text-purple-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 0 0 2.625.372 9.337 9.337 0 0 0 4.121-.952 4.125 4.125 0 0 0-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 0 1 8.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0 1 11.964-3.07M12 6.375a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0Zm8.25 2.25a2.625 2.625 0 1 1-5.25 0 2.625 2.625 0 0 1 5.25 0Z" />
            </svg>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <h2 class="text-lg font-semibold text-white mb-4">Recent Domains</h2>
        <div v-if="domains.length === 0" class="text-center py-8 text-opanel-text-muted">
          No domains yet. Create your first domain to get started.
        </div>
        <div v-else class="space-y-3">
          <router-link
            v-for="domain in domains.slice(0, 5)"
            :key="domain.id"
            :to="`/domains/${domain.id}`"
            class="flex items-center justify-between p-3 bg-opanel-bg rounded-lg hover:bg-opanel-bg/80 transition-colors cursor-pointer"
          >
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-opanel-primary/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-opanel-primary" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
                </svg>
              </div>
              <span class="text-sm font-medium text-white">{{ domain.name }}</span>
            </div>
            <span :class="statusClass(domain.status)">{{ domain.status }}</span>
          </router-link>
        </div>
      </div>

      <div class="card">
        <h2 class="text-lg font-semibold text-white mb-4">Recent Databases</h2>
        <div v-if="databases.length === 0" class="text-center py-8 text-opanel-text-muted">
          No databases yet. Create your first database to get started.
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="db in databases.slice(0, 5)"
            :key="db.id"
            class="flex items-center justify-between p-3 bg-opanel-bg rounded-lg"
          >
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-green-900/30 flex items-center justify-center">
                <svg class="w-4 h-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
                </svg>
              </div>
              <span class="text-sm font-medium text-white">{{ db.name }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api'
import type { Domain, Database } from '@/types'

const domains = ref<Domain[]>([])
const databases = ref<Database[]>([])
const userCount = ref(0)

const activeDomains = computed(() => domains.value.filter(d => d.status === 'active').length)

function statusClass(status: string) {
  const base = 'badge '
  switch (status) {
    case 'active': return base + 'badge-success'
    case 'suspended': return base + 'badge-danger'
    case 'pending': return base + 'badge-warning'
    default: return base + 'badge-warning'
  }
}

onMounted(async () => {
  try {
    const [d, db, u] = await Promise.all([
      api.listDomains(),
      api.listDatabases(),
      api.listUsers().catch(() => []),
    ])
    domains.value = d
    databases.value = db
    userCount.value = u.length
  } catch (e) {
    console.error('Failed to load dashboard:', e)
  }
})
</script>
