import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Avatar, StatusBadge } from '@/components'
import { useUsers } from '@/features/users/hooks/useUsers'
import { adaptUser } from '@/features/users/lib/adaptUser'
import { lastActive } from '@/lib/utils'

// Settings is intentionally minimal: the only real, persisted surface is the
// admin-accounts list (backed by /api/admin/users?role=admin). Store config,
// payment gateways, and notification thresholds have no backend yet — the
// former mock tabs were removed rather than shipped as a fake Save button.
export default function SettingsPage() {
  const { t } = useTranslation()
  const { data: adminsRes, isLoading: adminsLoading } = useUsers({ role: 'admin', limit: 100 })
  const adminRows = (adminsRes?.data ?? []).map(adaptUser)

  return (
    <div className="page">
      <PageHead
        crumbs={[t('grp_system'), t('nav_settings')]}
        title={t('nav_settings')}
        sub="Admin accounts — promote or suspend admins from each user's profile"
      />

      <div className="acard">
        <div className="panelhead">
          <Icon name="shield" size={17} />
          <h3>Admin accounts</h3>
        </div>
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Admin</th>
                <th>Role</th>
                <th>Last active</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {adminRows.length === 0 && (
                <tr>
                  <td colSpan={4} className="muted" style={{ textAlign: 'center', padding: 20 }}>
                    {adminsLoading ? 'Loading…' : 'No admin accounts found.'}
                  </td>
                </tr>
              )}
              {adminRows.map((a) => (
                <tr key={a.id}>
                  <td>
                    <div className="cellprod">
                      <Avatar name={a.name} />
                      <div className="pn">
                        <b>{a.name}</b>
                        <span>{a.email}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span
                      className="pill-role"
                      style={{
                        background: 'rgba(214,51,255,.16)',
                        color: '#d883ff',
                        border: '1px solid var(--border)',
                      }}
                    >
                      Admin
                    </span>
                  </td>
                  <td className="muted" style={{ fontSize: 12.5 }}>
                    {lastActive(a.raw.lastSeen)}
                  </td>
                  <td>
                    <StatusBadge s={a.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div
          style={{
            padding: 16,
            borderTop: '1px solid var(--border)',
            display: 'flex',
            gap: 8,
            alignItems: 'center',
            color: 'var(--text-dim)',
            fontSize: 12.5,
          }}
        >
          <Icon name="alert" size={15} /> Grant or revoke admin access from a user's profile
          (Users → role). Demoting or suspending the last remaining admin is blocked.
        </div>
      </div>
    </div>
  )
}
