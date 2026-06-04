import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Chip, ComingSoonNote } from '@/components'
import { promos, CATS } from '@/lib/mock/demo'

type PromoType = 'percent' | 'fixed' | 'cashback'

export default function PromoEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isNew = !id
  // TODO: wire to GET/POST/PUT /api/admin/promos (mock; backend route stubbed 501).
  const p = isNew ? { code: '', type: 'percent' as PromoType, value: 10, status: 'active' } : promos.find((x) => x.code === id) || promos[0]
  const [type, setType] = useState<PromoType>(p.type)

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_promos'), isNew ? 'New promo' : p.code]}
        title={isNew ? 'New promo code' : 'Edit promo'}
        sub={isNew ? 'Create a discount or cashback campaign' : p.code}
      >
        <button className="abtn" onClick={() => navigate('/promos')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary">
          <Icon name="check" size={15} /> {t('save')}
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Code &amp; value</h3>
            <div className="g2">
              <div>
                <label className="alabel">Code</label>
                <div style={{ display: 'flex', gap: 8 }}>
                  <input className="afield mono" defaultValue={p.code} placeholder="SUMMER20" style={{ textTransform: 'uppercase' }} />
                  <button className="abtn sm">
                    <Icon name="refresh" size={14} />
                  </button>
                </div>
              </div>
              <div>
                <label className="alabel">Status</label>
                <select className="select" style={{ width: '100%' }} defaultValue={p.status}>
                  <option value="active">Active</option>
                  <option value="paused">Paused</option>
                  <option value="draft">Draft</option>
                </select>
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 16 }}>
              Discount type
            </label>
            <div className="g3">
              {(
                [
                  ['percent', 'Percent %'],
                  ['fixed', 'Fixed $'],
                  ['cashback', 'Cashback'],
                ] as [PromoType, string][]
              ).map(([k, l]) => (
                <Chip key={k} on={type === k} onClick={() => setType(k)} style={{ justifyContent: 'center', padding: 12 }}>
                  {l}
                </Chip>
              ))}
            </div>
            <div className="g2" style={{ marginTop: 16 }}>
              <div>
                <label className="alabel">Value {type === 'fixed' ? '($)' : '(%)'}</label>
                <input className="afield" defaultValue={p.value} />
              </div>
              <div>
                <label className="alabel">Min order amount ($)</label>
                <input className="afield" defaultValue="0" />
              </div>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Limits &amp; eligibility</h3>
            <div className="g2">
              <div>
                <label className="alabel">Max total uses</label>
                <input className="afield" defaultValue="5000" />
              </div>
              <div>
                <label className="alabel">Per-user limit</label>
                <input className="afield" defaultValue="1" />
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 16 }}>
              Applicable categories
            </label>
            <div className="chiprow">
              {CATS.map((c, i) => (
                <div key={c} className={'chip' + (i < 2 ? ' on' : '')}>
                  {c}
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Validity</h3>
            <label className="alabel">Start date</label>
            <input className="afield" type="text" defaultValue="2026-06-01" />
            <label className="alabel" style={{ marginTop: 12 }}>
              End date
            </label>
            <input className="afield" type="text" defaultValue="2026-06-30" />
          </div>
          <div className="acard pad" style={{ textAlign: 'center' }}>
            <div className="alabel" style={{ textAlign: 'start' }}>
              Customer preview
            </div>
            <div
              style={{
                padding: 20,
                background: 'var(--grad-soft)',
                borderRadius: 'var(--ar-md)',
                border: '1.5px dashed var(--ff-code-bd)',
              }}
            >
              <div className="mono" style={{ fontSize: 22, fontWeight: 800, letterSpacing: '.08em' }}>
                {p.code || 'SUMMER20'}
              </div>
              <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>
                {type === 'fixed' ? '$' + p.value + ' off' : type === 'cashback' ? p.value + '% cashback' : p.value + '% off'} your
                order
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
