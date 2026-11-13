import { describe, expect, it } from 'vitest'
import { moveItem } from './moveItem'

describe('moveItem', () => {
  const list = ['a', 'b', 'c', 'd']

  it('moves an item down and up', () => {
    expect(moveItem(list, 0, 2)).toEqual(['b', 'c', 'a', 'd'])
    expect(moveItem(list, 3, 1)).toEqual(['a', 'd', 'b', 'c'])
  })

  it('moves to the ends', () => {
    expect(moveItem(list, 2, 0)).toEqual(['c', 'a', 'b', 'd'])
    expect(moveItem(list, 1, 3)).toEqual(['a', 'c', 'd', 'b'])
  })

  it('does not mutate the input', () => {
    const copy = [...list]
    moveItem(list, 0, 3)
    expect(list).toEqual(copy)
  })

  it('ignores no-op and out-of-range moves', () => {
    expect(moveItem(list, 1, 1)).toBe(list)
    expect(moveItem(list, -1, 2)).toBe(list)
    expect(moveItem(list, 0, 4)).toBe(list)
  })
})
