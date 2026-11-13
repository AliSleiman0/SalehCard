/** Due-date state of an open task, for its badge colour. */
export type DueState = 'none' | 'overdue' | 'today' | 'upcoming'

/** Classifies a due date against `now` (local calendar days). */
export function dueState(dueAt: string | undefined, now: Date = new Date()): DueState {
  if (!dueAt) return 'none'
  const d = new Date(dueAt)
  if (Number.isNaN(d.getTime())) return 'none'
  const day = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  if (day(d) < day(now)) return 'overdue'
  if (day(d) === day(now)) return 'today'
  return 'upcoming'
}

/** A `<input type="date">` value (YYYY-MM-DD, local) → ISO at local noon, so
 * the date doesn't shift a day across time zones. Empty → undefined. */
export function dateInputToIso(v: string): string | undefined {
  if (!v) return undefined
  const [y, m, d] = v.split('-').map(Number)
  if (!y || !m || !d) return undefined
  return new Date(y, m - 1, d, 12, 0, 0).toISOString()
}
