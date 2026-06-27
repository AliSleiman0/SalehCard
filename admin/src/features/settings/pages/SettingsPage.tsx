import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Avatar, StatusBadge, Toggle, Tabs, ComingSoonNote } from '@/components'
import { useUsers } from '@/features/users/hooks/useUsers'
import { adaptUser } from '@/features/users/lib/adaptUser'
import { lastActive } from '@/lib/utils'

type Tab = 'general' | 'gateways' | 'notifications' | 'admins'

export default function SettingsPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('general')
  const [visa, setVisa] = useState(true)
  const [usdt, setUsdt] = useState(true)
  const { data: adminsRes, isLoading: adminsLoading } = useUsers({ role: 'admin', limit: 100 })
  const adminRows = (adminsRes?.data ?? []).map(adaptUser)

  return (
    <div className="page">
      <PageHead crumbs={[t('grp_system'), t('nav_settings')]} title={t('nav_settings')} sub="Configure the SalehCard platform">
        <button className="abtn primary">
          <Icon name="check" size={15} /> {t('save')}
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'general', label: 'General' },
          { k: 'gateways', label: 'Payment gateways' },
          { k: 'notifications', label: 'Notifications' },
          { k: 'admins', label: 'Admin accounts' },
        ]}
      />

      {tab === 'general' && (
        <div className="formgrid">
          <div className="fieldset">
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Store details</h3>
              <label className="alabel">Store name</label>
              <input className="afield" defaultValue="SalehCard" />
              <div className="g2" style={{ marginTop: 14 }}>
                <div>
                  <label className="alabel">Support email</label>
                  <input className="afield" defaultValue="support@salehcard.co" />
                </div>
                <div>
                  <label className="alabel">Support phone</label>
                  <input className="afield" defaultValue="+90 552 118 0042" />
                </div>
              </div>
            </div>
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Localization</h3>
              <div className="g2">
                <div>
                  <label className="alabel">Default language</label>
                  <select className="select" style={{ width: '100%' }}>
                    <option>English (EN)</option>
                    <option>العربية (AR)</option>
                    <option>Türkçe (TR)</option>
                  </select>
                </div>
                <div>
                  <label className="alabel">Default currency</label>
                  <select className="select" style={{ width: '100%' }}>
                    <option>USD — US Dollar</option>
                    <option>TRY — Turkish Lira</option>
                  </select>
                </div>
              </div>
              <label className="alabel" style={{ marginTop: 14 }}>
                Enabled languages
              </label>
              <div className="chiprow">
                <div className="chip on">English</div>
                <div className="chip on">العربية · RTL</div>
                <div className="chip on">Türkçe</div>
              </div>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Brand</h3>
            <label className="alabel">Logo</label>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
              <div className="sb-mark" style={{ width: 48, height: 48, borderRadius: 14 }}>
                <svg width="26" height="26" viewBox="0 0 24 24" fill="none">
                  <path d="M16 5H10a3 3 0 0 0 0 6h4a3 3 0 0 1 0 6H7" stroke="#fff" strokeWidth="2.6" strokeLinecap="round" />
                  <circle cx="18.5" cy="6" r="1.7" fill="#22e3c8" />
                </svg>
              </div>
              <button className="abtn sm">
                <Icon name="upload" size={14} /> Replace
              </button>
            </div>
            <label className="alabel">Brand gradient</label>
            <div style={{ height: 40, borderRadius: 'var(--ar-sm)', background: 'var(--grad)', marginBottom: 6 }} />
            <div className="ahint">Electric blue → violet → magenta. Shared with the storefront.</div>
          </div>
        </div>
      )}

      {tab === 'gateways' && (
        <div className="fieldset" style={{ maxWidth: 760 }}>
          <div className="acard pad">
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 11,
                  background: 'linear-gradient(135deg,#3b5bff,#8a3bff)',
                  display: 'grid',
                  placeItems: 'center',
                  color: '#fff',
                }}
              >
                <Icon name="card" size={20} />
              </div>
              <div style={{ flex: 1 }}>
                <b style={{ fontSize: 15 }}>Visa / Mastercard</b>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  Credit &amp; debit card processing
                </div>
              </div>
              <Toggle on={visa} onClick={() => setVisa(!visa)} />
            </div>
            <div className="g2">
              <div>
                <label className="alabel">API key</label>
                <input className="afield mono" type="password" defaultValue="sk_live_8842xxxxxxxxxxxx" />
              </div>
              <div>
                <label className="alabel">Webhook URL</label>
                <input className="afield mono" defaultValue="https://api.salehcard.co/hooks/visa" />
              </div>
            </div>
          </div>
          <div className="acard pad">
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 11,
                  background: 'linear-gradient(135deg,#22e3c8,#26a17b)',
                  display: 'grid',
                  placeItems: 'center',
                  color: '#04121a',
                }}
              >
                <Icon name="coins" size={20} />
              </div>
              <div style={{ flex: 1 }}>
                <b style={{ fontSize: 15 }}>USDT (crypto)</b>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  TRC20 &amp; ERC20 stablecoin payments
                </div>
              </div>
              <Toggle on={usdt} onClick={() => setUsdt(!usdt)} />
            </div>
            <div className="g2">
              <div>
                <label className="alabel">Wallet address (TRC20)</label>
                <input className="afield mono" defaultValue="TJ8x…44Qm" />
              </div>
              <div>
                <label className="alabel">Auto-confirm after</label>
                <select className="select" style={{ width: '100%' }}>
                  <option>1 confirmation</option>
                  <option>Manual review queue</option>
                </select>
              </div>
            </div>
            <div className="ahint">Open item: set to manual review to route top-ups through the USDT queue in Finance.</div>
          </div>
        </div>
      )}

      {tab === 'notifications' && (
        <div className="acard" style={{ maxWidth: 760 }}>
          <div className="panelhead">
            <Icon name="bell" size={17} />
            <h3>Alert thresholds</h3>
          </div>
          <div>
            {(
              [
                ['Low-stock alert', 'Notify when available codes drop below threshold', true],
                ['Critical stock alert', 'Urgent notification when codes hit zero', true],
                ['Large order alert', 'Flag orders above a set amount for review', true],
                ['Failed payment alert', 'Notify on repeated payment failures', false],
                ['New reseller signup', 'Notify admins when an agent applies', true],
                ['Daily revenue digest', 'Email summary at end of day', false],
              ] as [string, string, boolean][]
            ).map(([name, desc, on], i) => (
              <div className="health" key={i} style={{ padding: '14px 18px' }}>
                <div>
                  <b style={{ fontSize: 13.5 }}>{name}</b>
                  <div className="faint" style={{ fontSize: 12.5 }}>
                    {desc}
                  </div>
                </div>
                <div className={'tog' + (on ? ' on' : '')} />
              </div>
            ))}
          </div>
          <div style={{ padding: 18, borderTop: '1px solid var(--border)' }}>
            <div className="g3">
              <div>
                <label className="alabel">Low-stock threshold (default)</label>
                <input className="afield" defaultValue="50" />
              </div>
              <div>
                <label className="alabel">Large order amount ($)</label>
                <input className="afield" defaultValue="500" />
              </div>
              <div>
                <label className="alabel">Failed payment count</label>
                <input className="afield" defaultValue="3" />
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === 'admins' && (
        <div className="acard">
          <div className="panelhead">
            <Icon name="shield" size={17} />
            <h3>Admin accounts</h3>
            <div className="ph-act">
              <button className="abtn xs primary">
                <Icon name="plus" size={13} /> Invite admin
              </button>
            </div>
          </div>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Admin</th>
                  <th>Role</th>
                  <th>Last active</th>
                  <th>Status</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {adminRows.length === 0 && (
                  <tr>
                    <td colSpan={5} className="muted" style={{ textAlign: 'center', padding: 20 }}>
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
                    <td>
                      <div className="row-actions">
                        <span className="iact">
                          <Icon name="edit" size={15} />
                        </span>
                        <span className="iact danger">
                          <Icon name="trash" size={15} />
                        </span>
                      </div>
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
            <Icon name="alert" size={15} /> Open item: granular permission model per role (Super admin / Editor / Viewer)
            is a TODO — roles are display-only for now.
          </div>
        </div>
      )}
    </div>
  )
}
