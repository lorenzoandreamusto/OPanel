<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-opanel-text">DNS Management</h1>
        <p class="text-opanel-text-muted mt-1">Manage DNS zones and records</p>
      </div>
      <button @click="openCreateZone" class="btn-primary">
        <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4"/></svg>
        Add Zone
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-opanel-accent"></div>
    </div>

    <!-- Empty State -->
    <div v-else-if="zones.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 mx-auto text-opanel-text-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9"/></svg>
      <h3 class="text-lg font-medium text-opanel-text mb-2">No DNS Zones</h3>
      <p class="text-opanel-text-muted mb-4">Create a DNS zone to start managing records</p>
      <button @click="openCreateZone" class="btn-primary">Create Zone</button>
    </div>

    <!-- Zone List -->
    <div v-else class="space-y-4">
      <!-- Zone Cards -->
      <div v-for="zone in zones" :key="zone.id" class="card">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <button @click="selectZone(zone)" class="text-left">
              <h3 class="text-lg font-medium text-opanel-text hover:text-opanel-accent transition-colors">{{ zone.name }}</h3>
            </button>
            <span :class="zone.enabled ? 'badge-success' : 'badge-danger'" class="badge">
              {{ zone.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>
          <div class="flex items-center space-x-2">
            <button @click="selectZone(zone)" class="btn-ghost text-sm">
              <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
              Records
            </button>
            <button @click="confirmDeleteZone(zone)" class="btn-ghost text-sm text-red-400 hover:text-red-300">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Zone Detail Panel -->
    <div v-if="selectedZone" class="card space-y-4">
      <div class="flex items-center justify-between border-b border-opanel-border pb-4">
        <div>
          <h2 class="text-xl font-bold text-opanel-text">{{ selectedZone.name }} - DNS Records</h2>
          <p class="text-opanel-text-muted text-sm mt-1">Manage DNS records for this zone</p>
        </div>
        <div class="flex items-center space-x-2">
          <button @click="openCreateRecord" class="btn-primary text-sm">
            <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4"/></svg>
            Add Record
          </button>
          <button @click="selectedZone = null" class="btn-ghost text-sm">Close</button>
        </div>
      </div>

      <!-- Records Table -->
      <div v-if="records.length === 0" class="text-center py-8">
        <p class="text-opanel-text-muted">No DNS records configured</p>
        <button @click="openCreateRecord" class="btn-primary mt-4">Add First Record</button>
      </div>

      <table v-else class="w-full">
        <thead>
          <tr class="text-left text-opanel-text-muted text-sm border-b border-opanel-border">
            <th class="pb-3 font-medium">Type</th>
            <th class="pb-3 font-medium">Name</th>
            <th class="pb-3 font-medium">Value</th>
            <th class="pb-3 font-medium">TTL</th>
            <th class="pb-3 font-medium">Priority</th>
            <th class="pb-3 font-medium">Status</th>
            <th class="pb-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="record in records" :key="record.id" class="border-b border-opanel-border hover:bg-opanel-panel/50 transition-colors">
            <td class="py-3">
              <span class="badge" :class="recordTypeClass(record.type)">{{ record.type }}</span>
            </td>
            <td class="py-3 text-opanel-text font-mono text-sm">{{ record.name }}</td>
            <td class="py-3 text-opanel-text font-mono text-sm max-w-xs truncate">{{ record.value }}</td>
            <td class="py-3 text-opanel-text-muted text-sm">{{ record.ttl }}</td>
            <td class="py-3 text-opanel-text-muted text-sm">{{ record.priority || '-' }}</td>
            <td class="py-3">
              <span :class="record.enabled ? 'badge-success' : 'badge-danger'" class="badge">
                {{ record.enabled ? 'Active' : 'Disabled' }}
              </span>
            </td>
            <td class="py-3 text-right">
              <button @click="openEditRecord(record)" class="btn-ghost text-sm mr-2">Edit</button>
              <button @click="confirmDeleteRecord(record)" class="btn-ghost text-sm text-red-400 hover:text-red-300">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Zone Modal -->
    <div v-if="showCreateZoneModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeCreateZone">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-4">Create DNS Zone</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Domain</label>
            <select v-model="newZone.domain_id" class="input w-full">
              <option value="">Select a domain...</option>
              <option v-for="domain in availableDomains" :key="domain.id" :value="domain.id">{{ domain.name }}</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end space-x-3 mt-6">
          <button @click="closeCreateZone" class="btn-ghost">Cancel</button>
          <button @click="createZone" :disabled="!newZone.domain_id" class="btn-primary">Create Zone</button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Record Modal -->
    <div v-if="showRecordModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeRecordModal">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-4">{{ editingRecord ? 'Edit Record' : 'Add Record' }}</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Type</label>
            <select v-model="recordForm.type" class="input w-full">
              <option value="A">A</option>
              <option value="AAAA">AAAA</option>
              <option value="CNAME">CNAME</option>
              <option value="MX">MX</option>
              <option value="TXT">TXT</option>
              <option value="SRV">SRV</option>
              <option value="NS">NS</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Name</label>
            <input v-model="recordForm.name" type="text" class="input w-full" placeholder="@ or subdomain">
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Value</label>
            <input v-model="recordForm.value" type="text" class="input w-full" :placeholder="recordPlaceholder">
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-opanel-text-muted mb-1">TTL</label>
              <input v-model.number="recordForm.ttl" type="number" class="input w-full" placeholder="3600">
            </div>
            <div v-if="recordForm.type === 'MX' || recordForm.type === 'SRV'">
              <label class="block text-sm font-medium text-opanel-text-muted mb-1">Priority</label>
              <input v-model.number="recordForm.priority" type="number" class="input w-full" placeholder="0">
            </div>
          </div>
        </div>
        <div class="flex justify-end space-x-3 mt-6">
          <button @click="closeRecordModal" class="btn-ghost">Cancel</button>
          <button @click="saveRecord" :disabled="!recordForm.name || !recordForm.value" class="btn-primary">
            {{ editingRecord ? 'Update' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Zone Confirmation -->
    <div v-if="showDeleteZoneModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteZoneModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-2">Delete DNS Zone</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete the DNS zone for <strong class="text-opanel-text">{{ zoneToDelete?.name }}</strong>?
          This will remove all DNS records and cannot be undone.
        </p>
        <div class="flex justify-end space-x-3">
          <button @click="showDeleteZoneModal = false" class="btn-ghost">Cancel</button>
          <button @click="deleteZone" class="btn-danger">Delete Zone</button>
        </div>
      </div>
    </div>

    <!-- Delete Record Confirmation -->
    <div v-if="showDeleteRecordModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteRecordModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-2">Delete DNS Record</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete this <strong class="text-opanel-text">{{ recordToDelete?.type }}</strong> record
          for <strong class="text-opanel-text">{{ recordToDelete?.name }}</strong>?
        </p>
        <div class="flex justify-end space-x-3">
          <button @click="showDeleteRecordModal = false" class="btn-ghost">Cancel</button>
          <button @click="deleteRecord" class="btn-danger">Delete Record</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { useToast } from '../composables/useToast'
import type { DNSZone, DNSRecord, Domain } from '../types'

const { success, error: toastError } = useToast()

const loading = ref(true)
const zones = ref<DNSZone[]>([])
const selectedZone = ref<DNSZone | null>(null)
const records = ref<DNSRecord[]>([])
const allDomains = ref<Domain[]>([])

const showCreateZoneModal = ref(false)
const showRecordModal = ref(false)
const showDeleteZoneModal = ref(false)
const showDeleteRecordModal = ref(false)

const newZone = ref({ domain_id: 0 })
const editingRecord = ref<DNSRecord | null>(null)
const recordForm = ref({ type: 'A', name: '', value: '', ttl: 3600, priority: 0 })
const zoneToDelete = ref<DNSZone | null>(null)
const recordToDelete = ref<DNSRecord | null>(null)

const availableDomains = computed(() => {
  const zoneDomainIds = zones.value.map(z => z.domain_id)
  return allDomains.value.filter(d => !zoneDomainIds.includes(d.id))
})

const recordPlaceholder = computed(() => {
  switch (recordForm.value.type) {
    case 'A': return '192.168.1.1'
    case 'AAAA': return '::1'
    case 'CNAME': return 'example.com'
    case 'MX': return 'mail.example.com'
    case 'TXT': return 'v=spf1 include:_spf.google.com ~all'
    case 'SRV': return '_sip._tcp.example.com'
    case 'NS': return 'ns1.example.com'
    default: return ''
  }
})

function recordTypeClass(type: string) {
  switch (type) {
    case 'A': return 'badge-success'
    case 'AAAA': return 'badge-success'
    case 'CNAME': return 'badge-warning'
    case 'MX': return 'badge-info'
    case 'TXT': return 'badge-info'
    case 'SRV': return 'badge-warning'
    case 'NS': return 'badge-danger'
    default: return ''
  }
}

async function loadZones() {
  loading.value = true
  try {
    zones.value = await api.listDNSZones()
    allDomains.value = await api.listDomains()
  } catch (e: any) {
    toastError(e.message || 'Failed to load DNS zones')
  } finally {
    loading.value = false
  }
}

async function selectZone(zone: DNSZone) {
  selectedZone.value = zone
  try {
    records.value = await api.listDNSRecords(zone.id)
  } catch (e: any) {
    toastError(e.message || 'Failed to load records')
    records.value = []
  }
}

function openCreateZone() {
  newZone.value = { domain_id: 0 }
  showCreateZoneModal.value = true
}

function closeCreateZone() {
  showCreateZoneModal.value = false
  newZone.value = { domain_id: 0 }
}

async function createZone() {
  if (!newZone.value.domain_id) return
  try {
    await api.createDNSZone({ domain_id: newZone.value.domain_id })
    success('DNS zone created')
    closeCreateZone()
    await loadZones()
  } catch (e: any) {
    toastError(e.message || 'Failed to create zone')
  }
}

function confirmDeleteZone(zone: DNSZone) {
  zoneToDelete.value = zone
  showDeleteZoneModal.value = true
}

async function deleteZone() {
  if (!zoneToDelete.value) return
  try {
    await api.deleteDNSZone(zoneToDelete.value.id)
    success('DNS zone deleted')
    showDeleteZoneModal.value = false
    if (selectedZone.value?.id === zoneToDelete.value.id) {
      selectedZone.value = null
      records.value = []
    }
    await loadZones()
  } catch (e: any) {
    toastError(e.message || 'Failed to delete zone')
  }
}

function openCreateRecord() {
  editingRecord.value = null
  recordForm.value = { type: 'A', name: '@', value: '', ttl: 3600, priority: 0 }
  showRecordModal.value = true
}

function openEditRecord(record: DNSRecord) {
  editingRecord.value = record
  recordForm.value = {
    type: record.type,
    name: record.name,
    value: record.value,
    ttl: record.ttl,
    priority: record.priority
  }
  showRecordModal.value = true
}

function closeRecordModal() {
  showRecordModal.value = false
  editingRecord.value = null
}

async function saveRecord() {
  if (!selectedZone.value) return
  try {
    if (editingRecord.value) {
      await api.updateDNSRecord(editingRecord.value.id, {
        type: recordForm.value.type,
        name: recordForm.value.name,
        value: recordForm.value.value,
        ttl: recordForm.value.ttl,
        priority: recordForm.value.priority
      })
      success('Record updated')
    } else {
      await api.createDNSRecord(selectedZone.value.id, {
        type: recordForm.value.type,
        name: recordForm.value.name,
        value: recordForm.value.value,
        ttl: recordForm.value.ttl,
        priority: recordForm.value.priority
      })
      success('Record created')
    }
    closeRecordModal()
    await selectZone(selectedZone.value)
  } catch (e: any) {
    toastError(e.message || 'Failed to save record')
  }
}

function confirmDeleteRecord(record: DNSRecord) {
  recordToDelete.value = record
  showDeleteRecordModal.value = true
}

async function deleteRecord() {
  if (!recordToDelete.value) return
  try {
    await api.deleteDNSRecord(recordToDelete.value.id)
    success('Record deleted')
    showDeleteRecordModal.value = false
    if (selectedZone.value) {
      await selectZone(selectedZone.value)
    }
  } catch (e: any) {
    toastError(e.message || 'Failed to delete record')
  }
}

onMounted(loadZones)
</script>
