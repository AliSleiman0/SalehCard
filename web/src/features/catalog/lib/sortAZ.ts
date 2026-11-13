/** Natural A→Z by product name: case-insensitive, accents/Arabic collated by
 * the browser, and numbers compared by value ("JAWAKER 150" before
 * "JAWAKER 1,000"). Stable for equal names; does not mutate the input. */
const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

// Leading punctuation is ignored ("(Sigma Plus" files under S); blanks go last.
const key = (s: string) => s.trim().replace(/^[^\p{L}\p{N}]+/u, '')

export function sortAZ<T extends { brand: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const ka = key(a.brand)
    const kb = key(b.brand)
    if (!ka !== !kb) return ka ? -1 : 1
    return collator.compare(ka, kb)
  })
}
