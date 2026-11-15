import { describe, expect, it } from 'vitest'
import en from './en.json'
import ar from './ar.json'
import tr from './tr.json'

type Dict = Record<string, string>
const PLURAL = /_(zero|one|two|few|many|other)$/

/** English plural forms collapse onto their base key; ar/tr carry the base key only. */
const baseKeys = (d: Dict) => new Set(Object.keys(d).map((k) => k.replace(PLURAL, '')))
const vars = (s: string) => [...s.matchAll(/\{\{\s*(\w+)[^}]*\}\}/g)].map((m) => m[1]).sort()

function valueFor(d: Dict, base: string): string | undefined {
  return d[base] ?? d[`${base}_other`] ?? d[`${base}_one`]
}

describe('locale parity', () => {
  const enKeys = baseKeys(en as Dict)

  for (const [name, dict] of [['ar', ar], ['tr', tr]] as const) {
    it(`${name} has exactly the English keys`, () => {
      const other = baseKeys(dict as Dict)
      expect([...enKeys].filter((k) => !other.has(k)), `missing in ${name}`).toEqual([])
      expect([...other].filter((k) => !enKeys.has(k)), `only in ${name}`).toEqual([])
    })

    it(`${name} uses the same placeholders as English`, () => {
      const wrong = [...enKeys].filter((k) => {
        const e = valueFor(en as Dict, k)
        const o = valueFor(dict as Dict, k)
        if (e === undefined || o === undefined) return false
        // English plural forms may drop {{count}} ("One product"); compare the rest.
        const strip = (v: string[]) => v.filter((x) => x !== 'count')
        return strip(vars(e)).join() !== strip(vars(o)).join()
      })
      expect(wrong).toEqual([])
    })
  }

  it('has no empty values', () => {
    const empty = [en, ar, tr].flatMap((d) =>
      Object.entries(d as Dict).filter(([, v]) => typeof v !== 'string' || v.trim() === '').map(([k]) => k),
    )
    expect(empty).toEqual([])
  })
})
