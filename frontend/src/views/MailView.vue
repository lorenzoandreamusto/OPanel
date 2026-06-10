<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-opanel-text">Mail Management</h1>
        <p class="text-opanel-text-muted mt-1">Manage email domains and accounts</p>
      </div>
      <button @click="openCreateDomain" class="btn-primary">
        <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4"/></svg>
        Enable Mail
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-opanel-accent"></div>
    </div>

    <!-- Empty State -->
    <div v-else-if="mailDomains.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 mx-auto text-opanel-text-muted mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
      <h3 class="text-lg font-medium text-opanel-text mb-2">No Mail Domains</h3>
      <p class="text-opanel-text-muted mb-4">Enable mail for a domain to start managing email accounts</p>
      <button @click="openCreateDomain" class="btn-primary">Enable Mail</button>
    </div>

    <!-- Mail Domain List -->
    <div v-else class="space-y-3">
      <div v-for="md in mailDomains" :key="md.id" class="card">
        <!-- Domain Header -->
        <div class="flex items-center justify-between cursor-pointer" @click="toggleDomain(md)">
          <div class="flex items-center space-x-3">
            <svg class="w-5 h-5 text-opanel-text-muted transition-transform" :class="{ 'rotate-90': expandedDomain === md.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5l7 7-7 7"/></svg>
            <h3 class="text-lg font-medium text-opanel-text">{{ md.name }}</h3>
            <span :class="md.enabled ? 'badge-success' : 'badge-danger'" class="badge">
              {{ md.enabled ? 'Enabled' : 'Disabled' }}
            </span>
            <span v-if="md.dkim_enabled" class="badge badge-info">DKIM</span>
          </div>
          <div class="flex items-center space-x-2">
            <button @click.stop="showDKIM(md)" class="btn-ghost text-sm">
              <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
              DKIM
            </button>
            <button @click.stop="confirmDeleteDomain(md)" class="btn-ghost text-sm text-red-400 hover:text-red-300">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
            </button>
          </div>
        </div>

        <!-- Expanded: Mail Accounts -->
        <div v-if="expandedDomain === md.id" class="mt-4 border-t border-opanel-border pt-4">
          <div class="flex items-center justify-between mb-3">
            <h4 class="text-sm font-medium text-opanel-text-muted">Email Accounts</h4>
            <button @click="openCreateAccount(md)" class="btn-primary text-sm">
              <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4"/></svg>
              Add Account
            </button>
          </div>

          <div v-if="loadingAccounts" class="text-center py-4">
            <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-opanel-accent mx-auto"></div>
          </div>

          <div v-else-if="domainAccounts.length === 0" class="text-center py-6">
            <p class="text-opanel-text-muted text-sm">No email accounts yet</p>
          </div>

          <table v-else class="w-full text-sm">
            <thead>
              <tr class="text-left text-opanel-text-muted border-b border-opanel-border">
                <th class="pb-2 font-medium">Email</th>
                <th class="pb-2 font-medium">Quota</th>
                <th class="pb-2 font-medium">Used</th>
                <th class="pb-2 font-medium">Status</th>
                <th class="pb-2 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="account in domainAccounts" :key="account.id" class="border-b border-opanel-border/50 hover:bg-opanel-panel/30">
                <td class="py-2 text-opanel-text font-mono">{{ account.username }}@{{ md.name }}</td>
                <td class="py-2 text-opanel-text-muted">{{ formatBytes(account.quota) }}</td>
                <td class="py-2 text-opanel-text-muted">{{ formatBytes(account.used) }}</td>
                <td class="py-2">
                  <span :class="account.enabled ? 'badge-success' : 'badge-danger'" class="badge">
                    {{ account.enabled ? 'Active' : 'Disabled' }}
                  </span>
                </td>
                <td class="py-2 text-right">
                  <button @click="openEditAccount(account)" class="btn-ghost text-xs mr-2">Edit</button>
                  <button @click="confirmDeleteAccount(account)" class="btn-ghost text-xs text-red-400 hover:text-red-300">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Create Mail Domain Modal -->
    <div v-if="showCreateDomainModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeCreateDomain">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-4">Enable Mail for Domain</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Domain</label>
            <select v-model="newMailDomain.domain_id" class="input w-full">
              <option value="">Select a domain...</option>
              <option v-for="domain in availableDomains" :key="domain.id" :value="domain.id">{{ domain.name }}</option>
            </select>
          </div>
          <p class="text-xs text-opanel-text-muted">This will enable mail services, generate DKIM keys, and configure Postfix + Dovecot for this domain.</p>
        </div>
        <div class="flex justify-end space-x-3 mt-6">
          <button @click="closeCreateDomain" class="btn-ghost">Cancel</button>
          <button @click="createMailDomain" :disabled="!newMailDomain.domain_id" class="btn-primary">Enable Mail</button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Account Modal -->
    <div v-if="showAccountModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeAccountModal">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-4">{{ editingAccount ? 'Edit Account' : 'Create Email Account' }}</h3>
        <div class="space-y-4">
          <div v-if="!editingAccount">
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Username</label>
            <div class="flex items-center">
              <input v-model="accountForm.username" type="text" class="input flex-1" placeholder="username">
              <span class="text-opanel-text-muted ml-2">@{{ selectedMailDomain?.name }}</span>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">{{ editingAccount ? 'New Password (leave empty to keep)' : 'Password' }}</label>
            <input v-model="accountForm.password" type="password" class="input w-full" placeholder="••••••••">
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">Quota (MB)</label>
            <input v-model.number="accountForm.quotaMB" type="number" class="input w-full" placeholder="1024">
          </div>
        </div>
        <div class="flex justify-end space-x-3 mt-6">
          <button @click="closeAccountModal" class="btn-ghost">Cancel</button>
          <button @click="saveAccount" :disabled="!accountForm.username || (!editingAccount && !accountForm.password)" class="btn-primary">
            {{ editingAccount ? 'Update' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- DKIM Info Modal -->
    <div v-if="showDKIMModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDKIMModal = false">
      <div class="card w-full max-w-lg">
        <h3 class="text-lg font-bold text-opanel-text mb-4">DKIM Record for {{ dkimDomain }}</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">DNS TXT Record Name</label>
            <div class="input w-full bg-opanel-bg font-mono text-sm">default._domainkey.{{ dkimDomain }}</div>
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1">DNS TXT Record Value</label>
            <textarea :value="dkimRecord" readonly class="input w-full font-mono text-xs h-24 bg-opanel-bg"></textarea>
          </div>
          <p class="text-xs text-opanel-text-muted">Add this TXT record to your DNS to enable DKIM signing for outgoing mail.</p>
        </div>
        <div class="flex justify-end mt-6">
          <button @click="showDKIMModal = false" class="btn-ghost">Close</button>
        </div>
      </div>
    </div>

    <!-- Delete Domain Confirmation -->
    <div v-if="showDeleteDomainModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteDomainModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-2">Disable Mail Domain</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to disable mail for <strong class="text-opanel-text">{{ domainToDelete?.name }}</strong>?
          All email accounts for this domain will be deleted.
        </p>
        <div class="flex justify-end space-x-3">
          <button @click="showDeleteDomainModal = false" class="btn-ghost">Cancel</button>
          <button @click="deleteMailDomain" class="btn-danger">Disable Mail</button>
        </div>
      </div>
    </div>

    <!-- Delete Account Confirmation -->
    <div v-if="showDeleteAccountModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteAccountModal = false">
      <div class="card w-full max-w-md">
        <h3 class="text-lg font-bold text-opanel-text mb-2">Delete Email Account</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete <strong class="text-opanel-text">{{ accountToDelete?.username }}@{{ selectedMailDomain?.name }}</strong>?
          All emails for this account will be permanently deleted.
        </p>
        <div class="flex justify-end space-x-3">
          <button @click="showDeleteAccountModal = false" class="btn-ghost">Cancel</button>
          <button @click="deleteAccount" class="btn-danger">Delete Account</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { useToast } from '../composables/useToast'
import type { MailDomain, MailAccount, Domain } from '../types'

const { success, error: toastError } = useToast()

const loading = ref(true)
const loadingAccounts = ref(false)
const mailDomains = ref<MailDomain[]>([])
const allDomains = ref<Domain[]>([])
const expandedDomain = ref<number | null>(null)
const domainAccounts = ref<MailAccount[]>([])

const showCreateDomainModal = ref(false)
const showAccountModal = ref(false)
const showDeleteDomainModal = ref(false)
const showDeleteAccountModal = ref(false)
const showDKIMModal = ref(false)

const newMailDomain = ref({ domain_id: 0 })
const editingAccount = ref<MailAccount | null>(null)
const selectedMailDomain = ref<MailDomain | null>(null)
const accountForm = ref({ username: '', password: '', quotaMB: 1024 })
const domainToDelete = ref<MailDomain | null>(null)
const accountToDelete = ref<MailAccount | null>(null)
const dkimDomain = ref('')
const dkimRecord = ref('')

const availableDomains = computed(() => {
  const mailDomainIds = mailDomains.value.map(md => md.domain_id)
  return allDomains.value.filter(d => !mailDomainIds.includes(d.id))
})

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

async function loadMailDomains() {
  loading.value = true
  try {
    mailDomains.value = await api.listMailDomains()
    allDomains.value = await api.listDomains()
  } catch (e: any) {
    toastError(e.message || 'Failed to load mail domains')
  } finally {
    loading.value = false
  }
}

async function toggleDomain(md: MailDomain) {
  if (expandedDomain.value === md.id) {
    expandedDomain.value = null
    domainAccounts.value = []
    selectedMailDomain.value = null
    return
  }

  expandedDomain.value = md.id
  selectedMailDomain.value = md
  loadingAccounts.value = true
  try {
    domainAccounts.value = await api.listMailAccounts(md.id)
  } catch (e: any) {
    toastError(e.message || 'Failed to load accounts')
    domainAccounts.value = []
  } finally {
    loadingAccounts.value = false
  }
}

function openCreateDomain() {
  newMailDomain.value = { domain_id: 0 }
  showCreateDomainModal.value = true
}

function closeCreateDomain() {
  showCreateDomainModal.value = false
}

async function createMailDomain() {
  if (!newMailDomain.value.domain_id) return
  try {
    await api.createMailDomain({ domain_id: newMailDomain.value.domain_id })
    success('Mail domain enabled')
    closeCreateDomain()
    await loadMailDomains()
  } catch (e: any) {
    toastError(e.message || 'Failed to enable mail domain')
  }
}

function confirmDeleteDomain(md: MailDomain) {
  domainToDelete.value = md
  showDeleteDomainModal.value = true
}

async function deleteMailDomain() {
  if (!domainToDelete.value) return
  try {
    await api.deleteMailDomain(domainToDelete.value.id)
    success('Mail domain disabled')
    showDeleteDomainModal.value = false
    if (expandedDomain.value === domainToDelete.value.id) {
      expandedDomain.value = null
      domainAccounts.value = []
      selectedMailDomain.value = null
    }
    await loadMailDomains()
  } catch (e: any) {
    toastError(e.message || 'Failed to disable mail domain')
  }
}

function openCreateAccount(md: MailDomain) {
  editingAccount.value = null
  selectedMailDomain.value = md
  accountForm.value = { username: '', password: '', quotaMB: 1024 }
  showAccountModal.value = true
}

function openEditAccount(account: MailAccount) {
  editingAccount.value = account
  accountForm.value = { username: account.username, password: '', quotaMB: Math.round(account.quota / (1024 * 1024)) }
  showAccountModal.value = true
}

function closeAccountModal() {
  showAccountModal.value = false
  editingAccount.value = null
}

async function saveAccount() {
  if (!selectedMailDomain.value) return
  try {
    if (editingAccount.value) {
      const updateData: any = {}
      if (accountForm.value.password) updateData.password = accountForm.value.password
      updateData.quota = accountForm.value.quotaMB * 1024 * 1024
      await api.updateMailAccount(editingAccount.value.id, updateData)
      success('Account updated')
    } else {
      await api.createMailAccount(selectedMailDomain.value.id, {
        username: accountForm.value.username,
        password: accountForm.value.password,
        quota: accountForm.value.quotaMB * 1024 * 1024
      })
      success('Account created')
    }
    closeAccountModal()
    if (expandedDomain.value) {
      await toggleDomain(selectedMailDomain.value)
      await toggleDomain(selectedMailDomain.value)
    }
  } catch (e: any) {
    toastError(e.message || 'Failed to save account')
  }
}

function confirmDeleteAccount(account: MailAccount) {
  accountToDelete.value = account
  showDeleteAccountModal.value = true
}

async function deleteAccount() {
  if (!accountToDelete.value) return
  try {
    await api.deleteMailAccount(accountToDelete.value.id)
    success('Account deleted')
    showDeleteAccountModal.value = false
    if (expandedDomain.value && selectedMailDomain.value) {
      domainAccounts.value = await api.listMailAccounts(selectedMailDomain.value.id)
    }
  } catch (e: any) {
    toastError(e.message || 'Failed to delete account')
  }
}

async function showDKIM(md: MailDomain) {
  dkimDomain.value = md.name
  try {
    const result = await api.getDKIMRecord(md.name)
    dkimRecord.value = result.record
  } catch (e: any) {
    dkimRecord.value = 'DKIM key not generated'
  }
  showDKIMModal.value = true
}

onMounted(loadMailDomains)
</script>
