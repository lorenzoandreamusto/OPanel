<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-white">All Domains</h2>
      <button @click="openCreateModal" class="btn-primary text-sm">
        + Add Domain
      </button>
    </div>

    <!-- Search -->
    <div v-if="!loading && domains.length > 0">
      <input
        v-model="searchQuery"
        type="text"
        class="input max-w-sm"
        placeholder="Search domains..."
      />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <!-- Empty state -->
    <div v-else-if="domains.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 text-opanel-text-muted mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
      </svg>
      <p class="text-opanel-text-muted">No domains found</p>
    </div>

    <!-- No results -->
    <div v-else-if="filteredDomains.length === 0" class="card text-center py-12">
      <p class="text-opanel-text-muted">No domains match "{{ searchQuery }}"</p>
    </div>

    <!-- Table -->
    <div v-else class="card overflow-hidden p-0">
      <table class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Domain</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Type</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">PHP</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Status</th>
            <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-opanel-border">
          <tr v-for="domain in filteredDomains" :key="domain.id" class="hover:bg-opanel-bg/50">
            <td class="px-6 py-4">
              <router-link
                :to="{ name: 'domain-detail', params: { id: domain.id } }"
                class="text-sm font-medium text-opanel-primary hover:text-opanel-primary-hover transition-colors"
              >
                {{ domain.name }}
              </router-link>
            </td>
            <td class="px-6 py-4">
              <span :class="domain.hosting_type === 'php' ? 'badge badge-success' : 'badge bg-blue-900/50 text-blue-400'">
                {{ domain.hosting_type === 'php' ? 'PHP' : 'Static' }}
              </span>
            </td>
            <td class="px-6 py-4">
              <span class="text-sm text-opanel-text-muted">
                {{ domain.hosting_type === 'php' ? `PHP ${domain.php_version}` : '-' }}
              </span>
            </td>
            <td class="px-6 py-4">
              <span :class="statusClass(domain.status)">{{ domain.status }}</span>
            </td>
            <td class="px-6 py-4 text-right">
              <div class="flex items-center justify-end gap-2">
                <a
                  :href="`http://${domain.name}`"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-opanel-text-muted hover:text-opanel-primary transition-colors"
                  title="Open site"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
                  </svg>
                </a>
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
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeCreateModal">
      <div class="card w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-semibold text-white mb-4">Create Domain</h3>

        <!-- Tabs -->
        <div class="flex border-b border-opanel-border mb-6">
          <button
            v-for="(tab, idx) in tabs"
            :key="tab.key"
            @click="activeTab = idx"
            class="px-4 py-2.5 text-sm font-medium transition-colors relative"
            :class="activeTab === idx ? 'text-opanel-primary' : 'text-opanel-text-muted hover:text-opanel-text'"
          >
            {{ tab.label }}
            <div
              v-if="activeTab === idx"
              class="absolute bottom-0 left-0 right-0 h-0.5 bg-opanel-primary"
            />
          </button>
        </div>

        <!-- Error -->
        <div v-if="error" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3 mb-4">
          {{ error }}
        </div>

        <!-- Tab 1: Basic Settings -->
        <div v-if="activeTab === 0" class="space-y-5">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Domain Name</label>
            <input
              v-model="form.name"
              type="text"
              class="input"
              placeholder="example.com"
              @input="autoFillDocumentRoot"
            />
            <p v-if="submitted && !form.name.trim()" class="text-red-400 text-xs mt-1">Domain name is required</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-2">Hosting Type</label>
            <div class="flex gap-3">
              <label
                class="flex-1 flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="form.hosting_type === 'php' ? 'border-opanel-primary bg-opanel-primary/10' : 'border-opanel-border hover:border-opanel-primary/50'"
              >
                <input v-model="form.hosting_type" type="radio" value="php" class="sr-only" />
                <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center" :class="form.hosting_type === 'php' ? 'border-opanel-primary' : 'border-opanel-text-muted'">
                  <div v-if="form.hosting_type === 'php'" class="w-2.5 h-2.5 rounded-full bg-opanel-primary" />
                </div>
                <div>
                  <div class="text-sm font-medium text-white">PHP Website</div>
                  <div class="text-xs text-opanel-text-muted">Full PHP support</div>
                </div>
              </label>
              <label
                class="flex-1 flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="form.hosting_type === 'static' ? 'border-opanel-primary bg-opanel-primary/10' : 'border-opanel-border hover:border-opanel-primary/50'"
              >
                <input v-model="form.hosting_type" type="radio" value="static" class="sr-only" />
                <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center" :class="form.hosting_type === 'static' ? 'border-opanel-primary' : 'border-opanel-text-muted'">
                  <div v-if="form.hosting_type === 'static'" class="w-2.5 h-2.5 rounded-full bg-opanel-primary" />
                </div>
                <div>
                  <div class="text-sm font-medium text-white">Static Website</div>
                  <div class="text-xs text-opanel-text-muted">HTML/CSS/JS only</div>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- Tab 2: PHP Configuration -->
        <div v-if="activeTab === 1" class="space-y-5">
          <div v-if="form.hosting_type === 'php'">
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">PHP Version</label>
            <select v-model="form.php_version" class="input">
              <option value="8.1">PHP 8.1</option>
              <option value="8.2">PHP 8.2</option>
              <option value="8.3">PHP 8.3</option>
              <option value="8.4">PHP 8.4</option>
            </select>
            <p class="text-xs text-opanel-text-muted mt-1.5">PHP-FPM pool will be created for this version.</p>
          </div>
          <div v-else class="text-center py-8">
            <svg class="w-10 h-10 text-opanel-text-muted mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
            </svg>
            <p class="text-opanel-text-muted text-sm">PHP configuration is only available for PHP websites.</p>
            <p class="text-opanel-text-muted text-xs mt-1">Static sites serve HTML/CSS/JS directly.</p>
          </div>
        </div>

        <!-- Tab 3: Advanced Settings -->
        <div v-if="activeTab === 2" class="space-y-5">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Document Root</label>
            <input v-model="form.document_root" type="text" class="input" placeholder="/var/www/vhosts/example.com/httpdocs" />
            <p class="text-xs text-opanel-text-muted mt-1.5">Auto-filled based on domain name. Files will be served from this directory.</p>
          </div>

          <div class="flex items-center justify-between p-3 rounded-lg border border-opanel-border">
            <div>
              <div class="text-sm font-medium text-white">SSL/TLS Certificate</div>
              <div class="text-xs text-opanel-text-muted">Enable Let's Encrypt for this domain</div>
            </div>
            <button
              @click="form.ssl_enabled = !form.ssl_enabled"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="form.ssl_enabled ? 'bg-opanel-primary' : 'bg-opanel-border'"
            >
              <span
                class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                :class="form.ssl_enabled ? 'translate-x-6' : 'translate-x-1'"
              />
            </button>
          </div>

          <div class="flex items-center justify-between p-3 rounded-lg border border-opanel-border">
            <div>
              <div class="text-sm font-medium text-white">Auto-create Database</div>
              <div class="text-xs text-opanel-text-muted">Create a MariaDB database with the same name</div>
            </div>
            <button
              @click="form.auto_db = !form.auto_db"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="form.auto_db ? 'bg-opanel-primary' : 'bg-opanel-border'"
            >
              <span
                class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                :class="form.auto_db ? 'translate-x-6' : 'translate-x-1'"
              />
            </button>
          </div>

          <div class="flex items-center justify-between p-3 rounded-lg border border-opanel-border">
            <div>
              <div class="text-sm font-medium text-white">Enable Mail</div>
              <div class="text-xs text-opanel-text-muted">Create mail domain with MX, SPF, DKIM DNS records</div>
            </div>
            <button
              @click="form.mail_enabled = !form.mail_enabled"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="form.mail_enabled ? 'bg-opanel-primary' : 'bg-opanel-border'"
            >
              <span
                class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                :class="form.mail_enabled ? 'translate-x-6' : 'translate-x-1'"
              />
            </button>
          </div>
        </div>

        <!-- Tab 4: Review -->
        <div v-if="activeTab === 3" class="space-y-4">
          <h4 class="text-sm font-medium text-opanel-text-muted uppercase tracking-wider">Summary</h4>
          <div class="space-y-3">
            <div class="flex justify-between py-2 border-b border-opanel-border">
              <span class="text-sm text-opanel-text-muted">Domain</span>
              <span class="text-sm text-white font-medium">{{ form.name || '-' }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-opanel-border">
              <span class="text-sm text-opanel-text-muted">Hosting Type</span>
              <span :class="form.hosting_type === 'php' ? 'badge badge-success' : 'badge bg-blue-900/50 text-blue-400'">
                {{ form.hosting_type === 'php' ? 'PHP Website' : 'Static Website' }}
              </span>
            </div>
            <div v-if="form.hosting_type === 'php'" class="flex justify-between py-2 border-b border-opanel-border">
              <span class="text-sm text-opanel-text-muted">PHP Version</span>
              <span class="text-sm text-white">{{ form.php_version }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-opanel-border">
              <span class="text-sm text-opanel-text-muted">Document Root</span>
              <span class="text-sm text-white font-mono text-xs">{{ form.document_root }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-opanel-border">
              <span class="text-sm text-opanel-text-muted">SSL/TLS</span>
              <span :class="form.ssl_enabled ? 'text-green-400 text-sm' : 'text-opanel-text-muted text-sm'">
                {{ form.ssl_enabled ? 'Enabled' : 'Disabled' }}
              </span>
            </div>
            <div class="flex justify-between py-2">
              <span class="text-sm text-opanel-text-muted">Auto Database</span>
              <span :class="form.auto_db ? 'text-green-400 text-sm' : 'text-opanel-text-muted text-sm'">
                {{ form.auto_db ? 'Yes' : 'No' }}
              </span>
            </div>
            <div class="flex justify-between py-2">
              <span class="text-sm text-opanel-text-muted">Mail</span>
              <span :class="form.mail_enabled ? 'text-green-400 text-sm' : 'text-opanel-text-muted text-sm'">
                {{ form.mail_enabled ? 'Enabled (MX + SPF + DKIM)' : 'Disabled' }}
              </span>
            </div>
          </div>
        </div>

        <!-- Navigation -->
        <div class="flex justify-between mt-6 pt-4 border-t border-opanel-border">
          <button
            v-if="activeTab > 0"
            @click="activeTab--"
            class="btn-ghost text-sm"
          >
            Back
          </button>
          <div v-else />
          <div class="flex gap-3">
            <button type="button" @click="closeCreateModal" class="btn-ghost text-sm">Cancel</button>
            <button
              v-if="activeTab < tabs.length - 1"
              @click="nextTab"
              class="btn-primary text-sm"
            >
              Next
            </button>
            <button
              v-else
              @click="createDomain"
              class="btn-primary text-sm"
              :disabled="creating"
            >
              {{ creating ? 'Creating...' : 'Create Domain' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="domainToDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="domainToDelete = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-2">Delete Domain</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete <strong class="text-white">{{ domainToDelete.name }}</strong>?
          This will remove all associated files, Nginx config, PHP-FPM pool, DNS zone, and mail domain.
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
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api'
import type { Domain, CreateDomainRequest } from '@/types'

const domains = ref<Domain[]>([])
const loading = ref(true)
const searchQuery = ref('')

const showCreateModal = ref(false)
const activeTab = ref(0)
const creating = ref(false)
const error = ref<string | null>(null)
const submitted = ref(false)

const domainToDelete = ref<Domain | null>(null)
const deleting = ref(false)

const tabs = [
  { key: 'basic', label: 'Basic Settings' },
  { key: 'php', label: 'PHP Config' },
  { key: 'advanced', label: 'Advanced' },
  { key: 'review', label: 'Review' },
]

interface DomainForm {
  name: string
  hosting_type: 'php' | 'static'
  php_version: string
  document_root: string
  ssl_enabled: boolean
  auto_db: boolean
  mail_enabled: boolean
}

const defaultForm = (): DomainForm => ({
  name: '',
  hosting_type: 'php',
  php_version: '8.4',
  document_root: '/var/www/vhosts//httpdocs',
  ssl_enabled: false,
  auto_db: false,
  mail_enabled: false,
})

const form = ref<DomainForm>(defaultForm())

const filteredDomains = computed(() => {
  if (!searchQuery.value.trim()) return domains.value
  const q = searchQuery.value.toLowerCase()
  return domains.value.filter(d => d.name.toLowerCase().includes(q))
})

function autoFillDocumentRoot() {
  const name = form.value.name.trim()
  if (name) {
    form.value.document_root = `/var/www/vhosts/${name}/httpdocs`
  } else {
    form.value.document_root = '/var/www/vhosts//httpdocs'
  }
}

function openCreateModal() {
  form.value = defaultForm()
  activeTab.value = 0
  submitted.value = false
  error.value = null
  showCreateModal.value = true
}

function closeCreateModal() {
  showCreateModal.value = false
  error.value = null
}

function nextTab() {
  submitted.value = true
  if (activeTab.value === 0 && !form.value.name.trim()) {
    error.value = 'Domain name is required'
    return
  }
  error.value = null
  submitted.value = false
  activeTab.value++
}

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
  if (!form.value.name.trim()) {
    error.value = 'Domain name is required'
    return
  }

  creating.value = true
  error.value = null

  const payload: CreateDomainRequest = {
    name: form.value.name.trim(),
    php_version: form.value.hosting_type === 'php' ? form.value.php_version : undefined,
    hosting_type: form.value.hosting_type,
    ssl_enabled: form.value.ssl_enabled,
    auto_db: form.value.auto_db,
    mail_enabled: form.value.mail_enabled,
  }

  try {
    await api.createDomain(payload)
    showCreateModal.value = false
    form.value = defaultForm()
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
