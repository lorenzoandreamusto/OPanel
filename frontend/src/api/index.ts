import type { LoginRequest, LoginResponse, User, Domain, CreateDomainRequest, Database, DatabaseUser, Backup, SystemStats, FileInfo, DNSZone, DNSRecord, CreateDNSZoneRequest, CreateDNSRecordRequest, UpdateDNSRecordRequest, MailDomain, MailAccount, CreateMailDomainRequest, CreateMailAccountRequest, UpdateMailAccountRequest, MailAutoconfig, DKIMRecord } from '@/types'

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

  async createDomain(data: CreateDomainRequest): Promise<Domain> {
    return this.request<Domain>('/domains', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDomain(id: number, data: { status?: string; php_version?: string; hosting_type?: string; ssl_enabled?: boolean; auto_db?: boolean }): Promise<Domain> {
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

  // File Manager
  async listFiles(domain: string, path: string = '/httpdocs'): Promise<FileInfo[]> {
    return this.request<FileInfo[]>(`/files/${domain}?path=${encodeURIComponent(path)}`)
  }

  async readFile(domain: string, path: string): Promise<{content: string; path: string}> {
    return this.request<{content: string; path: string}>(`/files/${domain}/read?path=${encodeURIComponent(path)}`)
  }

  async writeFile(domain: string, path: string, content: string): Promise<void> {
    return this.request<void>(`/files/${domain}/write`, {
      method: 'POST',
      body: JSON.stringify({ path, content }),
    })
  }

  async createDirectory(domain: string, path: string): Promise<void> {
    return this.request<void>(`/files/${domain}/mkdir`, {
      method: 'POST',
      body: JSON.stringify({ path }),
    })
  }

  async deleteFile(domain: string, path: string): Promise<void> {
    return this.request<void>(`/files/${domain}?path=${encodeURIComponent(path)}`, {
      method: 'DELETE',
    })
  }

  // Monitoring
  async getMonitoringStats(): Promise<SystemStats> {
    return this.request<SystemStats>('/monitoring/stats')
  }

  // Backups
  async listBackups(domainId?: number): Promise<Backup[]> {
    const query = domainId ? `?domain_id=${domainId}` : ''
    return this.request<Backup[]>(`/backups${query}`)
  }

  async createBackup(data: {domain_id: number; domain_name: string; name?: string}): Promise<{id: number; name: string}> {
    return this.request<{id: number; name: string}>('/backups', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteBackup(id: number): Promise<void> {
    return this.request<void>(`/backups/${id}`, { method: 'DELETE' })
  }

  async restoreBackup(id: number, domainName: string): Promise<void> {
    return this.request<void>(`/backups/${id}/restore`, {
      method: 'POST',
      body: JSON.stringify({ domain_name: domainName }),
    })
  }

  // WordPress
  async installWordPress(data: {domain_id: number; domain_name: string; site_name: string; admin_user: string; admin_pass: string; admin_email: string}): Promise<{id: number}> {
    return this.request<{id: number}>('/wordpress/install', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // DNS Zones
  async listDNSZones(): Promise<DNSZone[]> {
    return this.request<DNSZone[]>('/dns/zones')
  }

  async getDNSZone(id: number): Promise<DNSZone> {
    return this.request<DNSZone>(`/dns/zones/${id}`)
  }

  async createDNSZone(data: CreateDNSZoneRequest): Promise<DNSZone> {
    return this.request<DNSZone>('/dns/zones', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteDNSZone(id: number): Promise<void> {
    return this.request<void>(`/dns/zones/${id}`, { method: 'DELETE' })
  }

  // DNS Records
  async listDNSRecords(zoneId: number): Promise<DNSRecord[]> {
    return this.request<DNSRecord[]>(`/dns/zones/${zoneId}/records`)
  }

  async createDNSRecord(zoneId: number, data: CreateDNSRecordRequest): Promise<DNSRecord> {
    return this.request<DNSRecord>(`/dns/zones/${zoneId}/records`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDNSRecord(recordId: number, data: UpdateDNSRecordRequest): Promise<DNSRecord> {
    return this.request<DNSRecord>(`/dns/records/${recordId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteDNSRecord(recordId: number): Promise<void> {
    return this.request<void>(`/dns/records/${recordId}`, { method: 'DELETE' })
  }

  // Mail Domains
  async listMailDomains(): Promise<MailDomain[]> {
    return this.request<MailDomain[]>('/mail/domains')
  }

  async getMailDomain(id: number): Promise<MailDomain> {
    return this.request<MailDomain>(`/mail/domains/${id}`)
  }

  async createMailDomain(data: CreateMailDomainRequest): Promise<MailDomain> {
    return this.request<MailDomain>('/mail/domains', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteMailDomain(id: number): Promise<void> {
    return this.request<void>(`/mail/domains/${id}`, { method: 'DELETE' })
  }

  // Mail Accounts
  async listMailAccounts(domainId: number): Promise<MailAccount[]> {
    return this.request<MailAccount[]>(`/mail/domains/${domainId}/accounts`)
  }

  async createMailAccount(domainId: number, data: CreateMailAccountRequest): Promise<MailAccount> {
    return this.request<MailAccount>(`/mail/domains/${domainId}/accounts`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateMailAccount(accountId: number, data: UpdateMailAccountRequest): Promise<MailAccount> {
    return this.request<MailAccount>(`/mail/accounts/${accountId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteMailAccount(accountId: number): Promise<void> {
    return this.request<void>(`/mail/accounts/${accountId}`, { method: 'DELETE' })
  }

  // Mail Autoconfig
  async getMailAutoconfig(domain: string): Promise<MailAutoconfig> {
    return this.request<MailAutoconfig>(`/mail/autoconfig/${domain}`)
  }

  // DKIM
  async getDKIMRecord(domain: string): Promise<DKIMRecord> {
    return this.request<DKIMRecord>(`/mail/dkim/${domain}`)
  }
}

export const api = new ApiClient()
