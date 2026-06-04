import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Art,
  RoleBadge,
  StatusBadge,
  FfBadge,
  PayChip,
  Chip,
  Tabs,
  Segmented,
  Modal,
  ComingSoonNote,
} from '@/components'
import { money } from '@/lib/utils'
import { users, orders, transactions } from '@/lib/mock/demo'
import type { DemoUser } from '@/lib/mock/demo'

type Tab = 'overview' | 'orders' | 'wallet' | 'ids' | 'role'

export default function UserDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  // TODO: wire to GET /api/admin/users/:id (mock; backend route stubbed 501).
  const u = users.find((x) => x.id === id) || users[0]
  const [tab, setTab] = useState<Tab>('overview')
  const [adjOpen, setAdjOpen] = useState(false)
  const uOrders = orders.slice(0, 5)

  return (
    <div className="page">
      <PageHead crumbs={[t('nav_users'), u.name]} title={u.name} sub={u.email}>
        <button className="abtn" onClick={() => navigate('/users')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn">
          <Icon name="send" size={15} /> Message
        </button>
        {u.status === 'active' ? (
          <button className="abtn danger">
            <Icon name="x" size={15} /> {t('suspend')}
          </button>
        ) : (
          <button className="abtn ok">
            <Icon name="check" size={15} /> Reactivate
          </button>
        )}
      </PageHead>

      <ComingSoonNote mock />

      <div className="g3" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 18 }}>
        {(
          [
            ['Wallet balance', money(u.balance, u.cur), 'wallet'],
            ['Total orders', String(u.orders), 'bag'],
            ['Total spent', money(u.spent, u.cur), 'coins'],
            ['Loyalty points', '2,140', 'star'],
          ] as [string, string, 'wallet' | 'bag' | 'coins' | 'star'][]
        ).map(([l, v, ic]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: 'var(--grad-soft)', color: 'var(--brand-1)' }}>
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
          { k: 'orders', label: 'Order history' },
          { k: 'wallet', label: 'Wallet & cashback' },
          { k: 'ids', label: 'Saved IDs' },
          { k: 'role', label: 'Role & access' },
        ]}
      />

      {tab === 'overview' && (
        <div className="formgrid">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Profile</h3>
            <div className="deflist">
              {(
                [
                  ['Full name', u.name],
                  ['Email', u.email],
                  ['User ID', u.id],
                  ['Role', <RoleBadge key="r" role={u.role} />],
                  ['Status', <StatusBadge key="s" s={u.status} />],
                  ['Currency', u.cur],
                  ['Registered', u.joined],
                  ['Last login', '2 hours ago'],
                ] as [string, React.ReactNode][]
              ).map(([k, v]) => (
                <div className="defrow" key={k}>
                  <span className="dk">{k}</span>
                  <span className="dv">{v}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="fieldset">
            <div
              className="acard pad"
              style={{
                background:
                  'radial-gradient(120% 130% at 100% 0%, rgba(214,51,255,.25), transparent 55%), linear-gradient(135deg,#1b1248,#2a1466)',
                border: 0,
                color: '#fff',
              }}
            >
              <div style={{ fontSize: 12.5, opacity: 0.8, fontWeight: 700 }}>Current balance</div>
              <div className="num" style={{ fontSize: 34, fontWeight: 800, letterSpacing: '-.02em', margin: '4px 0 14px' }}>
                {money(u.balance, u.cur)}
              </div>
              <button
                className="abtn sm"
                style={{ background: 'rgba(255,255,255,.16)', border: 0, color: '#fff' }}
                onClick={() => setAdjOpen(true)}
              >
                <Icon name="edit" size={14} /> Adjust balance
              </button>
            </div>
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Cashback &amp; loyalty</h3>
              <div className="deflist">
                <div className="defrow">
                  <span className="dk">Cashback earned</span>
                  <span className="dv num">$48.20</span>
                </div>
                <div className="defrow">
                  <span className="dk">Loyalty points</span>
                  <span className="dv num">2,140</span>
                </div>
                <div className="defrow">
                  <span className="dk">Tier</span>
                  <span className="dv">Silver member</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === 'orders' && (
        <div className="acard">
          <div className="panelhead">
            <Icon name="bag" size={17} />
            <h3>Order history</h3>
            <span className="faint" style={{ fontSize: 12.5, marginInlineStart: 6 }}>
              {u.orders} total
            </span>
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
                {uOrders.map((o) => (
                  <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                    <td className="mono strong">{o.id}</td>
                    <td>
                      <div className="cellprod">
                        <Art art={o.art} size={28} radius={6} />
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

      {tab === 'wallet' && (
        <div className="acard">
          <div className="panelhead">
            <Icon name="wallet" size={17} />
            <h3>Wallet transactions</h3>
            <div className="ph-act">
              <button className="abtn xs primary" onClick={() => setAdjOpen(true)}>
                <Icon name="edit" size={13} /> {t('adjust')}
              </button>
            </div>
          </div>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Method</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {transactions.slice(0, 6).map((tx) => (
                  <tr key={tx.id}>
                    <td className="mono strong">{tx.id}</td>
                    <td className="muted" style={{ textTransform: 'capitalize' }}>
                      {tx.type}
                    </td>
                    <td className="num strong" style={{ color: tx.amount < 0 ? 'var(--danger)' : 'var(--ok)' }}>
                      {tx.amount < 0 ? '−' : '+'}
                      {money(Math.abs(tx.amount), tx.cur).replace('−', '')}
                    </td>
                    <td>{tx.method === '—' ? <span className="faint">—</span> : <PayChip p={tx.method} />}</td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {tx.date}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'ids' && (
        <div className="g2">
          {(
            [
              ['PUBG Mobile', '5521 8842 1190'],
              ['Mobile Legends', '8842 · Server 2201'],
              ['TikTok', '@yusuf.plays'],
              ['Bigo Live', 'ID 44120882'],
            ] as [string, string][]
          ).map(([game, gid]) => (
            <div className="acard pad" key={game} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: 10,
                  background: 'var(--grad-soft)',
                  display: 'grid',
                  placeItems: 'center',
                  color: 'var(--brand-1)',
                }}
              >
                <Icon name="id" size={18} />
              </div>
              <div>
                <div style={{ fontWeight: 800, fontSize: 14 }}>{game}</div>
                <div className="mono faint" style={{ fontSize: 12.5 }}>
                  {gid}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {tab === 'role' && (
        <div className="formgrid">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Role management</h3>
            <label className="alabel">Account role</label>
            <div className="g3">
              {(
                [
                  ['customer', 'Customer'],
                  ['reseller', 'Reseller'],
                  ['admin', 'Admin'],
                ] as [string, string][]
              ).map(([k, l]) => (
                <Chip key={k} on={u.role === k} style={{ justifyContent: 'center', padding: 12 }}>
                  {l}
                </Chip>
              ))}
            </div>
            <div className="ahint">
              Promoting to reseller unlocks wholesale pricing and a sub-balance. Promoting to admin grants console access.
            </div>
            <div style={{ marginTop: 16, display: 'flex', gap: 10 }}>
              <button className="abtn primary">
                <Icon name="check" size={15} /> {t('save')}
              </button>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Danger zone</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <button className="abtn danger" style={{ justifyContent: 'flex-start' }}>
                <Icon name="x" size={14} /> Suspend account
              </button>
              <button className="abtn danger" style={{ justifyContent: 'flex-start' }}>
                <Icon name="trash" size={14} /> Delete account &amp; data
              </button>
            </div>
          </div>
        </div>
      )}

      {adjOpen && <AdjustModal user={u} onClose={() => setAdjOpen(false)} />}
    </div>
  )
}

function AdjustModal({ user, onClose }: { user: DemoUser; onClose: () => void }) {
  const { t } = useTranslation()
  const [dir, setDir] = useState<'credit' | 'debit'>('credit')
  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 4 }}>Adjust wallet balance</h3>
        <div className="faint" style={{ fontSize: 12.5, marginBottom: 16 }}>
          {user.name} · current {money(user.balance, user.cur)}
        </div>
        <Segmented<'credit' | 'debit'>
          block
          value={dir}
          onChange={setDir}
          items={[
            { k: 'credit', label: 'Credit (+)' },
            { k: 'debit', label: 'Debit (−)' },
          ]}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          Amount ({user.cur})
        </label>
        <input className="afield" placeholder="0.00" />
        <label className="alabel" style={{ marginTop: 14 }}>
          Reason (required)
        </label>
        <textarea className="afield" placeholder="Goodwill credit, manual refund, correction…" style={{ minHeight: 64 }} />
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={onClose}>
            <Icon name="check" size={15} /> Apply adjustment
          </button>
        </div>
      </div>
    </Modal>
  )
}
