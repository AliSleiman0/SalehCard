import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, ImageArt, Button } from '@/components'
import type { SavedId } from '@/lib/mock/demo'

export function gameToProduct(g: string): string {
  const m: Record<string, string> = {
    'PUBG MOBILE': 'pubg-uc',
    'MOBILE LEGENDS': 'mlbb',
    TIKTOK: 'tiktok',
  }
  return m[g] || 'pubg-uc'
}

export function SavedIdCard({ s, compact = false }: { s: SavedId; compact?: boolean }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  return (
    <div
      className="panel card-pad hover-pop"
      style={{ padding: 14, cursor: 'pointer' }}
      onClick={() => navigate('/product/' + gameToProduct(s.game))}
    >
      <div className="row" style={{ gap: 10 }}>
        <ImageArt
          art={s.art}
          word={s.game.split(' ')[0]}
          h={40}
          wordSize={11}
          radius={10}
          style={{ width: 52, flex: 'none' }}
        />
        <div className="col" style={{ gap: 1, minWidth: 0 }}>
          <span style={{ fontWeight: 800, fontSize: 13 }}>{s.game}</span>
          <span
            className="tiny faint num"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {s.value}
          </span>
        </div>
      </div>
      {!compact && (
        <div className="row between" style={{ marginTop: 10 }}>
          <span className="tiny faint">{s.label}</span>
        </div>
      )}
      <Button variant="cyan" size="sm" block style={{ marginTop: 12 }}>
        <Icon name="repeat" size={14} />
        {t('reorder')}
      </Button>
    </div>
  )
}
