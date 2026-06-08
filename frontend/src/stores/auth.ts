import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import { api } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('opanel_token'))
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await api.login({ username, password })
      token.value = res.token
      user.value = res.user
      localStorage.setItem('opanel_token', res.token)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchUser() {
    if (!token.value) return
    try {
      user.value = await api.getMe()
    } catch {
      logout()
    }
  }

  async function logout() {
    try {
      if (token.value) await api.logout()
    } catch {
      // ignore
    }
    token.value = null
    user.value = null
    localStorage.removeItem('opanel_token')
  }

  return { user, token, loading, error, isAuthenticated, isAdmin, login, fetchUser, logout }
})
