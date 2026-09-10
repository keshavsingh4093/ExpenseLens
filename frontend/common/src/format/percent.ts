/** Text tone for a budget-style percentage: green under 80%, amber to 100%, red over. */
export function percentTone(percent: number): string {
  return percent >= 100 ? 'text-red-600 dark:text-red-400' : percent >= 80 ? 'text-amber-600 dark:text-amber-400' : 'text-brand-700 dark:text-brand-400'
}
