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
