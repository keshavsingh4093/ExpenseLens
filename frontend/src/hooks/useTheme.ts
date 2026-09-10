import { useContext } from 'react'
import { ThemeContext } from '@/app/themeContext'

export const useTheme = () => useContext(ThemeContext)
