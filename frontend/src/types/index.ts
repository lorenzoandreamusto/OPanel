export interface User {
  id: number
  username: string
  email: string
  role: 'admin' | 'user'
  created_at: string
  updated_at: string
}

export interface Domain {
  id: number
  name: string
  ip_address: string
  status: 'active' | 'suspended' | 'pending'
  document_root: string
  owner_id: number
  php_version: string
  hosting_type: 'static' | 'php'
  ssl_enabled: boolean
  auto_db: boolean
  created_at: string
  updated_at: string
}

export interface CreateDomainRequest {
  name: string
  php_version?: string
  hosting_type?: 'static' | 'php'
  ssl_enabled?: boolean
  auto_db?: boolean
  mail_enabled?: boolean
}

export interface Database {
  id: number
  name: string
  owner_id: number
  created_at: string
}

export interface DatabaseUser {
  id: number
  username: string
  host: string
  database_id: number
  privileges: string
  created_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface ApiError {
  error: string
}

export interface Backup {
  id: number
  name: string
  domain_id: number
  size: number
  status: string
  created_at: string
}

export interface SystemStats {
  cpu: number
  memory: {
    total: number
    used: number
    free: number
    available: number
    percent: number
  }
  disk: {
    total: number
    used: number
    free: number
    percent: number
  }
  load_avg: {
    load1: number
    load5: number
    load15: number
  }
  timestamp: number
}

export interface FileInfo {
  name: string
  size: number
  is_dir: boolean
  mode: string
  mod_time: string
  path: string
}

export interface WordPressInstall {
  id: number
  domain_id: number
  site_name: string
  admin_user: string
  status: string
  created_at: string
}

export interface DNSZone {
  id: number
  domain_id: number
  name: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface DNSRecord {
  id: number
  zone_id: number
  type: string
  name: string
  value: string
  ttl: number
  priority: number
  enabled: boolean
}

export interface CreateDNSZoneRequest {
  domain_id: number
}

export interface CreateDNSRecordRequest {
  type: string
  name: string
  value: string
  ttl?: number
  priority?: number
}

export interface UpdateDNSRecordRequest {
  type?: string
  name?: string
  value?: string
  ttl?: number
  priority?: number
  enabled?: boolean
}

export interface MailDomain {
  id: number
  domain_id: number
  name: string
  enabled: boolean
  dkim_enabled: boolean
  dkim_selector: string
  created_at: string
  updated_at: string
}

export interface MailAccount {
  id: number
  domain_id: number
  username: string
  password?: string
  quota: number
  used: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateMailDomainRequest {
  domain_id: number
}

export interface CreateMailAccountRequest {
  username: string
  password: string
  quota?: number
}

export interface UpdateMailAccountRequest {
  password?: string
  quota?: number
  enabled?: boolean
}

export interface MailAutoconfig {
  domain: string
  imap_host: string
  imap_port: number
  imap_ssl: number
  imap_starttls: number
  smtp_host: string
  smtp_port: number
  smtp_ssl: number
  smtp_starttls: number
  smtp_auth: boolean
  username: string
}

export interface DKIMRecord {
  domain: string
  selector: string
  record: string
}
