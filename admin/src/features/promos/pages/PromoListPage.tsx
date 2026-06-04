import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, PageHead, StatusBadge, ComingSoonNote } from '@/components'
import { promos } from '@/lib/mock/demo'

const TYPE_BADGE: Record<string, [string, string]> = {
  percent: ['st-info', 'Percent'],
  fixed: ['st-ok', 'Fixed'],
  cashback: ['st-warn', 'Cashback'],
}

export default function PromoListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  // TODO: wire to GET /api/admin/promos (mock; backend route stubbed 501).
  return (
    <div className="page page-wide">
      <PageHead crumbs={[t('grp_finance'), t('nav_promos')]} title="Promo codes" sub="6 campaigns · 4 active">
        <button className="abtn primary" onClick={() => navigate('/promos/new')}>
          <Icon name="plus" size={15} /> New promo
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder="Search codes…" />
          </div>
          <select className="select">
            <option>All types</option>
            <option>Percent</option>
            <option>Fixed</option>
            <option>Cashback</option>
          </select>
          <select className="select">
            <option>All statuses</option>
            <option>Active</option>
            <option>Expired</option>
            <option>Depleted</option>
          </select>
          <div className="tb-spacer" />
        </div>
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Code</th>
                <th>Type</th>
                <th>Value</th>
                <th>Usage</th>
                <th>Valid period</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {promos.map((p) => {
                const [cls, label] = TYPE_BADGE[p.type]
                const pct = Math.min(100, Math.round((p.used / p.limit) * 100))
                return (
                  <tr key={p.code} className="clickable" onClick={() => navigate(`/promos/${p.code}/edit`)}>
                    <td>
                      <span
                        className="mono strong"
                        style={{
                          fontSize: 13.5,
                          padding: '3px 8px',
                          background: 'var(--surface-2)',
                          borderRadius: 6,
                          border: '1px dashed var(--border-strong)',
                        }}
                      >
                        {p.code}
                      </span>
                    </td>
                    <td>
                      <span className={'st ' + cls}>
                        <i className="d" />
                        {label}
                      </span>
                    </td>
                    <td className="num strong">{p.type === 'fixed' ? '$' + p.value : p.value + '%'}</td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <span className="num muted" style={{ minWidth: 78, fontSize: 12.5 }}>
                          {p.used.toLocaleString()} / {p.limit > 50000 ? '∞' : p.limit.toLocaleString()}
                        </span>
                        <div className="meter" style={{ width: 70 }}>
                          <i style={{ width: pct + '%', background: pct > 90 ? 'var(--danger)' : 'var(--grad)' }} />
                        </div>
                      </div>
                    </td>
                    <td className="muted" style={{ fontSize: 12.5 }}>
                      {p.start} – {p.end}
                    </td>
                    <td>
                      <StatusBadge s={p.status} />
                    </td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <div className="row-actions">
                        <span className="iact" onClick={() => navigate(`/promos/${p.code}/edit`)}>
                          <Icon name="edit" size={15} />
                        </span>
                        <span className="iact danger">
                          <Icon name="trash" size={15} />
                        </span>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
