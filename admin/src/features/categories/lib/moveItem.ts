/** Moves list[from] to index `to`, returning a new array (the input is not
 *  mutated). Out-of-range or no-op moves return the input unchanged. */
export function moveItem<T>(list: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) return list
  const next = [...list]
  const [it] = next.splice(from, 1)
  next.splice(to, 0, it)
  return next
}
