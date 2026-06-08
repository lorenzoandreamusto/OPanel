<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-white">Users</h2>
      <button @click="showCreateModal = true" class="btn-primary text-sm">
        + Add User
      </button>
    </div>

    <div v-if="loading" class="text-center py-12 text-opanel-text-muted">Loading...</div>

    <div v-else class="card overflow-hidden p-0">
      <table class="w-full">
        <thead>
          <tr class="border-b border-opanel-border">
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Username</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Email</th>
            <th class="text-left text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Role</th>
            <th class="text-right text-xs font-medium text-opanel-text-muted uppercase tracking-wider px-6 py-3">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-opanel-border">
          <tr v-for="u in users" :key="u.id" class="hover:bg-opanel-bg/50">
            <td class="px-6 py-4">
              <span class="text-sm font-medium text-white">{{ u.username }}</span>
            </td>
            <td class="px-6 py-4">
              <span class="text-sm text-opanel-text-muted">{{ u.email }}</span>
            </td>
            <td class="px-6 py-4">
              <span :class="u.role === 'admin' ? 'badge bg-purple-900/50 text-purple-400' : 'badge bg-opanel-border/50 text-opanel-text-muted'">
                {{ u.role }}
              </span>
            </td>
            <td class="px-6 py-4 text-right">
              <div class="flex items-center justify-end gap-2">
                <button
                  @click="editUser(u)"
                  class="text-opanel-primary hover:text-opanel-primary-hover text-sm"
                >
                  Edit
                </button>
                <button
                  v-if="u.id !== currentUserId"
                  @click="confirmDelete(u)"
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
        <h3 class="text-lg font-semibold text-white mb-4">Create User</h3>
        <form @submit.prevent="createUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Username</label>
            <input v-model="form.username" type="text" class="input" required />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Email</label>
            <input v-model="form.email" type="email" class="input" required />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Password</label>
            <input v-model="form.password" type="password" class="input" required />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Role</label>
            <select v-model="form.role" class="input">
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div v-if="error" class="bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg px-4 py-3">
            {{ error }}
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="showCreateModal = false" class="btn-ghost">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              {{ saving ? 'Saving...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="editingUser" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="editingUser = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-4">Edit User: {{ editingUser.username }}</h3>
        <form @submit.prevent="updateUser" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Email</label>
            <input v-model="editForm.email" type="email" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">New Password (leave blank to keep)</label>
            <input v-model="editForm.password" type="password" class="input" placeholder="Unchanged" />
          </div>
          <div>
            <label class="block text-sm font-medium text-opanel-text-muted mb-1.5">Role</label>
            <select v-model="editForm.role" class="input">
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="editingUser = null" class="btn-ghost">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="userToDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="userToDelete = null">
      <div class="card w-full max-w-md mx-4">
        <h3 class="text-lg font-semibold text-white mb-2">Delete User</h3>
        <p class="text-opanel-text-muted mb-6">
          Are you sure you want to delete <strong class="text-white">{{ userToDelete.username }}</strong>?
        </p>
        <div class="flex justify-end gap-3">
          <button @click="userToDelete = null" class="btn-ghost">Cancel</button>
          <button @click="deleteUser" class="btn-danger" :disabled="deleting">
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
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'

const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.id)

const users = ref<User[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref<string | null>(null)

const showCreateModal = ref(false)
const form = ref({ username: '', email: '', password: '', role: 'user' })

const editingUser = ref<User | null>(null)
const editForm = ref({ email: '', password: '', role: 'user' })

const userToDelete = ref<User | null>(null)
const deleting = ref(false)

async function loadUsers() {
  loading.value = true
  try {
    users.value = await api.listUsers()
  } catch (e) {
    console.error('Failed to load users:', e)
  } finally {
    loading.value = false
  }
}

async function createUser() {
  saving.value = true
  error.value = null
  try {
    await api.createUser(form.value)
    showCreateModal.value = false
    form.value = { username: '', email: '', password: '', role: 'user' }
    await loadUsers()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to create user'
  } finally {
    saving.value = false
  }
}

function editUser(u: User) {
  editingUser.value = u
  editForm.value = { email: u.email, password: '', role: u.role }
}

async function updateUser() {
  if (!editingUser.value) return
  saving.value = true
  try {
    const data: Partial<User & { password?: string }> = { email: editForm.value.email, role: editForm.value.role as 'admin' | 'user' }
    if (editForm.value.password) data.password = editForm.value.password
    await api.updateUser(editingUser.value.id, data)
    editingUser.value = null
    await loadUsers()
  } catch (e) {
    console.error('Failed to update user:', e)
  } finally {
    saving.value = false
  }
}

function confirmDelete(u: User) {
  userToDelete.value = u
}

async function deleteUser() {
  if (!userToDelete.value) return
  deleting.value = true
  try {
    await api.deleteUser(userToDelete.value.id)
    userToDelete.value = null
    await loadUsers()
  } catch (e) {
    console.error('Failed to delete user:', e)
  } finally {
    deleting.value = false
  }
}

onMounted(loadUsers)
</script>
