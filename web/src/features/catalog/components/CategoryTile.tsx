import { useNavigate } from 'react-router-dom'
import { ImageArt } from '@/components'
import { type ViewCategory } from '../lib/adaptCategory'

// CategoryTile renders a single root category as a clickable art tile.
// Shared by HomePage and the /categories index page.
export function CategoryTile({ c }: { c: ViewCategory }) {
  const navigate = useNavigate()
  return (
    <div className="cattile hover-pop" onClick={() => navigate('/category/' + c.key)}>
      <ImageArt
        art={c.art}
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
