import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Chip, LoadingSpinner, ErrorState } from '@/components'
import { ApiError } from '@/lib/api-client'
import { usePromo, useCreatePromo, useUpdatePromo } from '../hooks/usePromos'
import type { PromoInput, PromoType } from '../api/promos'

// dateInput converts an ISO timestamp to a yyyy-mm-dd value for <input type=date>.
function dateInput(iso?: string): string {
  return iso ? iso.slice(0, 10) : ''
}

// randomCode generates a short uppercase code for the "generate" button.
function randomCode(): string {
  const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let s = ''
  for (let i = 0; i < 6; i++) s += alphabet[Math.floor(Math.random() * alphabet.length)]
  return 'SAVE' + s
}

export default function PromoEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isNew = !id

  const { data, isLoading, isError, refetch } = usePromo(id)
  const create = useCreatePromo()
  const update = useUpdatePromo(id ?? '')

  const [code, setCode] = useState('')
  const [type, setType] = useState<PromoType>('percent')
  const [value, setValue] = useState('10')
  const [minOrder, setMinOrder] = useState('0')
  const [maxUses, setMaxUses] = useState('0')
  const [startsAt, setStartsAt] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [active, setActive] = useState(true)
  const [error, setError] = useState('')

  // Populate the form once the existing promo loads (edit mode).
  useEffect(() => {
    const p = data?.data
    if (!p) return
    setCode(p.code)
    setType(p.type)
    setValue(String(p.value))
    setMinOrder(String(p.minOrder))
    setMaxUses(String(p.maxUses))
    setStartsAt(dateInput(p.startsAt))
    setExpiresAt(dateInput(p.expiresAt))
    setActive(p.active)
  }, [data])

  if (!isNew && isLoading) return <LoadingSpinner />
  if (!isNew && (isError || !data?.data)) {
    return (
      <div className="page">
        <ErrorState message="Couldn't load this promo." onRetry={() => refetch()} />
      </div>
    )
  }

  const saving = create.isPending || update.isPending

  const save = () => {
    const v = Number(value)
    const trimmed = code.trim().toUpperCase()
    if (!trimmed) {
      setError('Code is required.')
      return
    }
    if (!Number.isFinite(v) || v <= 0) {
      setError('Value must be greater than zero.')
      return
    }
    if (type === 'percent' && v > 100) {
      setError('Percent value must be between 0 and 100.')
      return
    }
    if (startsAt && expiresAt && expiresAt < startsAt) {
      setError('End date must be after the start date.')
      return
    }
    setError('')
    const input: PromoInput = {
      code: trimmed,
      type,
      value: v,
      minOrder: Number(minOrder) || 0,
      maxUses: Number(maxUses) || 0,
      startsAt: startsAt ? new Date(startsAt).toISOString() : null,
      expiresAt: expiresAt ? new Date(expiresAt).toISOString() : null,
      active,
    }
    const onError = (e: unknown) => setError(e instanceof ApiError ? e.message : 'Save failed.')
    if (isNew) {
      create.mutate(input, { onSuccess: () => navigate('/promos'), onError })
    } else {
      update.mutate(input, { onSuccess: () => navigate('/promos'), onError })
    }
  }

  const previewLine =
    type === 'fixed' ? `$${value || 0} off` : type === 'cashback' ? `$${value || 0} cashback` : `${value || 0}% off`

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_promos'), isNew ? 'New promo' : code]}
        title={isNew ? 'New promo code' : 'Edit promo'}
        sub={isNew ? 'Create a discount or cashback campaign' : code}
      >
        <button className="abtn" onClick={() => navigate('/promos')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary" onClick={save} disabled={saving}>
          <Icon name="check" size={15} /> {t('save')}
        </button>
      </PageHead>

      {error && (
        <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginBottom: 14 }}>{error}</div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Code &amp; value</h3>
            <div className="g2">
              <div>
                <label className="alabel">Code</label>
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    className="afield mono"
                    value={code}
                    onChange={(e) => setCode(e.target.value.toUpperCase())}
                    placeholder="SUMMER20"
                    style={{ textTransform: 'uppercase' }}
                  />
                  <button className="abtn sm" title="Generate" onClick={() => setCode(randomCode())}>
                    <Icon name="refresh" size={14} />
                  </button>
                </div>
              </div>
              <div>
                <label className="alabel">Status</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={active ? 'active' : 'paused'}
                  onChange={(e) => setActive(e.target.value === 'active')}
                >
                  <option value="active">Active</option>
                  <option value="paused">Paused</option>
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
                <label className="alabel">Value {type === 'percent' ? '(%)' : '($)'}</label>
                <input
                  className="afield"
                  type="number"
                  min="0"
                  step="0.01"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                />
              </div>
              <div>
                <label className="alabel">Min order amount ($)</label>
                <input
                  className="afield"
                  type="number"
                  min="0"
                  step="0.01"
                  value={minOrder}
                  onChange={(e) => setMinOrder(e.target.value)}
                />
              </div>
            </div>
          </div>
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Limits</h3>
            <label className="alabel">Max total uses (0 = unlimited)</label>
            <input
              className="afield"
              type="number"
              min="0"
              step="1"
              value={maxUses}
              onChange={(e) => setMaxUses(e.target.value)}
            />
            <div className="ahint" style={{ marginTop: 8 }}>
              Per-user limits and category eligibility are coming soon.
            </div>
          </div>
        </div>
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Validity</h3>
            <label className="alabel">Start date</label>
            <input className="afield" type="date" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} />
            <label className="alabel" style={{ marginTop: 12 }}>
              End date
            </label>
            <input className="afield" type="date" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
            <div className="ahint" style={{ marginTop: 8 }}>
              Leave a date empty for no bound.
            </div>
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
                {code || 'SUMMER20'}
              </div>
              <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>{previewLine} your order</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
