import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, PageHead, Avatar, StatusBadge, ComingSoonNote } from '@/components'
import { money } from '@/lib/utils'
import { resellers, tiers, tierColor } from '@/lib/mock/demo'

export default function ResellerListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  // TODO: wire to GET /api/admin/resellers + reseller-tiers (mock; routes stubbed 501).
  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_resellers')]}
        title="Reseller & agent management"
        sub="67 active resellers across 3 tiers"
      >
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn primary">
          <Icon name="plus" size={15} /> Add reseller
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="g3 mb16">
        {tiers.map((tier) => (
          <div className="acard pad" key={tier.name} style={{ borderColor: tier.color + '55' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 12 }}>
              <span
                style={{
                  width: 30,
                  height: 30,
                  borderRadius: 9,
                  background: tier.color + '22',
                  color: tier.color,
                  display: 'grid',
                  placeItems: 'center',
                }}
              >
                <Icon name="shield" size={16} />
              </span>
              <b style={{ fontSize: 15 }}>{tier.name}</b>
              <span className="faint" style={{ marginInlineStart: 'auto', fontSize: 12.5 }}>
                {tier.count} agents
              </span>
            </div>
            <div className="g2">
              <div>
                <div className="faint" style={{ fontSize: 11.5, fontWeight: 700 }}>
                  Discount
                </div>
                <div className="num" style={{ fontSize: 19, fontWeight: 800, color: tier.color }}>
                  {tier.discount}%
                </div>
              </div>
              <div>
                <div className="faint" style={{ fontSize: 11.5, fontWeight: 700 }}>
                  Balance limit
                </div>
                <div className="num" style={{ fontSize: 19, fontWeight: 800 }}>
                  ${tier.limit / 1000}k
                </div>
              </div>
            </div>
            <button className="abtn xs" style={{ width: '100%', marginTop: 12 }}>
              <Icon name="edit" size={13} /> Edit tier
            </button>
          </div>
        ))}
      </div>

      <div className="acard">
        <div className="panelhead">
          <Icon name="handshake" size={17} />
          <h3>All resellers</h3>
        </div>
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Reseller</th>
                <th>Tier</th>
                <th>Sub-balance</th>
                <th>Margin</th>
                <th>Orders</th>
                <th>Volume</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {resellers.map((r) => (
                <tr key={r.id} className="clickable" onClick={() => navigate(`/resellers/${r.id}`)}>
                  <td>
                    <div className="cellprod">
                      <Avatar name={r.name} color={tierColor[r.tier]} />
                      <div className="pn">
                        <b>{r.name}</b>
                        <span>{r.email}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span
                      className="pill-role"
                      style={{
                        background: tierColor[r.tier] + '22',
                        color: tierColor[r.tier],
                        border: '1px solid ' + tierColor[r.tier] + '55',
                      }}
                    >
                      {r.tier}
                    </span>
                  </td>
                  <td className="num strong">{money(r.balance)}</td>
                  <td className="num">{r.margin}%</td>
                  <td className="num muted">{r.orders.toLocaleString()}</td>
                  <td className="num">{money(r.vol)}</td>
                  <td>
                    <StatusBadge s={r.status} />
                  </td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <div className="row-actions">
                      <span className="iact">
                        <Icon name="chevright" size={16} />
                      </span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
