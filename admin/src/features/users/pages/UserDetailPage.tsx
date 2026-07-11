import { useEffect, useState } from 'react'
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
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money, relativeTime } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useCan, useIsSuperAdmin } from '@/stores/auth'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import { useRoles } from '@/features/roles/hooks/useRoles'
import { useUser, useUpdateUserRole, useUpdateUserStatus, useAdjustWallet, useDeleteUser } from '../hooks/useUsers'
import { adaptUser } from '../lib/adaptUser'
import type { AdminUserDetail, UserStatus } from '../api/users'
import type { UserRole } from '@/types'

type Tab = 'overview' | 'orders' | 'wallet' | 'ids' | 'role'

export default function UserDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError, refetch } = useUser(id)
  const [tab, setTab] = useState<Tab>('overview')
  const [adjOpen, setAdjOpen] = useState(false)
  // The Order-history tab reads the orders domain — hide it (and its fetch) when
  // the admin's role lacks orders.view.
  const canViewOrders = useCan()('orders.view')

  const detail = data?.data
  const view = detail ? adaptUser(detail) : null

  const updateStatus = useUpdateUserStatus(id ?? '')

  if (isLoading) return <LoadingSpinner />
  if (isError || !detail || !view) {
    return (
      <div className="page">
        <ErrorState message="Couldn't load this user." onRetry={() => refetch()} />
      </div>
    )
  }

  const toggleStatus = () => {
    const next: UserStatus = view.status === 'suspended' ? 'active' : 'suspended'
    updateStatus.mutate(next)
  }

  return (
    <div className="page">
      <PageHead crumbs={[t('nav_users'), view.name]} title={view.name} sub={view.email || view.phone}>
        <button className="abtn" onClick={() => navigate('/users')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        {view.status === 'suspended' ? (
          <button className="abtn ok" onClick={toggleStatus} disabled={updateStatus.isPending}>
            <Icon name="check" size={15} /> Reactivate
          </button>
        ) : (
          <button className="abtn danger" onClick={toggleStatus} disabled={updateStatus.isPending}>
            <Icon name="x" size={15} /> {t('suspend')}
          </button>
        )}
      </PageHead>

      <div className="g3" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 18 }}>
        {(
          [
            ['Wallet balance', money(view.balance, view.cur), 'wallet'],
            ['Total orders', String(view.orders), 'bag'],
            ['Total spent', money(view.spent, view.cur), 'coins'],
            ['Loyalty points', view.loyalty.toLocaleString(), 'star'],
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
          ...(canViewOrders ? [{ k: 'orders' as const, label: 'Order history' }] : []),
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
                  ['Display name', view.name],
                  ['Email', view.email || '—'],
                  ['Phone', view.phone || '—'],
                  ['User ID', view.id],
                  ['Role', <RoleBadge key="r" role={view.role} />],
                  ['Status', <StatusBadge key="s" s={view.status} />],
                  ['Currency', view.cur],
                  ['Registered', view.joined],
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
                {money(view.balance, view.cur)}
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
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Loyalty</h3>
              <div className="deflist">
                <div className="defrow">
                  <span className="dk">Loyalty points</span>
                  <span className="dv num">{view.loyalty.toLocaleString()}</span>
                </div>
                <div className="defrow">
                  <span className="dk">Saved game IDs</span>
                  <span className="dv num">{detail.savedPlayerIds.length}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === 'orders' && <OrdersTab email={view.email} total={view.orders} />}

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
          {detail.transactions.length === 0 ? (
            <EmptyState title="No wallet activity yet" />
          ) : (
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
                  {detail.transactions.map((tx) => (
                    <tr key={tx.id}>
                      <td className="mono strong">{tx.id.slice(-8)}</td>
                      <td className="muted" style={{ textTransform: 'capitalize' }}>
                        {tx.type}
                      </td>
                      <td className="num strong" style={{ color: tx.amount < 0 ? 'var(--danger)' : 'var(--ok)' }}>
                        {tx.amount < 0 ? '−' : '+'}
                        {money(Math.abs(tx.amount), view.cur).replace('−', '')}
                      </td>
                      <td>{tx.method ? <PayChip p={tx.method} /> : <span className="faint">—</span>}</td>
                      <td className="muted" style={{ fontSize: 12 }}>
                        {relativeTime(tx.createdAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {tab === 'ids' && (
        <div className="g2">
          {detail.savedPlayerIds.length === 0 ? (
            <EmptyState title="No saved game IDs" />
          ) : (
            detail.savedPlayerIds.map((pid, i) => (
              <div
                className="acard pad"
                key={`${pid.label} ${pid.value} ${i}`}
                style={{ display: 'flex', alignItems: 'center', gap: 12 }}
              >
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
                  <div className="mono" style={{ fontWeight: 800, fontSize: 14 }}>
                    {pid.value}
                  </div>
                  <div className="faint" style={{ fontSize: 12.5 }}>
                    {pid.label}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {tab === 'role' && (
        <RoleTab
          id={view.id}
          role={view.role}
          adminRoleId={detail.adminRoleId ?? null}
          onSuspend={toggleStatus}
          suspending={updateStatus.isPending}
        />
      )}

      {adjOpen && <AdjustModal user={detail} onClose={() => setAdjOpen(false)} />}
    </div>
  )
}

/** Order history for the user — reuses the wired admin orders endpoint, searched
 *  by the customer's email (the list backend resolves email → their orders). */
function OrdersTab({ email, total }: { email: string; total: number }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useOrders({ q: email, limit: 5 })
  const rows = (data?.data ?? []).map(adaptOrder)

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="bag" size={17} />
        <h3>Order history</h3>
        <span className="faint" style={{ fontSize: 12.5, marginInlineStart: 6 }}>
          {total} completed
        </span>
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Couldn't load orders." />
      ) : rows.length === 0 ? (
        <EmptyState title="No orders yet" />
      ) : (
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
              {rows.map((o) => (
                <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                  <td className="mono strong">{o.id.slice(-8)}</td>
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
      )}
    </div>
  )
}

/** Role management + danger zone. Role changes persist via the admin endpoint.
 *  Granting/revoking admin access — and picking an admin's RBAC role — is
 *  restricted to super admins (the backend enforces the same with a 403). */
function RoleTab({
  id,
  role,
  adminRoleId,
  onSuspend,
  suspending,
}: {
  id: string
  role: UserRole
  adminRoleId: string | null
  onSuspend: () => void
  suspending: boolean
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isSuperAdmin = useIsSuperAdmin()
  const [pending, setPending] = useState<UserRole>(role)
  // '' = built-in Super Admin (no custom role assigned).
  const [pendingRoleId, setPendingRoleId] = useState<string>(adminRoleId ?? '')
  const updateRole = useUpdateUserRole(id)
  const del = useDeleteUser(id)
  // Custom roles for the assignment dropdown (any admin may list them; only
  // super admins see this control).
  const { data: rolesRes } = useRoles()
  const customRoles = rolesRes?.data ?? []

  // Keep the selection in sync if the underlying user changes (refetch).
  useEffect(() => setPending(role), [role])
  useEffect(() => setPendingRoleId(adminRoleId ?? ''), [adminRoleId])

  const remove = () => {
    if (
      !window.confirm(
        'Delete this account? It is anonymized — email/phone are freed and it can no longer sign in, but its orders and ledger are kept. This cannot be undone.',
      )
    )
      return
    del.mutate(undefined, { onSuccess: () => navigate('/users') })
  }

  // Non-super admins can't grant/revoke admin access, so they don't see the
  // Admin chip (role changes involving admin would 403 anyway). Promotion
  // between customer and reseller only needs users.manage.
  const roles: [UserRole, string][] = isSuperAdmin
    ? [
        ['customer', 'Customer'],
        ['reseller', 'Reseller'],
        ['admin', 'Admin'],
      ]
    : [
        ['customer', 'Customer'],
        ['reseller', 'Reseller'],
      ]

  const dirty = pending !== role || (pending === 'admin' && pendingRoleId !== (adminRoleId ?? ''))
  const save = () =>
    updateRole.mutate({
      role: pending,
      adminRoleId: pending === 'admin' && pendingRoleId ? pendingRoleId : null,
    })

  return (
    <div className="formgrid">
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Role management</h3>
        <label className="alabel">Account role</label>
        <div className="g3">
          {roles.map(([k, l]) => (
            <Chip key={k} on={pending === k} onClick={() => setPending(k)} style={{ justifyContent: 'center', padding: 12 }}>
              {l}
            </Chip>
          ))}
        </div>
        {pending === 'admin' && isSuperAdmin && (
          <div style={{ marginTop: 14 }}>
            <label className="alabel">Admin role (permissions)</label>
            <select
              className="select"
              style={{ width: '100%' }}
              value={pendingRoleId}
              onChange={(e) => setPendingRoleId(e.target.value)}
            >
              <option value="">Super Admin — full access, manages roles &amp; admins</option>
              {customRoles.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}
                  {r.description ? ` — ${r.description}` : ''}
                </option>
              ))}
            </select>
            <div className="ahint">
              Define roles under System → Roles. A permission change applies on the admin's next
              sign-in or session refresh.
            </div>
          </div>
        )}
        {!isSuperAdmin && (
          <div className="ahint">Only a super admin can grant or change admin access.</div>
        )}
        <div style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center' }}>
          <button
            className="abtn primary"
            disabled={!dirty || updateRole.isPending}
            onClick={save}
          >
            <Icon name="check" size={15} /> {t('save')}
          </button>
          {updateRole.isError && (
            <span style={{ color: 'var(--danger)', fontSize: 12.5 }}>
              {updateRole.error instanceof ApiError ? updateRole.error.message : "Couldn't update role."}
            </span>
          )}
        </div>
      </div>
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Danger zone</h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <button className="abtn danger" style={{ justifyContent: 'flex-start' }} onClick={onSuspend} disabled={suspending}>
            <Icon name="x" size={14} /> Suspend account
          </button>
          <button
            className="abtn danger"
            style={{ justifyContent: 'flex-start' }}
            onClick={remove}
            disabled={del.isPending}
          >
            <Icon name="trash" size={14} /> Delete account &amp; data
          </button>
        </div>
      </div>
    </div>
  )
}

function AdjustModal({ user, onClose }: { user: AdminUserDetail; onClose: () => void }) {
  const { t } = useTranslation()
  const [dir, setDir] = useState<'credit' | 'debit'>('credit')
  const [amount, setAmount] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const adjust = useAdjustWallet(user.id)

  const apply = () => {
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) {
      setError('Enter an amount greater than zero.')
      return
    }
    if (!reason.trim()) {
      setError('A reason is required.')
      return
    }
    setError('')
    adjust.mutate(
      { direction: dir, amount: amt, reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Adjustment failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 4 }}>Adjust wallet balance</h3>
        <div className="faint" style={{ fontSize: 12.5, marginBottom: 16 }}>
          {user.email || user.phone} · current {money(user.walletBalance)}
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
          Amount (USD)
        </label>
        <input
          className="afield"
          type="number"
          min="0"
          step="0.01"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          Reason (required)
        </label>
        <textarea
          className="afield"
          placeholder="Goodwill credit, manual refund, correction…"
          style={{ minHeight: 64 }}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>
        )}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={adjust.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={apply} disabled={adjust.isPending}>
            <Icon name="check" size={15} /> Apply adjustment
          </button>
        </div>
      </div>
    </Modal>
  )
}
