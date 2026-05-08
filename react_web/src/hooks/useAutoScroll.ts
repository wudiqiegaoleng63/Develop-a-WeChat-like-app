import { useEffect, useRef } from 'react'

export function useAutoScroll(deps: React.DependencyList) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return ref
}
