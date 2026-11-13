/** Rows of a supplier catalog that match `term` (case-insensitive substring of
 * the product name, its category or the upstream id), in their original order.
 * Blank term returns everything. Client feedback 2026-11-12 #3. */
export function filterCatalog<T extends { name: string; category: string; upstreamId: string }>(
  rows: T[],
  term: string,
): T[] {
  const q = term.trim().toLowerCase()
  if (!q) return rows
  return rows.filter(
    (r) =>
      r.name.toLowerCase().includes(q) ||
      r.category.toLowerCase().includes(q) ||
      r.upstreamId.toLowerCase().includes(q),
  )
}
