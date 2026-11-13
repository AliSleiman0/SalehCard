import { describe, it, expect } from 'vitest'
import { adaptNode, crumbLink, crumbName, localizedName } from './adaptCategory'
import type { Category, CategoryCrumb } from '@/types'

function cat(over: Partial<Category>): Category {
  return {
    id: 'n1',
    slug: 'jawaker',
    name: { en: 'Jawaker', ar: 'جواكر', tr: '' },
    sortOrder: 2,
    rootDomain: 'games',
    depth: 1,
    visible: true,
    ...over,
  } as Category
}

describe('adaptNode', () => {
  it('maps a subcategory to a drill-down tile', () => {
    const v = adaptNode(cat({ productCount: 7, image: 'https://x/img.jpg' }), 'ar')
    expect(v).toMatchObject({
      key: 'n1',
      name: 'جواكر',
      to: '/catalog/n1',
      count: 7,
      image: 'https://x/img.jpg',
      order: 2,
    })
  })

  it('keeps roots on the whole-domain page with the curated label', () => {
    const v = adaptNode(cat({ id: 'r', depth: 0, slug: 'games', name: { en: 'شحن', ar: '', tr: '' } }), 'en')
    expect(v.key).toBe('games')
    expect(v.name).toBe('Games')
    expect(v.to).toBeUndefined()
  })
})

describe('breadcrumb helpers', () => {
  const root: CategoryCrumb = { id: 'r', slug: 'games', name: { en: 'x', ar: 'الألعاب', tr: '' }, depth: 0, rootDomain: 'games' }
  const mid: CategoryCrumb = { id: 'm', slug: 'jawaker', name: { en: 'Jawaker', ar: '', tr: '' }, depth: 1, rootDomain: 'games' }

  it('links roots to the domain page and deeper nodes to the drill-down', () => {
    expect(crumbLink(root)).toBe('/category/games')
    expect(crumbLink(mid)).toBe('/catalog/m')
  })

  it('labels roots with the curated name (Arabic DB name in Arabic)', () => {
    expect(crumbName(root, 'en')).toBe('Games')
    expect(crumbName(root, 'ar')).toBe('الألعاب')
    expect(crumbName(mid, 'ar')).toBe('Jawaker') // falls back to en
  })

  it('localizedName falls back en → slug', () => {
    expect(localizedName({ name: { en: '', ar: '' }, slug: 'cat-1' }, 'tr')).toBe('cat-1')
  })
})
