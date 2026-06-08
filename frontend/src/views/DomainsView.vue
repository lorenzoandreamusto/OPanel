<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-white">All Domains</h2>
      <button @click="showCreateModal = true" class="btn-primary text-sm">
        + Add Domain
      </button>
    </div>

    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <div v-else-if="domains.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 text-opanel-text-muted mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
      </svg>
      <p class="text-opanel-text-muted">No domains found</p>
    </div>

    <div v-else class="card overflow-hidden p-0">
      <table class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Domain</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">IP</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Status</th>
            <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-opanel-border">
          <tr v-for="domain in domains" :key="domain.id" class="hover:bg-opanel-bg/50">
            <td class="px-6 py-4">
              <span class="text-sm font-medium text-white">{{ domain.name }}</span>
            </td>
            <td class="px-6 py-4">
              <span class="text-sm text-opanel-text-muted">{{ domain.ip_address || '-' }}</span>
            </td>
            <td class="px-6 py-4">
              <span :class="statusClass(domain.status)">{{ domain.status }}</span>
            </td>
            <td class="px-6 py-4 text-right">
              <div class="flex items-center justify-end gap-2">
                <button
                  v-if="domain.status === 'active'"
                  @click="toggleStatus(domain, 'suspended')"
                  class="text-yellow-400 hover:text-yellow-300 text-sm"
                >
                  Suspend
                </button>
                <button
                  v-if="domain.status === 'suspended'"
                  @click="toggleStatus(domain, 'active')"
                  class="text-green-400 hover:text-green-300 text-sm"
                >
                  Activate
                </button>
                <button
                  @click="confirmDelete(domain)"
                  class="text-red-400 hover:text-red-300 text-sm"
                >
                  Delete
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Create Domain</h3>
        <form @submit.prevent="createDomain" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Domain Name</label>
            <input v-model="newDomainName" type="text" class="input" placeholder="example.com" required />
          </div>
          <div v-if="error" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
            {{ error }}
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="showCreateModal = false" class="btn-ghost">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="domainToDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="domainToDelete = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-2">Delete Domain</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete <strong class="text-white">{{ domainToDelete.name }}</strong>?
          This will remove all associated files, Nginx config, and PHP-FPM pool.
        </p>
        <div class="flex justify-end gap-3">
          <button @click="domainToDelete = null" class="btn-ghost">Cancel</button>
          <button @click="deleteDomain" class="btn-danger" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import type { Domain } from '@/types'

const domains = ref<Domain[]>([])
const loading = ref(true)
const showCreateModal = ref(false)
const newDomainName = ref('')
const creating = ref(false)
const error = ref<string | null>(null)
const domainToDelete = ref<Domain | null>(null)
const deleting = ref(false)

function statusClass(status: string) {
  const base = 'badge '
  switch (status) {
    case 'active': return base + 'badge-success'
    case 'suspended': return base + 'badge-danger'
    case 'pending': return base + 'badge-warning'
    default: return base + 'badge-warning'
  }
}

async function loadDomains() {
  loading.value = true
  try {
    domains.value = await api.listDomains()
  } catch (e) {
    console.error('Failed to load domains:', e)
  } finally {
    loading.value = false
  }
}

async function createDomain() {
  creating.value = true
  error.value = null
  try {
    await api.createDomain({ name: newDomainName.value })
    showCreateModal.value = false
    newDomainName.value = ''
    await loadDomains()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to create domain'
  } finally {
    creating.value = false
  }
}

function confirmDelete(domain: Domain) {
  domainToDelete.value = domain
}

async function deleteDomain() {
  if (!domainToDelete.value) return
  deleting.value = true
  try {
    await api.deleteDomain(domainToDelete.value.id)
    domainToDelete.value = null
    await loadDomains()
  } catch (e) {
    console.error('Failed to delete domain:', e)
  } finally {
    deleting.value = false
  }
}

async function toggleStatus(domain: Domain, status: string) {
  try {
    await api.updateDomain(domain.id, { status })
    await loadDomains()
  } catch (e) {
    console.error('Failed to update domain:', e)
  }
}

onMounted(loadDomains)
</script>
