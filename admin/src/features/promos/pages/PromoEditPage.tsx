import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { Icon, PageHead, Chip, LoadingSpinner, ErrorState } from '@/components'
import { useGoBack } from '@/lib/goBack'
import { useCan } from '@/stores/auth'
import { serverErrorText } from '@/features/categories/lib/serverErrorText'
import { usePromo, useCreatePromo, useUpdatePromo } from '../hooks/usePromos'
import type { PromoInput, PromoType } from '../api/promos'

// Fixed server sentences sent under a generic code, translated for non-English admins.
const PROMO_SERVER_MESSAGES: Record<string, string> = {
  'a promo with this code already exists': 'promo_srv_conflict',
}

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
  const goBack = useGoBack('/promos')
  const can = useCan()
  const canManage = can('promos.manage')
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
        <PageHead back="/promos" title={t('nav_promos')} />
        <ErrorState message={t('promo_load_one_error')} onRetry={() => refetch()} />
      </div>
    )
  }

  const saving = create.isPending || update.isPending

  const save = () => {
    const v = Number(value)
    const trimmed = code.trim().toUpperCase()
    if (!trimmed) {
      setError(t('promo_err_code'))
      return
    }
    if (!Number.isFinite(v) || v <= 0) {
      setError(t('promo_err_value'))
      return
    }
    if (type === 'percent' && v > 100) {
      setError(t('promo_err_percent'))
      return
    }
    if (startsAt && expiresAt && expiresAt < startsAt) {
      setError(t('promo_err_dates'))
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
    const onError = (e: unknown) => setError(serverErrorText(e, 'exp_save_error', PROMO_SERVER_MESSAGES))
    if (isNew) {
      create.mutate(input, { onSuccess: () => goBack(), onError })
    } else {
      update.mutate(input, { onSuccess: () => goBack(), onError })
    }
  }

  const previewLine = t(`promo_preview_${type}`, { v: value || 0 })

  return (
    <div className="page">
      <PageHead
        back="/promos"
        crumbs={[t('nav_promos'), isNew ? t('promo_new') : code]}
        title={isNew ? t('promo_new_title') : t('promo_edit_title')}
        sub={isNew ? t('promo_new_sub') : code}
      >
        {canManage && (
          <button className="abtn primary" onClick={save} disabled={saving}>
            <Icon name="check" size={15} /> {t('save')}
          </button>
        )}
      </PageHead>

      {error && (
        <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginBottom: 14 }}>{error}</div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('promo_section_code')}</h3>
            <div className="g2">
              <div>
                <label className="alabel">{t('promo_col_code')}</label>
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    className="afield mono"
                    value={code}
                    onChange={(e) => setCode(e.target.value.toUpperCase())}
                    placeholder="SUMMER20"
                    style={{ textTransform: 'uppercase' }}
                  />
                  <button className="abtn sm" title={t('promo_generate')} onClick={() => setCode(randomCode())}>
                    <Icon name="refresh" size={14} />
                  </button>
                </div>
              </div>
              <div>
                <label className="alabel">{t('status')}</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={active ? 'active' : 'paused'}
                  onChange={(e) => setActive(e.target.value === 'active')}
                >
                  <option value="active">{t('promo_status_active')}</option>
                  <option value="paused">{t('promo_status_paused')}</option>
                </select>
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 16 }}>
              {t('promo_discount_type')}
            </label>
            <div className="g3">
              {(
                [
                  ['percent', t('promo_chip_percent')],
                  ['fixed', t('promo_chip_fixed')],
                  ['cashback', t('promo_type_cashback')],
                ] as [PromoType, string][]
              ).map(([k, l]) => (
                <Chip key={k} on={type === k} onClick={() => setType(k)} style={{ justifyContent: 'center', padding: 12 }}>
                  {l}
                </Chip>
              ))}
            </div>
            <div className="g2" style={{ marginTop: 16 }}>
              <div>
                <label className="alabel">
                  {t('promo_col_value')} {type === 'percent' ? '(%)' : '($)'}
                </label>
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
                <label className="alabel">{t('promo_min_order')}</label>
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
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('promo_limits')}</h3>
            <label className="alabel">{t('promo_max_uses')}</label>
            <input
              className="afield"
              type="number"
              min="0"
              step="1"
              value={maxUses}
              onChange={(e) => setMaxUses(e.target.value)}
            />
            <div className="ahint" style={{ marginTop: 8 }}>
              {t('promo_coming_soon')}
            </div>
          </div>
        </div>
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('promo_validity')}</h3>
            <label className="alabel">{t('promo_start')}</label>
            <input className="afield" type="date" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} />
            <label className="alabel" style={{ marginTop: 12 }}>
              {t('promo_end')}
            </label>
            <input className="afield" type="date" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
            <div className="ahint" style={{ marginTop: 8 }}>
              {t('promo_no_bound')}
            </div>
          </div>
          <div className="acard pad" style={{ textAlign: 'center' }}>
            <div className="alabel" style={{ textAlign: 'start' }}>
              {t('promo_preview')}
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
              <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>{previewLine}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
