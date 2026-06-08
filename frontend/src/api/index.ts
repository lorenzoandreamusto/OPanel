import type { LoginRequest, LoginResponse, User, Domain, Database, DatabaseUser } from '@/types'

const BASE_URL = '/api'

class ApiClient {
  private getToken(): string | null {
    return localStorage.getItem('opanel_token')
  }

  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const token = this.getToken()
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    }
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(`${BASE_URL}${path}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Auth
  async login(data: LoginRequest): Promise<LoginResponse> {
    return this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async logout(): Promise<void> {
    return this.request<void>('/auth/logout', { method: 'POST' })
  }

  async getMe(): Promise<User> {
    return this.request<User>('/auth/me')
  }

  // Users
  async listUsers(): Promise<User[]> {
    return this.request<User[]>('/users')
  }

  async getUser(id: number): Promise<User> {
    return this.request<User>(`/users/${id}`)
  }

  async createUser(data: { username: string; email: string; password: string; role: string }): Promise<User> {
    return this.request<User>('/users', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateUser(id: number, data: Partial<User & { password?: string }>): Promise<User> {
    return this.request<User>(`/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteUser(id: number): Promise<void> {
    return this.request<void>(`/users/${id}`, { method: 'DELETE' })
  }

  // Domains
  async listDomains(): Promise<Domain[]> {
    return this.request<Domain[]>('/domains')
  }

  async getDomain(id: number): Promise<Domain> {
    return this.request<Domain>(`/domains/${id}`)
  }

  async createDomain(data: { name: string }): Promise<Domain> {
    return this.request<Domain>('/domains', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDomain(id: number, data: { status?: string }): Promise<Domain> {
    return this.request<Domain>(`/domains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteDomain(id: number): Promise<void> {
    return this.request<void>(`/domains/${id}`, { method: 'DELETE' })
  }

  // Databases
  async listDatabases(): Promise<Database[]> {
    return this.request<Database[]>('/databases')
  }

  async getDatabase(id: number): Promise<Database> {
    return this.request<Database>(`/databases/${id}`)
  }

  async createDatabase(data: { name: string }): Promise<Database> {
    return this.request<Database>('/databases', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteDatabase(id: number): Promise<void> {
    return this.request<void>(`/databases/${id}`, { method: 'DELETE' })
  }

  async listDatabaseUsers(dbId: number): Promise<DatabaseUser[]> {
    return this.request<DatabaseUser[]>(`/databases/${dbId}/users`)
  }

  async createDatabaseUser(dbId: number, data: { username: string; password: string; privileges: string }): Promise<DatabaseUser> {
    return this.request<DatabaseUser>(`/databases/${dbId}/users`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDatabaseUser(dbId: number, userId: number, data: { password?: string; privileges?: string }): Promise<DatabaseUser> {
    return this.request<DatabaseUser>(`/databases/${dbId}/users/${userId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteDatabaseUser(dbId: number, userId: number): Promise<void> {
    return this.request<void>(`/databases/${dbId}/users/${userId}`, { method: 'DELETE' })
  }
}

export const api = new ApiClient()
