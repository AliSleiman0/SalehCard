import { useTranslation } from 'react-i18next'
import { useLocaleStore } from '@/stores/locale'
import { useCategories } from '../hooks/useCategories'
import { adaptNode } from '../lib/adaptCategory'
import { CategoryTile } from './CategoryTile'

// SubcategoryGrid shows a node's visible subcategories as tiles (client msgs
// #39/#43/#45/#53/#55: Jawaker, Clash of Clans, Alfa vs MTC… as their own
// sections). Renders nothing while loading or when the node has no children.
export function SubcategoryGrid({ parentId }: { parentId: string }) {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const { data } = useCategories({ parentId, withCounts: true })
  const tiles = (data?.data ?? [])
    .map((c) => adaptNode(c, locale))
    .sort((a, b) => a.order - b.order || a.name.localeCompare(b.name))
  if (tiles.length === 0) return null
  return (
    <section style={{ marginBottom: 28 }}>
      <h2 className="h3" style={{ marginBottom: 12 }}>
        {t('subcategories')}
      </h2>
      <div className="catgrid">
        {tiles.map((c) => (
          <CategoryTile key={c.key} c={c} />
        ))}
      </div>
    </section>
  )
}
