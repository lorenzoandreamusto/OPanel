<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-white">All Databases</h2>
      <button @click="showCreateModal = true" class="btn-primary text-sm">
        + Add Database
      </button>
    </div>

    <!-- Search -->
    <div v-if="databases.length > 0">
      <input
        v-model="search"
        type="text"
        class="input w-full max-w-sm"
        placeholder="Search databases..."
      />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <!-- Empty -->
    <div v-else-if="databases.length === 0" class="card text-center py-12">
      <svg class="w-12 h-12 text-opanel-text-muted mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
      </svg>
      <p class="text-opanel-text-muted">No databases found</p>
    </div>

    <!-- Table -->
    <div v-else class="card overflow-hidden p-0">
      <table class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Name</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Created</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Users</th>
            <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-opanel-border">
          <template v-for="db in filteredDatabases" :key="db.id">
            <tr class="hover:bg-opanel-bg/50">
              <td class="px-6 py-4">
                <span class="text-sm font-medium text-white">{{ db.name }}</span>
              </td>
              <td class="px-6 py-4">
                <span class="text-sm text-opanel-text-muted">{{ formatDate(db.created_at) }}</span>
              </td>
              <td class="px-6 py-4">
                <span class="text-sm text-opanel-text-muted">{{ dbUserCounts[db.id] ?? '-' }}</span>
              </td>
              <td class="px-6 py-4 text-right space-x-3">
                <button
                  @click="openManageUsers(db)"
                  class="text-opanel-primary hover:text-opanel-primary/80 text-sm"
                >
                  Manage Users
                </button>
                <button
                  @click="confirmDelete(db)"
                  class="text-red-400 hover:text-red-300 text-sm"
                >
                  Delete
                </button>
              </td>
            </tr>
            <!-- Expanded Users Section -->
            <tr v-if="expandedDbId === db.id">
              <td colspan="4" class="px-6 py-4 bg-opanel-bg/30">
                <div class="space-y-4">
                  <div class="flex items-center justify-between">
                    <h4 class="text-sm font-semibold text-white">Database Users for "{{ db.name }}"</h4>
                    <button @click="showCreateUserModal = true" class="btn-primary text-xs py-1 px-3">
                      + Add User
                    </button>
                  </div>

                  <div v-if="loadingUsers" class="text-center py-4 text-opanel-text-muted text-sm">Loading users...</div>

                  <div v-else-if="dbUsers.length === 0" class="text-center py-4 text-opanel-text-muted text-sm">
                    No users for this database.
                  </div>

                  <table v-else class="w-full">
                    <thead>
                      <tr class="border-b border-opanel-border">
                        <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider pb-2">Username</th>
                        <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider pb-2">Host</th>
                        <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider pb-2">Privileges</th>
                        <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider pb-2">Actions</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-opanel-border/50">
                      <tr v-for="user in dbUsers" :key="user.id" class="hover:bg-opanel-bg/30">
                        <td class="py-2">
                          <span class="text-sm text-white">{{ user.username }}</span>
                        </td>
                        <td class="py-2">
                          <span class="text-sm text-opanel-text-muted">{{ user.host }}</span>
                        </td>
                        <td class="py-2">
                          <span class="text-sm text-opanel-text-muted">{{ user.privileges }}</span>
                        </td>
                        <td class="py-2 text-right space-x-3">
                          <button
                            @click="openEditUser(user)"
                            class="text-opanel-primary hover:text-opanel-primary/80 text-sm"
                          >
                            Edit
                          </button>
                          <button
                            @click="confirmDeleteUser(user)"
                            class="text-red-400 hover:text-red-300 text-sm"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- Create Database Modal -->
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

    <!-- Delete Database Confirmation -->
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

    <!-- Create Database User Modal -->
    <div v-if="showCreateUserModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeCreateUserModal">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Add Database User</h3>
        <form @submit.prevent="createDatabaseUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Username</label>
            <input v-model="newUser.username" type="text" class="input" placeholder="dbuser" required />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Password</label>
            <input v-model="newUser.password" type="password" class="input" placeholder="password" required />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Privileges</label>
            <select v-model="newUser.privileges" class="input">
              <option value="ALL PRIVILEGES">ALL PRIVILEGES</option>
              <option value="SELECT, INSERT, UPDATE, DELETE">SELECT, INSERT, UPDATE, DELETE</option>
              <option value="SELECT">SELECT</option>
              <option value="SELECT, INSERT">SELECT, INSERT</option>
              <option value="SELECT, INSERT, UPDATE">SELECT, INSERT, UPDATE</option>
              <option value="custom">Custom...</option>
            </select>
          </div>
          <div v-if="newUser.privileges === 'custom'">
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Custom Privileges</label>
            <input v-model="newUser.customPrivileges" type="text" class="input" placeholder="SELECT, CREATE TEMPORARY TABLES" />
          </div>
          <div v-if="userError" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
            {{ userError }}
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="closeCreateUserModal" class="btn-ghost">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="creatingUser">
              {{ creatingUser ? 'Creating...' : 'Create User' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Database User Modal -->
    <div v-if="userToEdit" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="userToEdit = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Edit User "{{ userToEdit.username }}"</h3>
        <form @submit.prevent="updateDatabaseUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">New Password (leave blank to keep current)</label>
            <input v-model="editUser.password" type="password" class="input" placeholder="New password" />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Privileges</label>
            <select v-model="editUser.privileges" class="input">
              <option value="ALL PRIVILEGES">ALL PRIVILEGES</option>
              <option value="SELECT, INSERT, UPDATE, DELETE">SELECT, INSERT, UPDATE, DELETE</option>
              <option value="SELECT">SELECT</option>
              <option value="SELECT, INSERT">SELECT, INSERT</option>
              <option value="SELECT, INSERT, UPDATE">SELECT, INSERT, UPDATE</option>
              <option value="custom">Custom...</option>
            </select>
          </div>
          <div v-if="editUser.privileges === 'custom'">
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Custom Privileges</label>
            <input v-model="editUser.customPrivileges" type="text" class="input" placeholder="SELECT, CREATE TEMPORARY TABLES" />
          </div>
          <div v-if="editUserError" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
            {{ editUserError }}
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="userToEdit = null" class="btn-ghost">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="updatingUser">
              {{ updatingUser ? 'Saving...' : 'Save Changes' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Database User Confirmation -->
    <div v-if="userToDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="userToDelete = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-2">Delete Database User</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete user <strong class="text-white">{{ userToDelete.username }}</strong>@<strong class="text-white">{{ userToDelete.host }}</strong>?
        </p>
        <div class="flex justify-end gap-3">
          <button @click="userToDelete = null" class="btn-ghost">Cancel</button>
          <button @click="deleteDatabaseUser" class="btn-danger" :disabled="deletingUser">
            {{ deletingUser ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Toast Container -->
    <div class="fixed top-4 right-4 z-50 space-y-2">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="[
          'px-4 py-3 rounded-lg shadow-lg text-sm font-medium transition-all',
          toast.type === 'success' ? 'bg-green-900/90 text-green-200 border border-green-700' : '',
          toast.type === 'error' ? 'bg-red-900/90 text-red-200 border border-red-700' : '',
          toast.type === 'info' ? 'bg-opanel-card text-opanel-text border border-opanel-border' : '',
        ]"
      >
        {{ toast.message }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { api } from '@/api'
import type { Database, DatabaseUser } from '@/types'
import { useToast } from '@/composables/useToast'

const { toasts, success, error: toastError } = useToast()

const databases = ref<Database[]>([])
const loading = ref(true)
const search = ref('')

// Database CRUD
const showCreateModal = ref(false)
const newDbName = ref('')
const creating = ref(false)
const error = ref<string | null>(null)
const dbToDelete = ref<Database | null>(null)
const deleting = ref(false)

// Expanded row / user management
const expandedDbId = ref<number | null>(null)
const dbUsers = ref<DatabaseUser[]>([])
const loadingUsers = ref(false)
const dbUserCounts = ref<Record<number, number>>({})

// Create user
const showCreateUserModal = ref(false)
const newUser = reactive({ username: '', password: '', privileges: 'ALL PRIVILEGES', customPrivileges: '' })
const creatingUser = ref(false)
const userError = ref<string | null>(null)

// Edit user
const userToEdit = ref<DatabaseUser | null>(null)
const editUser = reactive({ password: '', privileges: '', customPrivileges: '' })
const updatingUser = ref(false)
const editUserError = ref<string | null>(null)

// Delete user
const userToDelete = ref<DatabaseUser | null>(null)
const deletingUser = ref(false)

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString()
}

const filteredDatabases = computed(() => {
  if (!search.value) return databases.value
  const q = search.value.toLowerCase()
  return databases.value.filter((db) => db.name.toLowerCase().includes(q))
})

async function loadDatabases() {
  loading.value = true
  try {
    databases.value = await api.listDatabases()
    // Load user counts
    const counts: Record<number, number> = {}
    await Promise.all(
      databases.value.map(async (db) => {
        try {
          const users = await api.listDatabaseUsers(db.id)
          counts[db.id] = users.length
        } catch {
          counts[db.id] = 0
        }
      })
    )
    dbUserCounts.value = counts
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
    success('Database created successfully')
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
    if (expandedDbId.value !== null) {
      expandedDbId.value = null
      dbUsers.value = []
    }
    success('Database deleted successfully')
    await loadDatabases()
  } catch (e) {
    toastError(e instanceof Error ? e.message : 'Failed to delete database')
  } finally {
    deleting.value = false
  }
}

async function openManageUsers(db: Database) {
  if (expandedDbId.value === db.id) {
    expandedDbId.value = null
    dbUsers.value = []
    return
  }
  expandedDbId.value = db.id
  loadingUsers.value = true
  try {
    dbUsers.value = await api.listDatabaseUsers(db.id)
    dbUserCounts.value[db.id] = dbUsers.value.length
  } catch (e) {
    toastError('Failed to load database users')
    dbUsers.value = []
  } finally {
    loadingUsers.value = false
  }
}

function closeCreateUserModal() {
  showCreateUserModal.value = false
  newUser.username = ''
  newUser.password = ''
  newUser.privileges = 'ALL PRIVILEGES'
  newUser.customPrivileges = ''
  userError.value = null
}

async function createDatabaseUser() {
  if (expandedDbId.value === null) return
  creatingUser.value = true
  userError.value = null
  try {
    const privileges = newUser.privileges === 'custom' ? newUser.customPrivileges : newUser.privileges
    await api.createDatabaseUser(expandedDbId.value, {
      username: newUser.username,
      password: newUser.password,
      privileges,
    })
    closeCreateUserModal()
    success('Database user created successfully')
    // Reload users
    dbUsers.value = await api.listDatabaseUsers(expandedDbId.value)
    dbUserCounts.value[expandedDbId.value] = dbUsers.value.length
  } catch (e: unknown) {
    userError.value = e instanceof Error ? e.message : 'Failed to create user'
  } finally {
    creatingUser.value = false
  }
}

function openEditUser(user: DatabaseUser) {
  userToEdit.value = user
  editUser.password = ''
  editUser.privileges = user.privileges
  editUser.customPrivileges = ''
  editUserError.value = null
}

async function updateDatabaseUser() {
  if (!userToEdit.value || expandedDbId.value === null) return
  updatingUser.value = true
  editUserError.value = null
  try {
    const data: { password?: string; privileges?: string } = {}
    if (editUser.password) {
      data.password = editUser.password
    }
    const privileges = editUser.privileges === 'custom' ? editUser.customPrivileges : editUser.privileges
    data.privileges = privileges
    await api.updateDatabaseUser(expandedDbId.value, userToEdit.value.id, data)
    userToEdit.value = null
    success('Database user updated successfully')
    dbUsers.value = await api.listDatabaseUsers(expandedDbId.value)
  } catch (e: unknown) {
    editUserError.value = e instanceof Error ? e.message : 'Failed to update user'
  } finally {
    updatingUser.value = false
  }
}

function confirmDeleteUser(user: DatabaseUser) {
  userToDelete.value = user
}

async function deleteDatabaseUser() {
  if (!userToDelete.value || expandedDbId.value === null) return
  deletingUser.value = true
  try {
    await api.deleteDatabaseUser(expandedDbId.value, userToDelete.value.id)
    userToDelete.value = null
    success('Database user deleted successfully')
    dbUsers.value = await api.listDatabaseUsers(expandedDbId.value)
    dbUserCounts.value[expandedDbId.value] = dbUsers.value.length
  } catch (e) {
    toastError(e instanceof Error ? e.message : 'Failed to delete user')
  } finally {
    deletingUser.value = false
  }
}

onMounted(loadDatabases)
</script>
