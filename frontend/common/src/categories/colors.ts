import type { Category } from './types'

/**
 * Validated categorical palette (8 slots, light/dark steps) assigned by entity.
 * System categories map by slug; "others" is the neutral fold; custom
 * categories use their stored color in both modes.
 */
const system: Record<string, { light: string; dark: string }> = {
  shopping: { light: '#2a78d6', dark: '#3987e5' },
  fuel: { light: '#eb6834', dark: '#d95926' },
  food: { light: '#1baf7a', dark: '#199e70' },
  groceries: { light: '#eda100', dark: '#c98500' },
  entertainment: { light: '#e87ba4', dark: '#d55181' },
  travel: { light: '#008300', dark: '#008300' },
  medical: { light: '#4a3aa7', dark: '#9085e9' },
  utilities: { light: '#e34948', dark: '#e66767' },
  others: { light: '#8a8f99', dark: '#8a8f99' },
}

export const uncategorizedColor = '#8a8f99'

export function isDarkMode(): boolean {
  return typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
}

export function categoryColor(c?: Category | null, dark = isDarkMode()): string {
  if (!c) return uncategorizedColor
  if (c.isSystem && system[c.slug]) return dark ? system[c.slug].dark : system[c.slug].light
  return c.color || uncategorizedColor
}
