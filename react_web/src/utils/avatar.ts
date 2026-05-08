import { BACKEND_URL } from './constants'

export function normalizeAvatarUrl(url: string): string {
  if (!url) return ''
  if (url.startsWith('http')) return url
  return BACKEND_URL + url
}
