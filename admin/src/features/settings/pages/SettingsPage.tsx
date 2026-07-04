import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Avatar, StatusBadge } from '@/components'
import { useUsers } from '@/features/users/hooks/useUsers'
import { adaptUser } from '@/features/users/lib/adaptUser'
import { lastActive } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useSettings, useUpdateSettings } from '../hooks/useSettings'

// The persisted surfaces here are the admin-accounts list (backed by
// /api/admin/users?role=admin) and the Loyalty program knobs (backed by
// /api/admin/settings). Other store config (gateways, notification thresholds)
// still has no admin form — those fields exist on the settings doc but aren't
// edited here yet.
export default function SettingsPage() {
  const { t } = useTranslation()
  const { data: adminsRes, isLoading: adminsLoading } = useUsers({ role: 'admin', limit: 100 })
  const adminRows = (adminsRes?.data ?? []).map(adaptUser)

  // --- Loyalty program ------------------------------------------------------
  const { data: settingsRes, isLoading: settingsLoading } = useSettings()
  const update = useUpdateSettings()
  const [loyaltyEnabled, setLoyaltyEnabled] = useState(true)
  const [earnRate, setEarnRate] = useState('5')
  const [loyaltyError, setLoyaltyError] = useState<string | null>(null)
  const [loyaltySaved, setLoyaltySaved] = useState(false)

  useEffect(() => {
    const s = settingsRes?.data
    if (!s) return
    setLoyaltyEnabled(s.loyaltyEnabled)
    setEarnRate(String(s.loyaltyEarnUsdPerPoint))
  }, [settingsRes])

  function saveLoyalty() {
    setLoyaltyError(null)
    setLoyaltySaved(false)
    const rate = Number(earnRate)
    if (!Number.isFinite(rate) || rate <= 0) {
      setLoyaltyError('Earn rate must be greater than zero.')
      return
    }
    update.mutate(
      { loyaltyEnabled, loyaltyEarnUsdPerPoint: rate },
      {
        onSuccess: () => setLoyaltySaved(true),
        onError: (e) =>
          setLoyaltyError(e instanceof ApiError ? e.message : 'Could not save loyalty settings.'),
      },
    )
  }

  return (
    <div className="page">
      <PageHead
        crumbs={[t('grp_system'), t('nav_settings')]}
        title={t('nav_settings')}
        sub="Loyalty program and admin accounts"
      />

      <div className="acard" style={{ marginBottom: 18 }}>
        <div className="panelhead">
          <Icon name="star" size={17} />
          <h3>Loyalty program</h3>
        </div>
        <div className="pad">
          <p className="muted" style={{ fontSize: 12.5, marginTop: 0, marginBottom: 16 }}>
            Customers earn <b>1 point per ${earnRate || '…'}</b> spent on a completed order
            (points = order total ÷ rate, rounded down). Changes apply to future orders only.
          </p>
          <div className="g2">
            <div>
              <label className="alabel">Status</label>
              <select
                className="select"
                style={{ width: '100%' }}
                value={loyaltyEnabled ? 'on' : 'off'}
                onChange={(e) => setLoyaltyEnabled(e.target.value === 'on')}
              >
                <option value="on">Enabled — orders earn points</option>
                <option value="off">Disabled — no points earned</option>
              </select>
            </div>
            <div>
              <label className="alabel">Earn rate — $ spent per point</label>
              <input
                className="afield"
                type="number"
                min="0"
                step="0.5"
                value={earnRate}
                onChange={(e) => setEarnRate(e.target.value)}
              />
            </div>
          </div>
          {loyaltyError && (
            <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 14 }}>
              {loyaltyError}
            </div>
          )}
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 16 }}>
            <button className="abtn primary" onClick={saveLoyalty} disabled={update.isPending || settingsLoading}>
              <Icon name="check" size={15} /> {t('save')}
            </button>
            {loyaltySaved && !update.isPending && (
              <span className="muted" style={{ fontSize: 12.5 }}>Saved.</span>
            )}
          </div>
        </div>
      </div>

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
