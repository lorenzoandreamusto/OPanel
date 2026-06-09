<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <button @click="$router.back()" class="btn-ghost text-sm">&larr; Back</button>
        <h2 class="text-xl font-bold text-white">File Manager</h2>
      </div>
      <div class="flex items-center gap-3">
        <select v-model="selectedDomain" class="input w-48" @change="loadFiles">
          <option value="">Select domain...</option>
          <option v-for="d in domains" :key="d.id" :value="d.name">{{ d.name }}</option>
        </select>
        <button v-if="selectedDomain" @click="showUploadModal = true" class="btn-primary text-sm">Upload</button>
        <button v-if="selectedDomain" @click="showCreateDirModal = true" class="btn-primary text-sm">New Folder</button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div v-if="selectedDomain" class="flex items-center gap-2 text-sm text-opanel-text-muted">
      <button @click="navigateTo('/')" class="hover:text-opanel-primary">{{ selectedDomain }}</button>
      <template v-for="(crumb, i) in breadcrumbs" :key="i">
        <span>/</span>
        <button @click="navigateTo(crumb.path)" class="hover:text-opanel-primary">{{ crumb.name }}</button>
      </template>
    </div>

    <!-- File listing -->
    <div v-if="selectedDomain" class="card p-0">
      <div v-if="loading" class="p-8 text-center text-opanel-text-muted">Loading...</div>
      <div v-else-if="files.length === 0" class="p-8 text-center text-opanel-text-muted">Empty directory</div>
      <table v-else class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Name</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Size</th>
            <th class="text-left px-4 py-3 text-sm font-medium text-opanel-text-muted">Modified</th>
            <th class="text-right px-4 py-3 text-sm font-medium text-opanel-text-muted">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="currentPath !== '/'" class="border-b border-opanel-border hover:bg-opanel-bg/50">
            <td colspan="4" class="px-4 py-3">
              <button @click="goUp" class="text-opanel-primary hover:underline text-sm">..</button>
            </td>
          </tr>
          <tr v-for="file in files" :key="file.path" class="border-b border-opanel-border last:border-0 hover:bg-opanel-bg/50">
            <td class="px-4 py-3">
              <button v-if="file.is_dir" @click="navigateTo(file.path)" class="text-opanel-primary hover:underline text-sm">
                {{ file.name }}
              </button>
              <button v-else @click="editFile(file)" class="text-white hover:text-opanel-primary text-sm">
                {{ file.name }}
              </button>
            </td>
            <td class="px-4 py-3 text-sm text-opanel-text-muted">{{ file.is_dir ? '-' : formatSize(file.size) }}</td>
            <td class="px-4 py-3 text-sm text-opanel-text-muted">{{ file.mod_time }}</td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <a v-if="!file.is_dir" :href="downloadUrl(file.path)" class="text-opanel-text-muted hover:text-opanel-primary text-sm">Download</a>
                <button @click="confirmDelete(file)" class="text-opanel-text-muted hover:text-opanel-danger text-sm">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- No domain selected -->
    <div v-if="!selectedDomain" class="card text-center py-12 text-opanel-text-muted">
      Select a domain to manage files
    </div>

    <!-- Upload Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showUploadModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Upload File</h3>
        <input type="file" ref="fileInput" @change="handleUpload" class="input" />
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showUploadModal = false" class="btn-ghost text-sm">Cancel</button>
        </div>
      </div>
    </div>

    <!-- Create Directory Modal -->
    <div v-if="showCreateDirModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showCreateDirModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Create Directory</h3>
        <input v-model="newDirName" class="input" placeholder="Directory name" @keyup.enter="createDirectory" />
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showCreateDirModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="createDirectory" class="btn-primary text-sm">Create</button>
        </div>
      </div>
    </div>

    <!-- File Editor Modal -->
    <div v-if="showEditorModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showEditorModal = false">
      <div class="card w-full max-w-3xl max-h-[80vh] flex flex-col">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-white">{{ editingFile }}</h3>
          <span class="text-sm text-opanel-text-muted">{{ fileContent.length }} chars</span>
        </div>
        <textarea v-model="fileContent" class="input flex-1 font-mono text-sm min-h-[400px]" style="resize: vertical;"></textarea>
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showEditorModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="saveFile" class="btn-primary text-sm">Save</button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-semibold text-white mb-4">Delete {{ deletingFile?.is_dir ? 'Directory' : 'File' }}</h3>
        <p class="text-opanel-text-muted text-sm">Are you sure you want to delete <strong class="text-white">{{ deletingFile?.name }}</strong>?</p>
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showDeleteModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="deleteFile" class="btn-danger text-sm">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import type { Domain } from '@/types'

const toast = useToast()

const domains = ref<Domain[]>([])
const selectedDomain = ref('')
const currentPath = ref('/')
const files = ref<Array<{name: string; size: number; is_dir: boolean; mode: string; mod_time: string; path: string}>>([])
const loading = ref(false)

const showUploadModal = ref(false)
const showCreateDirModal = ref(false)
const showEditorModal = ref(false)
const showDeleteModal = ref(false)
const newDirName = ref('')
const editingFile = ref('')
const fileContent = ref('')
const deletingFile = ref<{name: string; path: string; is_dir: boolean} | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const breadcrumbs = computed(() => {
  if (currentPath.value === '/') return []
  const parts = currentPath.value.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function downloadUrl(path: string): string {
  return `/api/files/${selectedDomain.value}/download?path=${encodeURIComponent(path)}`
}

async function loadDomains() {
  try {
    domains.value = await api.listDomains()
  } catch {
    toast.error('Failed to load domains')
  }
}

async function loadFiles() {
  if (!selectedDomain.value) return
  loading.value = true
  try {
    const resp = await fetch(`/api/files/${selectedDomain.value}?path=${encodeURIComponent(currentPath.value)}`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    files.value = await resp.json()
  } catch {
    toast.error('Failed to load files')
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string) {
  currentPath.value = path
  loadFiles()
}

function goUp() {
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  currentPath.value = parts.length ? '/' + parts.join('/') : '/'
  loadFiles()
}

async function createDirectory() {
  if (!newDirName.value) return
  try {
    await fetch(`/api/files/${selectedDomain.value}/mkdir`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: JSON.stringify({ path: currentPath.value + '/' + newDirName.value })
    })
    toast.success('Directory created')
    showCreateDirModal.value = false
    newDirName.value = ''
    loadFiles()
  } catch {
    toast.error('Failed to create directory')
  }
}

async function handleUpload() {
  const input = fileInput.value
  if (!input?.files?.length) return
  const file = input.files[0]
  const formData = new FormData()
  formData.append('file', file)
  formData.append('path', currentPath.value + '/' + file.name)

  try {
    await fetch(`/api/files/${selectedDomain.value}/upload`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: formData
    })
    toast.success('File uploaded')
    showUploadModal.value = false
    loadFiles()
  } catch {
    toast.error('Failed to upload file')
  }
}

async function editFile(file: {name: string; path: string}) {
  try {
    const resp = await fetch(`/api/files/${selectedDomain.value}/read?path=${encodeURIComponent(file.path)}`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    const data = await resp.json()
    editingFile.value = file.path
    fileContent.value = data.content || ''
    showEditorModal.value = true
  } catch {
    toast.error('Failed to read file')
  }
}

async function saveFile() {
  try {
    await fetch(`/api/files/${selectedDomain.value}/write`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` },
      body: JSON.stringify({ path: editingFile.value, content: fileContent.value })
    })
    toast.success('File saved')
    showEditorModal.value = false
  } catch {
    toast.error('Failed to save file')
  }
}

function confirmDelete(file: {name: string; path: string; is_dir: boolean}) {
  deletingFile.value = file
  showDeleteModal.value = true
}

async function deleteFile() {
  if (!deletingFile.value) return
  try {
    await fetch(`/api/files/${selectedDomain.value}?path=${encodeURIComponent(deletingFile.value.path)}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('opanel_token')}` }
    })
    toast.success('Deleted')
    showDeleteModal.value = false
    loadFiles()
  } catch {
    toast.error('Failed to delete')
  }
}

watch(selectedDomain, () => {
  currentPath.value = '/'
  loadFiles()
})

onMounted(loadDomains)
</script>
