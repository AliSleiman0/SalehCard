import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Art, Stars, StatusBadge, Checkbox, Chip, ComingSoonNote } from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { reviews } from '@/lib/mock/demo'

export default function ReviewsPage() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState('pending')
  // TODO: wire to GET/PUT /api/admin/reviews (mock; backend route stubbed 501).
  const rows = reviews.filter((r) => filter === 'all' || r.status === filter)
  const bulk = useBulk(rows.map((r) => r.id))

  return (
    <div className="page page-wide">
      <PageHead crumbs={[t('grp_content'), t('nav_reviews')]} title="Review moderation" sub="4 reviews awaiting moderation">
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {(
              [
                ['pending', 'Pending', 4],
                ['approved', 'Approved', null],
                ['rejected', 'Rejected', null],
                ['all', t('all'), null],
              ] as [string, string, number | null][]
            ).map(([k, l, n]) => (
              <Chip key={k} on={filter === k} onClick={() => setFilter(k)}>
                {l}
                {n != null && (
                  <span className="sb-badge" style={{ position: 'static', display: 'inline-grid' }}>
                    {n}
                  </span>
                )}
              </Chip>
            ))}
          </div>
          <div className="tb-spacer" />
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder="Search reviews…" />
          </div>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs ok">
                <Icon name="check" size={13} /> {t('approve')}
              </button>
              <button className="abtn xs danger">
                <Icon name="x" size={13} /> {t('reject')}
              </button>
            </div>
          </div>
        )}

        <div>
          {rows.map((r) => (
            <div key={r.id} style={{ borderBottom: '1px solid var(--border)' }}>
              <div style={{ display: 'flex', gap: 14, padding: '14px 18px', alignItems: 'flex-start' }}>
                <div style={{ paddingTop: 3 }}>
                  <Checkbox on={bulk.sel.includes(r.id)} onClick={() => bulk.toggle(r.id)} />
                </div>
                <Art art={r.art} size={42} radius={10} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4, flexWrap: 'wrap' }}>
                    <b style={{ fontSize: 13.5 }}>{r.product}</b>
                    <Stars n={r.rating} size={13} />
                    <span className="faint" style={{ fontSize: 12 }}>
                      by {r.user} · {r.date}
                    </span>
                    <span style={{ marginInlineStart: 'auto' }}>
                      <StatusBadge s={r.status} />
                    </span>
                  </div>
                  <p style={{ fontSize: 13.5, color: 'var(--text-dim)', lineHeight: 1.5, maxWidth: 760 }}>{r.body}</p>
                  <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
                    {r.status === 'pending' ? (
                      <>
                        <button className="abtn xs ok">
                          <Icon name="check" size={13} /> {t('approve')}
                        </button>
                        <button className="abtn xs danger">
                          <Icon name="x" size={13} /> {t('reject')}
                        </button>
                        <button className="abtn xs">
                          <Icon name="trash" size={13} /> {t('delete')}
                        </button>
                      </>
                    ) : (
                      <button className="abtn xs">
                        <Icon name="refresh" size={13} /> Change status
                      </button>
                    )}
                    {r.rating <= 1 && (
                      <span className="st st-danger" style={{ fontSize: 11 }}>
                        <i className="d" />
                        Flagged · possible spam
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
