/** Natural A→Z by product name: case-insensitive, accents/Arabic collated by
 * the browser, and numbers compared by value ("JAWAKER 150" before
 * "JAWAKER 1,000"). Stable for equal names; does not mutate the input. */
const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

export function sortAZ<T extends { brand: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => collator.compare(a.brand.trim(), b.brand.trim()))
}
