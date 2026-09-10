import { useQuery } from '@tanstack/react-query'
import { notificationsApi, notificationKeys } from './api'

/** Unread badge count, polled every 60s while the app is open. */
export function useUnreadCount() {
  const q = useQuery({ queryKey: notificationKeys.unread, queryFn: notificationsApi.unreadCount, refetchInterval: 60_000, staleTime: 30_000 })
  return q.data?.unread ?? 0
}
