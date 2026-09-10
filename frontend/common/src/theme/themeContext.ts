import { createContext } from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ThemeCtx = { theme: Theme; setTheme: (t: Theme) => void }
export const ThemeContext = createContext<ThemeCtx>({ theme: 'system', setTheme: () => {} })
