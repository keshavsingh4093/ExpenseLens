import { describe, expect, it } from 'vitest'
import { formatMoney } from './money'

describe('formatMoney', () => {
  it('formats whole rupees without decimals', () => {
    expect(formatMoney(2154000)).toBe('₹21,540')
  })
  it('keeps paise when present', () => {
    expect(formatMoney(123050)).toBe('₹1,230.50')
  })
})
