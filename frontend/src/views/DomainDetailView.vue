<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <router-link to="/domains" class="text-opanel-text-muted hover:text-opanel-text transition-colors">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
          </svg>
        </router-link>
        <h2 class="text-lg font-semibold text-white">{{ domain?.name ?? 'Domain' }}</h2>
        <span v-if="domain" :class="statusClass(domain.status)">{{ domain.status }}</span>
      </div>
      <div class="flex items-center gap-3">
        <button
          v-if="domain"
          @click="toggleStatus"
          class="text-sm font-medium py-2 px-4 rounded-lg transition-colors"
          :class="domain.status === 'active' ? 'text-yellow-400 hover:bg-yellow-900/30' : 'text-green-400 hover:bg-green-900/30'"
        >
          {{ domain.status === 'active' ? 'Suspend' : 'Activate' }}
        </button>
        <a
          v-if="domain"
          :href="`http://${domain.name}`"
          target="_blank"
          rel="noopener noreferrer"
          class="btn-primary text-sm inline-flex items-center gap-2"
        >
          Open Site
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
          </svg>
        </a>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <!-- Not found -->
    <div v-else-if="!domain" class="card text-center py-12">
      <p class="text-opanel-text-muted">Domain not found.</p>
    </div>

    <template v-else>
      <!-- Info Cards -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="card">
          <div class="text-xs text-opanel-text-muted uppercase tracking-wider mb-1">Domain</div>
          <div class="text-sm text-white font-medium truncate">{{ domain.name }}</div>
        </div>
        <div class="card">
          <div class="text-xs text-opanel-text-muted uppercase tracking-wider mb-1">IP Address</div>
          <div class="text-sm text-white font-medium">{{ domain.ip_address || '-' }}</div>
        </div>
        <div class="card">
          <div class="text-xs text-opanel-text-muted uppercase tracking-wider mb-1">PHP Version</div>
          <div class="text-sm text-white font-medium">
            {{ domain.hosting_type === 'php' ? `PHP ${domain.php_version}` : 'N/A' }}
          </div>
        </div>
        <div class="card">
          <div class="text-xs text-opanel-text-muted uppercase tracking-wider mb-1">Hosting Type</div>
          <div class="text-sm font-medium" :class="domain.hosting_type === 'php' ? 'text-green-400' : 'text-blue-400'">
            {{ domain.hosting_type === 'php' ? 'PHP Website' : 'Static Website' }}
          </div>
        </div>
      </div>

      <!-- Management Grid -->
      <div>
        <h3 class="text-sm font-medium text-opanel-text-muted uppercase tracking-wider mb-4">Management</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <button
            v-for="card in managementCards"
            :key="card.key"
            @click="openCard(card.key)"
            class="card text-left hover:border-opanel-primary/50 transition-all group cursor-pointer"
          >
            <div class="flex items-start gap-3">
              <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0" :class="card.bgClass">
                <svg class="w-5 h-5" :class="card.iconClass" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" :d="card.icon" />
                </svg>
              </div>
              <div class="min-w-0">
                <div class="text-sm font-medium text-white group-hover:text-opanel-primary transition-colors">{{ card.title }}</div>
                <div class="text-xs text-opanel-text-muted mt-0.5">{{ card.description }}</div>
              </div>
            </div>
          </button>
        </div>
      </div>
    </template>

    <!-- Hosting Settings Modal -->
    <div v-if="showHostingModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showHostingModal = false">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Hosting Settings</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-2">Hosting Type</label>
            <div class="flex gap-3">
              <label class="flex-1 flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="editForm.hosting_type === 'php' ? 'border-opanel-primary bg-opanel-primary/10' : 'border-opanel-border hover:border-opanel-primary/50'">
                <input v-model="editForm.hosting_type" type="radio" value="php" class="sr-only" />
                <div class="w-4 h-4 rounded-full border-2 flex items-center justify-center" :class="editForm.hosting_type === 'php' ? 'border-opanel-primary' : 'border-opanel-text-muted'">
                  <div v-if="editForm.hosting_type === 'php'" class="w-2 h-2 rounded-full bg-opanel-primary" />
                </div>
                <span class="text-sm text-white">PHP</span>
              </label>
              <label class="flex-1 flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="editForm.hosting_type === 'static' ? 'border-opanel-primary bg-opanel-primary/10' : 'border-opanel-border hover:border-opanel-primary/50'">
                <input v-model="editForm.hosting_type" type="radio" value="static" class="sr-only" />
                <div class="w-4 h-4 rounded-full border-2 flex items-center justify-center" :class="editForm.hosting_type === 'static' ? 'border-opanel-primary' : 'border-opanel-text-muted'">
                  <div v-if="editForm.hosting_type === 'static'" class="w-2 h-2 rounded-full bg-opanel-primary" />
                </div>
                <span class="text-sm text-white">Static</span>
              </label>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Document Root</label>
            <code class="text-sm text-white bg-opanel-bg rounded px-3 py-2 block font-mono">{{ domain?.document_root }}</code>
          </div>
        </div>
        <div class="flex justify-end gap-3 mt-6">
          <button @click="showHostingModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="saveHostingSettings" class="btn-primary text-sm" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </div>
    </div>

    <!-- PHP Configuration Modal -->
    <div v-if="showPHPModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showPHPModal = false">
      <div class="card w-full max-w-lg mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">PHP Configuration</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">PHP Version</label>
            <select v-model="editForm.php_version" class="input">
              <option value="8.1">PHP 8.1</option>
              <option value="8.2">PHP 8.2</option>
              <option value="8.3">PHP 8.3</option>
              <option value="8.4">PHP 8.4</option>
            </select>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">memory_limit</label>
              <input v-model="phpSettings.memory_limit" type="text" class="input" placeholder="128M" />
            </div>
            <div>
              <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">upload_max_filesize</label>
              <input v-model="phpSettings.upload_max_filesize" type="text" class="input" placeholder="128M" />
            </div>
            <div>
              <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">post_max_size</label>
              <input v-model="phpSettings.post_max_size" type="text" class="input" placeholder="128M" />
            </div>
            <div>
              <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">max_execution_time</label>
              <input v-model="phpSettings.max_execution_time" type="text" class="input" placeholder="300" />
            </div>
          </div>
          <p class="text-xs text-opanel-text-muted">PHP settings are configured per-domain via PHP-FPM pool config.</p>
        </div>
        <div class="flex justify-end gap-3 mt-6">
          <button @click="showPHPModal = false" class="btn-ghost text-sm">Cancel</button>
          <button @click="savePHPSettings" class="btn-primary text-sm" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Database Modal -->
    <div v-if="showDBModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDBModal = false">
      <div class="card w-full max-w-lg mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Database</h3>
        <div v-if="associatedDB" class="space-y-4">
          <div class="flex items-center justify-between p-3 bg-opanel-bg rounded-lg">
            <div>
              <div class="text-sm font-medium text-white">{{ associatedDB.name }}</div>
              <div class="text-xs text-opanel-text-muted">Created {{ formatDate(associatedDB.created_at) }}</div>
            </div>
            <span class="badge badge-success">Connected</span>
          </div>
          <div v-if="dbUsers.length > 0">
            <h4 class="text-sm font-medium text-opanel-text-muted mb-2">Database Users</h4>
            <div class="space-y-2">
              <div v-for="user in dbUsers" :key="user.id" class="flex items-center justify-between p-2 bg-opanel-bg rounded-lg">
                <div>
                  <span class="text-sm text-white">{{ user.username }}@{{ user.host }}</span>
                  <span class="text-xs text-opanel-text-muted ml-2">{{ user.privileges }}</span>
                </div>
              </div>
            </div>
          </div>
          <p v-else class="text-sm text-opanel-text-muted">No database users configured.</p>
        </div>
        <div v-else class="text-center py-8">
          <svg class="w-10 h-10 text-opanel-text-muted mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
          </svg>
          <p class="text-opanel-text-muted text-sm">No database associated with this domain.</p>
          <p class="text-opanel-text-muted text-xs mt-1">Enable "Auto-create Database" when creating the domain.</p>
        </div>
        <div class="flex justify-end mt-6">
          <button @click="showDBModal = false" class="btn-ghost text-sm">Close</button>
        </div>
      </div>
    </div>

    <!-- Placeholder Modal -->
    <div v-if="showPlaceholderModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showPlaceholderModal = false">
      <div class="card w-full max-w-md mx-4 text-center">
        <svg class="w-12 h-12 text-opanel-primary mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17 17.25 21A2.652 2.652 0 0 0 21 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 1 1-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 0 0 4.486-6.336l-3.276 3.277a3.004 3.004 0 0 1-2.25-2.25l3.276-3.276a4.5 4.5 0 0 0-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085" />
        </svg>
        <h3 class="text-lg font-semibold text-white mb-2">{{ placeholderTitle }}</h3>
        <p class="text-opanel-text-muted text-sm mb-6">This feature will be available in a future sprint.</p>
        <button @click="showPlaceholderModal = false" class="btn-primary text-sm">Got it</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import type { Domain, Database, DatabaseUser } from '@/types'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const domain = ref<Domain | null>(null)
const loading = ref(true)
const saving = ref(false)

const showHostingModal = ref(false)
const showPHPModal = ref(false)
const showDBModal = ref(false)
const showPlaceholderModal = ref(false)
const placeholderTitle = ref('')

const associatedDB = ref<Database | null>(null)
const dbUsers = ref<DatabaseUser[]>([])

const editForm = reactive({
  hosting_type: 'php' as 'php' | 'static',
  php_version: '8.4',
})

const phpSettings = reactive({
  memory_limit: '128M',
  upload_max_filesize: '128M',
  post_max_size: '128M',
  max_execution_time: '300',
})

const managementCards = [
  { key: 'hosting', title: 'Hosting Settings', description: 'Type, document root', icon: 'M5.25 14.25h13.5m-13.5 0a3 3 0 0 1-3-3m3 3a3 3 0 1 0 0 6h13.5a3 3 0 1 0 0-6m-16.5-3a3 3 0 0 1 3-3h13.5a3 3 0 0 1 3 3m-19.5 0a4.5 4.5 0 0 1 .9-2.7L5.737 5.1a3.375 3.375 0 0 1 2.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 0 1 .9 2.7m0 0a3 3 0 0 1-3 3m0 3h.008v.008h-.008v-.008Zm0-6h.008v.008h-.008v-.008Zm-3 6h.008v.008h-.008v-.008Zm0-6h.008v.008h-.008v-.008Z', bgClass: 'bg-opanel-primary/10', iconClass: 'text-opanel-primary' },
  { key: 'php', title: 'PHP Configuration', description: 'Version, limits, options', icon: 'M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5', bgClass: 'bg-purple-900/30', iconClass: 'text-purple-400' },
  { key: 'dns', title: 'DNS Settings', description: 'DNS records management', icon: 'M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418', bgClass: 'bg-yellow-900/30', iconClass: 'text-yellow-400' },
  { key: 'ssl', title: 'SSL/TLS Certificates', description: 'Manage certificates', icon: 'M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z', bgClass: 'bg-green-900/30', iconClass: 'text-green-400' },
  { key: 'database', title: 'Database', description: 'Associated databases', icon: 'M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125', bgClass: 'bg-emerald-900/30', iconClass: 'text-emerald-400' },
  { key: 'ftp', title: 'FTP Accounts', description: 'Manage FTP access', icon: 'M13.5 21v-7.5a.75.75 0 0 1 .75-.75h3a.75.75 0 0 1 .75.75V21m-4.5 0H2.36m11.14 0H18m0 0h3.64m-1.39 0V9.349M3.75 21V9.349m0 0a3.001 3.001 0 0 0 3.75-.615A2.993 2.993 0 0 0 9.75 9.75c.896 0 1.7-.393 2.25-1.016a2.993 2.993 0 0 0 2.25 1.016c.896 0 1.7-.393 2.25-1.015a3.001 3.001 0 0 0 3.75.614m-16.5 0a3.004 3.004 0 0 1-.621-4.72l1.189-1.19A1.5 1.5 0 0 1 5.378 3h13.243a1.5 1.5 0 0 1 1.06.44l1.19 1.189a3 3 0 0 1-.621 4.72M6.75 18h3.75a.75.75 0 0 0 .75-.75V13.5a.75.75 0 0 0-.75-.75H6.75a.75.75 0 0 0-.75.75v3.75c0 .414.336.75.75.75Z', bgClass: 'bg-orange-900/30', iconClass: 'text-orange-400' },
  { key: 'email', title: 'Email Accounts', description: 'Manage email boxes', icon: 'M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75', bgClass: 'bg-cyan-900/30', iconClass: 'text-cyan-400' },
  { key: 'files', title: 'File Manager', description: 'Browse website files', icon: 'M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z', bgClass: 'bg-indigo-900/30', iconClass: 'text-indigo-400' },
  { key: 'logs-access', title: 'Access Logs', description: 'View visitor logs', icon: 'M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z', bgClass: 'bg-slate-900/30', iconClass: 'text-slate-400' },
  { key: 'logs-error', title: 'Error Logs', description: 'View error logs', icon: 'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z', bgClass: 'bg-red-900/30', iconClass: 'text-red-400' },
  { key: 'backup', title: 'Backup & Restore', description: 'Create and manage backups', icon: 'M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5', bgClass: 'bg-violet-900/30', iconClass: 'text-violet-400' },
  { key: 'cron', title: 'Scheduled Tasks', description: 'Manage cron jobs', icon: 'M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z', bgClass: 'bg-pink-900/30', iconClass: 'text-pink-400' },
]

function statusClass(status: string) {
  const base = 'badge '
  switch (status) {
    case 'active': return base + 'badge-success'
    case 'suspended': return base + 'badge-danger'
    case 'pending': return base + 'badge-warning'
    default: return base + 'badge-warning'
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString()
}

function openCard(key: string) {
  switch (key) {
    case 'hosting':
      if (domain.value) {
        editForm.hosting_type = domain.value.hosting_type
        showHostingModal.value = true
      }
      break
    case 'php':
      if (domain.value) {
        editForm.php_version = domain.value.php_version
        showPHPModal.value = true
      }
      break
    case 'database':
      loadDatabaseInfo()
      showDBModal.value = true
      break
    case 'files':
      if (domain.value) {
        router.push({ name: 'file-manager', params: { domain: domain.value.name } })
      }
      break
    default:
      placeholderTitle.value = managementCards.find(c => c.key === key)?.title ?? key
      showPlaceholderModal.value = true
  }
}

async function toggleStatus() {
  if (!domain.value) return
  const newStatus = domain.value.status === 'active' ? 'suspended' : 'active'
  try {
    domain.value = await api.updateDomain(domain.value.id, { status: newStatus })
    toast.success(`Domain ${newStatus === 'active' ? 'activated' : 'suspended'}`)
  } catch (e) {
    toast.error('Failed to update domain status')
  }
}

async function saveHostingSettings() {
  if (!domain.value) return
  saving.value = true
  try {
    domain.value = await api.updateDomain(domain.value.id, { hosting_type: editForm.hosting_type })
    showHostingModal.value = false
    toast.success('Hosting settings saved')
  } catch (e) {
    toast.error('Failed to save hosting settings')
  } finally {
    saving.value = false
  }
}

async function savePHPSettings() {
  if (!domain.value) return
  saving.value = true
  try {
    domain.value = await api.updateDomain(domain.value.id, { php_version: editForm.php_version })
    showPHPModal.value = false
    toast.success('PHP configuration saved')
  } catch (e) {
    toast.error('Failed to save PHP settings')
  } finally {
    saving.value = false
  }
}

async function loadDatabaseInfo() {
  if (!domain.value) return
  try {
    const dbs = await api.listDatabases()
    const expectedDBName = domain.value.name.replace(/\./g, '_')
    associatedDB.value = dbs.find(d => d.name === expectedDBName) ?? null
    if (associatedDB.value) {
      dbUsers.value = await api.listDatabaseUsers(associatedDB.value.id)
    } else {
      dbUsers.value = []
    }
  } catch (e) {
    associatedDB.value = null
    dbUsers.value = []
  }
}

onMounted(async () => {
  try {
    const id = Number(route.params.id)
    domain.value = await api.getDomain(id)
  } catch (e) {
    console.error('Failed to load domain:', e)
  } finally {
    loading.value = false
  }
})
</script>
