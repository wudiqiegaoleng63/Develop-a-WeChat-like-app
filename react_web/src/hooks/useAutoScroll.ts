import { useEffect, useRef, useCallback } from 'react'

export function useAutoScroll(deps: React.DependencyList) {
  const ref = useRef<HTMLDivElement>(null)
  const userScrolledUp = useRef(false)

  const handleScroll = useCallback(() => {
    if (!ref.current) return
    const { scrollTop, scrollHeight, clientHeight } = ref.current
    // If user is near the bottom (within 100px), they haven't scrolled up
    userScrolledUp.current = scrollHeight - scrollTop - clientHeight > 100
  }, [])

  useEffect(() => {
    if (ref.current && !userScrolledUp.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return { ref, handleScroll }
}
