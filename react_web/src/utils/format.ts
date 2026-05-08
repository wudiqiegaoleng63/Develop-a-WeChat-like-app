export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + units[i]
}

export function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  // If already formatted (e.g. "19:30" or "昨天"), return as-is
  if (dateStr.includes(':') && dateStr.length <= 10) return dateStr
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const isToday = date.toDateString() === now.toDateString()
    const time = date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    if (isToday) return time
    const yesterday = new Date(now)
    yesterday.setDate(yesterday.getDate() - 1)
    if (date.toDateString() === yesterday.toDateString()) return '昨天 ' + time
    return (date.getMonth() + 1) + '/' + date.getDate() + ' ' + time
  } catch {
    return dateStr
  }
}
