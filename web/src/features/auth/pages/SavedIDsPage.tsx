import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, ImageArt, Button, useToast } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { gameToProduct } from '@/features/auth/components/SavedIdCard'
import { DEMO } from '@/lib/mock/demo'

export default function SavedIDsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="row between" style={{ marginBottom: 24 }}>
        <h1 className="h1">{t('saved_players')}</h1>
      </div>
      <div className="cols-acct">
        <AcctSidebar active="savedids" />
        <div className="col" style={{ gap: 16 }}>
          <div className="grid" style={{ gridTemplateColumns: 'repeat(2,1fr)', gap: 16 }}>
            {DEMO.savedIds.map((s) => (
              <div key={s.id} className="panel card-pad">
                <div className="row between">
                  <div className="row" style={{ gap: 12 }}>
                    <ImageArt
                      art={s.art}
                      word={s.game.split(' ')[0]}
                      h={48}
                      wordSize={12}
                      radius={12}
                      style={{ width: 64, flex: 'none' }}
                    />
                    <div className="col" style={{ gap: 2 }}>
                      <span style={{ fontWeight: 800 }}>{s.game}</span>
                      <span className="tiny faint">
                        {s.label}
                        {s.nick && ` · ${s.nick}`}
                      </span>
                    </div>
                  </div>
                  <button
                    className="icon-btn"
                    style={{ width: 32, height: 32 }}
                    onClick={() => toast(t('manage'), 'user')}
                  >
                    <Icon name="trash" size={15} />
                  </button>
                </div>
                <div className="vault" style={{ marginTop: 14, fontSize: 14, padding: '12px 14px' }}>
                  <span className="code" style={{ filter: 'none' }}>
                    {s.value}
                  </span>
                  <Button
                    variant="cyan"
                    size="sm"
                    onClick={() => navigate('/product/' + gameToProduct(s.game))}
                  >
                    <Icon name="repeat" size={14} />
                    {t('reorder')}
                  </Button>
                </div>
              </div>
            ))}
            <button
              className="panel card-pad clickable"
              style={{
                border: '1.5px dashed var(--border-strong)',
                display: 'grid',
                placeItems: 'center',
                minHeight: 150,
                background: 'transparent',
              }}
              onClick={() => toast(t('add_id'), 'plus')}
            >
              <div className="col center" style={{ gap: 8, color: 'var(--text-dim)' }}>
                <Icon name="plusc" size={30} />
                <span style={{ fontWeight: 700 }}>{t('add_id')}</span>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
