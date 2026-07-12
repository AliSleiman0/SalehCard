import { useEffect, useState } from 'react'

// useDebouncedValue returns `value` after it has stopped changing for `ms`.
export function useDebouncedValue<T>(value: T, ms: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const h = setTimeout(() => setDebounced(value), ms)
    return () => clearTimeout(h)
  }, [value, ms])
  return debounced
}
