import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Avatar,
  RoleBadge,
  StatusBadge,
  Checkbox,
  Chip,
  Pagination,
  ComingSoonNote,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { money } from '@/lib/utils'
import { users } from '@/lib/mock/demo'

export default function UserListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [role, setRole] = useState('all')
  // TODO: wire to GET /api/admin/users (mock; backend route stubbed 501).
  const rows = users.filter((u) => role === 'all' || u.role === role)
  const bulk = useBulk(rows.map((r) => r.id))

  return (
    <div className="page page-wide">
      <PageHead crumbs={[t('grp_operations'), t('nav_users')]} title={t('nav_users')} sub="12,480 registered users">
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn primary">
          <Icon name="plus" size={15} /> Invite user
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder="Search name or email…" />
          </div>
          <div className="chiprow">
            {(
              [
                ['all', t('all')],
                ['customer', 'Customers'],
                ['reseller', 'Resellers'],
                ['admin', 'Admins'],
              ] as [string, string][]
            ).map(([k, l]) => (
              <Chip key={k} on={role === k} onClick={() => setRole(k)}>
                {l}
              </Chip>
            ))}
          </div>
          <select className="select">
            <option>All statuses</option>
            <option>Active</option>
            <option>Suspended</option>
          </select>
          <div className="tb-spacer" />
          <button className="abtn sm">
            <Icon name="filter" size={14} /> {t('filter')}
          </button>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs">
                <Icon name="send" size={13} /> Email
              </button>
              <button className="abtn xs danger">
                <Icon name="x" size={13} /> {t('suspend')}
              </button>
            </div>
          </div>
        )}

        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 36 }}>
                  <Checkbox on={bulk.all} onClick={bulk.toggleAll} />
                </th>
                <th>User</th>
                <th>Role</th>
                <th>Wallet</th>
                <th>Orders</th>
                <th>Total spent</th>
                <th>Joined</th>
                <th>{t('status')}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((u) => (
                <tr key={u.id} className="clickable" onClick={() => navigate(`/users/${u.id}`)}>
                  <td onClick={(e) => e.stopPropagation()}>
                    <Checkbox on={bulk.sel.includes(u.id)} onClick={() => bulk.toggle(u.id)} />
                  </td>
                  <td>
                    <div className="cellprod">
                      <Avatar name={u.name} />
                      <div className="pn">
                        <b>{u.name}</b>
                        <span>{u.email}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <RoleBadge role={u.role} />
                  </td>
                  <td className="num strong">{money(u.balance, u.cur)}</td>
                  <td className="num muted">{u.orders}</td>
                  <td className="num">{money(u.spent, u.cur)}</td>
                  <td className="muted" style={{ fontSize: 12.5 }}>
                    {u.joined}
                  </td>
                  <td>
                    <StatusBadge s={u.status} />
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
        <Pagination total={12480} pages={5} label="users" />
      </div>
    </div>
  )
}
