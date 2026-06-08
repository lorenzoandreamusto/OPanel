<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-white">All Databases</h2>
      <button @click="showCreateModal = true" class="btn-primary text-sm">
        + Add Database
      </button>
    </div>

    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <div v-else-if="databases.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 text-opanel-text-muted mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
      </svg>
      <p class="text-opanel-text-muted">No databases found</p>
    </div>

    <div v-else class="card overflow-hidden p-0">
      <table class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Name</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Created</th>
            <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-opanel-border">
          <tr v-for="db in databases" :key="db.id" class="hover:bg-opanel-bg/50">
            <td class="px-6 py-4">
              <span class="text-sm font-medium text-white">{{ db.name }}</span>
            </td>
            <td class="px-6 py-4">
              <span class="text-sm text-opanel-text-muted">{{ formatDate(db.created_at) }}</span>
            </td>
            <td class="px-6 py-4 text-right">
              <button
                @click="confirmDelete(db)"
                class="text-red-400 hover:text-red-300 text-sm"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Create Database</h3>
        <form @submit.prevent="createDatabase" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Database Name</label>
            <input v-model="newDbName" type="text" class="input" placeholder="mydb" required />
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
    <div v-if="dbToDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="dbToDelete = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-2">Delete Database</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete <strong class="text-white">{{ dbToDelete.name }}</strong>?
          This will permanently remove the database and all its data from MariaDB.
        </p>
        <div class="flex justify-end gap-3">
          <button @click="dbToDelete = null" class="btn-ghost">Cancel</button>
          <button @click="deleteDatabase" class="btn-danger" :disabled="deleting">
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
import type { Database } from '@/types'

const databases = ref<Database[]>([])
const loading = ref(true)
const showCreateModal = ref(false)
const newDbName = ref('')
const creating = ref(false)
const error = ref<string | null>(null)
const dbToDelete = ref<Database | null>(null)
const deleting = ref(false)

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString()
}

async function loadDatabases() {
  loading.value = true
  try {
    databases.value = await api.listDatabases()
  } catch (e) {
    console.error('Failed to load databases:', e)
  } finally {
    loading.value = false
  }
}

async function createDatabase() {
  creating.value = true
  error.value = null
  try {
    await api.createDatabase({ name: newDbName.value })
    showCreateModal.value = false
    newDbName.value = ''
    await loadDatabases()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to create database'
  } finally {
    creating.value = false
  }
}

function confirmDelete(db: Database) {
  dbToDelete.value = db
}

async function deleteDatabase() {
  if (!dbToDelete.value) return
  deleting.value = true
  try {
    await api.deleteDatabase(dbToDelete.value.id)
    dbToDelete.value = null
    await loadDatabases()
  } catch (e) {
    console.error('Failed to delete database:', e)
  } finally {
    deleting.value = false
  }
}

onMounted(loadDatabases)
</script>
