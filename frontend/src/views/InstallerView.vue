<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-white">One-Click Installer</h2>
    </div>

    <div v-if="!selectedApp" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <button
        @click="selectApp('wordpress')"
        class="card hover:border-opanel-primary/50 transition-colors cursor-pointer text-left"
      >
        <div class="flex items-center gap-4">
          <div class="w-14 h-14 rounded-xl bg-blue-900/30 flex items-center justify-center">
            <span class="text-3xl font-bold text-blue-400">W</span>
          </div>
          <div>
            <h3 class="text-lg font-semibold text-white">WordPress</h3>
            <p class="text-sm text-opanel-text-muted">The world's most popular CMS</p>
          </div>
        </div>
        <div class="mt-3 text-xs text-opanel-text-muted">
          Creates database, downloads latest WordPress, installs with WP-CLI
        </div>
      </button>
    </div>

    <div v-else class="card max-w-2xl">
      <div class="flex items-center gap-4 mb-6">
        <button @click="selectedApp = null" class="text-opanel-text-muted hover:text-white transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
          </svg>
        </button>
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-blue-900/30 flex items-center justify-center">
            <span class="text-xl font-bold text-blue-400">W</span>
          </div>
          <div>
            <h3 class="text-lg font-semibold text-white">Install WordPress</h3>
            <p class="text-sm text-opanel-text-muted">Configure and install on your domain</p>
          </div>
        </div>
      </div>

      <form @submit.prevent="installWordPress" class="space-y-4">
        <div>
          <label class="block text-sm text-opanel-text-muted mb-1">Domain</label>
          <select v-model="form.domain_id" class="input" required>
            <option value="">Select domain...</option>
            <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.name }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted mb-1">Site Title</label>
          <input v-model="form.site_name" class="input" placeholder="My WordPress Site" required />
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted mb-1">Admin Username</label>
          <input v-model="form.admin_user" class="input" placeholder="admin" required />
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted mb-1">Admin Password</label>
          <input v-model="form.admin_pass" type="password" class="input" placeholder="Strong password" required />
        </div>
        <div>
          <label class="block text-sm text-opanel-text-muted mb-1">Admin Email</label>
          <input v-model="form.admin_email" type="email" class="input" placeholder="admin@example.com" required />
        </div>

        <div v-if="error" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
          {{ error }}
        </div>

        <div v-if="result" class="bg-green-900/30 border border-green-800 text-green-400 text-sm rounded-lg px-4 py-3">
          WordPress installed successfully! Visit your domain to complete the setup.
        </div>

        <button type="submit" class="btn-primary w-full" :disabled="installing">
          {{ installing ? 'Installing...' : 'Install WordPress' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import type { Domain } from '@/types'

const toast = useToast()

const domains = ref<Domain[]>([])
const installing = ref(false)
const error = ref('')
const result = ref(false)
const selectedApp = ref<string | null>(null)

const form = ref({
  domain_id: '',
  site_name: '',
  admin_user: 'admin',
  admin_pass: '',
  admin_email: '',
})

function selectApp(app: string) {
  selectedApp.value = app
  error.value = ''
  result.value = false
}

async function loadDomains() {
  try {
    domains.value = await api.listDomains()
  } catch {}
}

async function installWordPress() {
  installing.value = true
  error.value = ''
  result.value = false

  const domain = domains.value.find(d => d.id === Number(form.value.domain_id))

  try {
    const resp = await fetch('/api/wordpress/install', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: JSON.stringify({
        domain_id: Number(form.value.domain_id),
        domain_name: domain?.name || '',
        site_name: form.value.site_name,
        admin_user: form.value.admin_user,
        admin_password: form.value.admin_pass,
        admin_email: form.value.admin_email,
      })
    })

    if (!resp.ok) {
      const data = await resp.json()
      throw new Error(data.error || 'Installation failed')
    }

    result.value = true
    toast.success('WordPress installed!')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Installation failed'
    toast.error('Installation failed')
  } finally {
    installing.value = false
  }
}

onMounted(loadDomains)
</script>
