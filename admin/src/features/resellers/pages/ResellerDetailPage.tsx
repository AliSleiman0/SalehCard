import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Art, FfBadge, StatusBadge, Spark, Chip, Tabs, Segmented, ComingSoonNote } from '@/components'
import { money } from '@/lib/utils'
import { resellers, orders, tierColor } from '@/lib/mock/demo'

type Tab = 'overview' | 'balance' | 'pricing' | 'orders'

export default function ResellerDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  // TODO: wire to GET /api/admin/resellers/:id (mock; backend route stubbed 501).
  const r = resellers.find((x) => x.id === id) || resellers[0]
  const [tab, setTab] = useState<Tab>('overview')
  const tc = tierColor[r.tier]

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_resellers'), r.name]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            {r.name}{' '}
            <span className="pill-role" style={{ background: tc + '22', color: tc, border: '1px solid ' + tc + '55' }}>
              {r.tier} agent
            </span>
          </span>
        }
        sub={r.email}
      >
        <button className="abtn" onClick={() => navigate('/resellers')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary">
          <Icon name="coins" size={15} /> Top up sub-balance
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="g3" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 18 }}>
        {(
          [
            ['Sub-balance', money(r.balance), 'wallet'],
            ['Avg. margin', r.margin + '%', 'activity'],
            ['Total orders', r.orders.toLocaleString(), 'bag'],
            ['Lifetime volume', money(r.vol), 'coins'],
          ] as [string, string, 'wallet' | 'activity' | 'bag' | 'coins'][]
        ).map(([l, v, ic]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: tc + '22', color: tc }}>
                <Icon name={ic} size={16} />
              </div>
              <div className="k-label">{l}</div>
            </div>
            <div className="k-val" style={{ fontSize: 22 }}>
              {v}
            </div>
          </div>
        ))}
      </div>

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'overview', label: 'Overview' },
          { k: 'balance', label: 'Sub-balance' },
          { k: 'pricing', label: 'Pricing rules' },
          { k: 'orders', label: 'Order history' },
        ]}
      />

      {tab === 'overview' && (
        <div className="formgrid">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Performance</h3>
            <Spark data={[20, 28, 24, 36, 40, 38, 48, 52, 49, 60, 64, 71]} color={tc} h={70} w={400} />
            <div className="deflist" style={{ marginTop: 8 }}>
              <div className="defrow">
                <span className="dk">This month</span>
                <span className="dv num">142 orders · {money(8400)}</span>
              </div>
              <div className="defrow">
                <span className="dk">Avg. order value</span>
                <span className="dv num">{money(38.7)}</span>
              </div>
              <div className="defrow">
                <span className="dk">Top category</span>
                <span className="dv">Game Top-ups</span>
              </div>
              <div className="defrow">
                <span className="dk">Member since</span>
                <span className="dv">Jan 2024</span>
              </div>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Tier assignment</h3>
            <div className="g3">
              {['Bronze', 'Silver', 'Gold'].map((tn) => (
                <Chip key={tn} on={r.tier === tn} style={{ justifyContent: 'center', padding: 12 }}>
                  {tn}
                </Chip>
              ))}
            </div>
            <div className="ahint">
              Tier sets the default discount ({r.margin}%) and balance limit.
            </div>
            <button className="abtn primary" style={{ marginTop: 14 }}>
              <Icon name="check" size={15} /> {t('save')}
            </button>
          </div>
        </div>
      )}

      {tab === 'balance' && (
        <div className="formgrid">
          <div className="acard">
            <div className="panelhead">
              <Icon name="wallet" size={17} />
              <h3>Sub-balance transactions</h3>
            </div>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Transaction</th>
                    <th>Type</th>
                    <th>Amount</th>
                    <th>Date</th>
                  </tr>
                </thead>
                <tbody>
                  {(
                    [
                      ['TX-50230', 'Top-up', '+2,000.00', 'Jun 4 · 11:20'],
                      ['TX-50198', 'Purchase', '−240.00', 'Jun 3 · 19:02'],
                      ['TX-50140', 'Admin adjust', '+150.00', 'Jun 2 · 09:44'],
                      ['TX-50090', 'Purchase', '−88.00', 'Jun 1 · 16:30'],
                    ] as [string, string, string, string][]
                  ).map(([txid, ty, amt, dt]) => (
                    <tr key={txid}>
                      <td className="mono strong">{txid}</td>
                      <td className="muted">{ty}</td>
                      <td className="num strong" style={{ color: amt[0] === '+' ? 'var(--ok)' : 'var(--danger)' }}>
                        {amt[0] === '+' ? '+$' : '−$'}
                        {amt.slice(1)}
                      </td>
                      <td className="muted" style={{ fontSize: 12 }}>
                        {dt}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Adjust sub-balance</h3>
            <Segmented<'topup' | 'deduct'>
              block
              value="topup"
              onChange={() => {}}
              items={[
                { k: 'topup', label: 'Top up (+)' },
                { k: 'deduct', label: 'Deduct (−)' },
              ]}
            />
            <label className="alabel" style={{ marginTop: 14 }}>
              Amount (USD)
            </label>
            <input className="afield" placeholder="0.00" />
            <label className="alabel" style={{ marginTop: 12 }}>
              Note
            </label>
            <input className="afield" placeholder="Reference / reason" />
            <button className="abtn primary" style={{ marginTop: 16, width: '100%' }}>
              <Icon name="check" size={15} /> Apply
            </button>
          </div>
        </div>
      )}

      {tab === 'pricing' && (
        <div className="acard">
          <div className="panelhead">
            <Icon name="tag" size={17} />
            <h3>Pricing rules</h3>
            <div className="ph-act">
              <button className="abtn xs">
                <Icon name="plus" size={13} /> Add override
              </button>
            </div>
          </div>
          <div style={{ padding: 18 }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                padding: 14,
                background: 'var(--surface-2)',
                borderRadius: 'var(--ar-sm)',
                marginBottom: 14,
              }}
            >
              <div style={{ flex: 1 }}>
                <b style={{ fontSize: 13.5 }}>Global discount</b>
                <div className="faint" style={{ fontSize: 12 }}>
                  Applied to all products by default
                </div>
              </div>
              <input className="afield" style={{ width: 80, padding: '7px 10px' }} defaultValue={r.margin} />
              <span style={{ fontWeight: 800 }}>%</span>
            </div>
            <label className="alabel">Per-product overrides</label>
            <table className="tbl">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Type</th>
                  <th>Override</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {(
                  [
                    ['Steam Wallet (US)', 'fixed', '−$1.50 / unit'],
                    ['USDT Voucher', 'percent', '15%'],
                    ['PUBG Mobile UC', 'percent', '14%'],
                  ] as [string, string, string][]
                ).map(([p, ty, ov]) => (
                  <tr key={p}>
                    <td className="strong">{p}</td>
                    <td className="muted" style={{ textTransform: 'capitalize' }}>
                      {ty}
                    </td>
                    <td className="num strong">{ov}</td>
                    <td>
                      <span className="iact danger">
                        <Icon name="trash" size={15} />
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'orders' && (
        <div className="acard">
          <div className="panelhead">
            <Icon name="bag" size={17} />
            <h3>Order history</h3>
          </div>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Product</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {orders.slice(0, 6).map((o) => (
                  <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                    <td className="mono strong">{o.id}</td>
                    <td>
                      <div className="cellprod">
                        <Art art={o.art} size={26} radius={6} />
                        <div className="pn">
                          <b style={{ fontSize: 12.5 }}>{o.product}</b>
                        </div>
                      </div>
                    </td>
                    <td>
                      <FfBadge ff={o.ff} />
                    </td>
                    <td className="num strong">{money(o.amount, o.cur)}</td>
                    <td>
                      <StatusBadge s={o.status} />
                    </td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {o.date}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
