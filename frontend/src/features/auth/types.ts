export type Preferences = {
  currency: string
  language: string
  theme: 'light' | 'dark' | 'system'
  timezone: string
  notifications: { budgetAlerts: boolean; warrantyReminders: boolean; weeklySummary: boolean }
}

export type User = {
  id: string
  email: string
  emailVerified: boolean
  name: string
  avatarUrl?: string
  hasPassword: boolean
  providers: string[]
  preferences: Preferences
  createdAt: string
}

export type Session = { accessToken: string; user: User }
