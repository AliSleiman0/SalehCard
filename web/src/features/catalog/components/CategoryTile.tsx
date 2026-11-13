import { useNavigate } from 'react-router-dom'
import { ImageArt } from '@/components'
import { type ViewCategory } from '../lib/adaptCategory'

// CategoryTile renders a category as a clickable art tile: root domains (Home,
// /categories) and subcategories (drill-down grids), linking to `c.to` when set.
export function CategoryTile({ c }: { c: ViewCategory }) {
  const navigate = useNavigate()
  return (
    <div className="cattile hover-pop" onClick={() => navigate(c.to ?? '/category/' + c.key)}>
      <ImageArt
        art={c.art}
        src={c.image}
        word={c.name}
        sub={c.count !== undefined ? `${c.count}+` : ''}
        h={120}
        radius={0}
        wordSize={22}
      />
      <div className="cap">
        <div style={{ fontWeight: 800, fontSize: 15 }}>{c.name}</div>
        <div className="tiny faint">{c.tag}</div>
      </div>
    </div>
  )
}
