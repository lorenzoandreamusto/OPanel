<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-white">Backups</h2>
      <button @click="showCreateModal = true" class="btn-primary text-sm">Create Backup</button>
    </div>

    <!-- Domain filter -->
    <div class="flex items-center gap-4">
      <select v-model="selectedDomain" class="input w-48" @change="loadBackups">
        <option value="">All domains</option>
        <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.name }}</option>
      </select>
    </div>

    <!-- Backups table -->
    <div class="card p-0">
      <div v-if="loading" class="p-8 text-center text-opanel-text-muted">Loading...</div>
      <div v-else-if="backups.length === 0" class="p-8 text-center text-opanel-text-muted">No backups found</div>
      <table v-else class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Name</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Domain</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Size</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Status</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Created</th>
            <th class="text-right px-4 py-3 text-sm font-medium text-opanel-text-muted">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="backup in backups" :key="backup.id" class="border-b border-opanel-border last:border-0 hover:bg-opanel-bg/50">
            <td class="px-4 py-3 text-sm text-white">{{ backup.name }}</td>
            <td class="px-4 py-3 text-sm text-opanel-text-muted">{{ getDomainName(backup.domain_id) }}</td>
            <td class="px-4 py-3 text-sm text-opanel-text-muted">{{ formatSize(backup.size) }}</td>
            <td class="px-4 py-3">
              <span :class="backup.status === 'completed' ? 'badge-success' : 'badge-warning'" class="badge text-xs">
                {{ backup.status }}
              </span>
            </td>
            <td class="px-4 py-3 text-sm text-opanel-text-muted">{{ formatDate(backup.created_at) }}</td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <button @click="downloadBackup(backup)" class="text-opanel-text-muted hover:text-opanel-primary text-sm">Download</button>
                <button @click="confirmRestore(backup)" class="text-opanel-text-muted hover:text-opanel-warning text-sm">Restore</button>
                <button @click="confirmDelete(backup)" class="text-opanel-text-muted hover:text-opanel-danger text-sm">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Backup Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Create Backup</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-opanel-text-muted mb-1">Domain</label>
            <select v-model="newBackup.domain_id" class="input">
              <option value="">Select domain...</option>
              <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-opanel-text-muted mb-1">Name (optional)</label>
            <input v-model="newBackup.name" class="input" placeholder="Auto-generated if empty" />
          </div>
        </div>
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showCreateModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="createBackup" class="btn-primary text-sm" :disabled="creating">
            {{ creating ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Delete Backup</h3>
        <p class="text-opanel-text-muted text-sm">Are you sure you want to delete <strong class="text-white">{{ deletingBackup?.name }}</strong>?</p>
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showDeleteModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="deleteBackup" class="btn-danger text-sm">Delete</button>
        </div>
      </div>
    </div>

    <!-- Restore Confirmation -->
    <div v-if="showRestoreModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showRestoreModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Restore Backup</h3>
        <p class="text-opanel-text-muted text-sm">This will overwrite current files for <strong class="text-white">{{ getDomainName(restoringBackup?.domain_id || 0) }}</strong>. Continue?</p>
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showRestoreModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="restoreBackup" class="btn-primary text-sm">Restore</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import type { Domain } from '@/types'

const toast = useToast()

interface Backup {
  id: number
  name: string
  domain_id: number
  size: number
  status: string
  created_at: string
}

const domains = ref<Domain[]>([])
const backups = ref<Backup[]>([])
const selectedDomain = ref('')
const loading = ref(false)
const creating = ref(false)

const showCreateModal = ref(false)
const showDeleteModal = ref(false)
const showRestoreModal = ref(false)
const newBackup = ref({ domain_id: '', name: '' })
const deletingBackup = ref<Backup | null>(null)
const restoringBackup = ref<Backup | null>(null)

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}

function getDomainName(id: number): string {
  const d = domains.value.find(d => d.id === id)
  return d ? d.name : 'Unknown'
}

async function loadDomains() {
  try {
    domains.value = await api.listDomains()
  } catch {}
}

async function loadBackups() {
  loading.value = true
  try {
    const query = selectedDomain.value ? `?domain_id=${selectedDomain.value}` : ''
    const resp = await fetch(`/api/backups${query}`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    backups.value = await resp.json()
  } catch {
    toast.error('Failed to load backups')
  } finally {
    loading.value = false
  }
}

async function createBackup() {
  if (!newBackup.value.domain_id) {
    toast.error('Please select a domain')
    return
  }
  creating.value = true
  try {
    const domain = domains.value.find(d => d.id === Number(newBackup.value.domain_id))
    await fetch('/api/backups', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: JSON.stringify({
        domain_id: Number(newBackup.value.domain_id),
        domain_name: domain?.name || '',
        name: newBackup.value.name
      })
    })
    toast.success('Backup created')
    showCreateModal.value = false
    newBackup.value = { domain_id: '', name: '' }
    loadBackups()
  } catch {
    toast.error('Failed to create backup')
  } finally {
    creating.value = false
  }
}

function confirmDelete(backup: Backup) {
  deletingBackup.value = backup
  showDeleteModal.value = true
}

async function deleteBackup() {
  if (!deletingBackup.value) return
  try {
    await fetch(`/api/backups/${deletingBackup.value.id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    toast.success('Backup deleted')
    showDeleteModal.value = false
    loadBackups()
  } catch {
    toast.error('Failed to delete backup')
  }
}

async function downloadBackup(backup: Backup) {
  try {
    const resp = await fetch(`/api/backups/${backup.id}/download`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    if (!resp.ok) throw new Error('Download failed')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${backup.name}.tar.gz`
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.error('Failed to download backup')
  }
}

function confirmRestore(backup: Backup) {
  restoringBackup.value = backup
  showRestoreModal.value = true
}

async function restoreBackup() {
  if (!restoringBackup.value) return
  const domain = domains.value.find(d => d.id === restoringBackup.value!.domain_id)
  try {
    await fetch(`/api/backups/${restoringBackup.value.id}/restore`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: JSON.stringify({ domain_name: domain?.name || '' })
    })
    toast.success('Backup restored')
    showRestoreModal.value = false
  } catch {
    toast.error('Failed to restore backup')
  }
}

onMounted(() => {
  loadDomains()
  loadBackups()
})
</script>
