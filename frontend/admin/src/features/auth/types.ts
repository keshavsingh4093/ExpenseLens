export type Admin = {
  id: string
  email: string
  name: string
  role: 'superadmin' | 'admin'
  lastLoginAt?: string
  createdAt: string
}

export type AdminSession = { accessToken: string; user: Admin }
