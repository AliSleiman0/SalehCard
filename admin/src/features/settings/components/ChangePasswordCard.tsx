import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Icon } from '@/components'
import { apiErrorText } from '@/lib/apiErrorText'
import { toast } from '@/stores/toast'
import { changeOwnPassword } from '@/features/users/api/users'

const MIN_PASSWORD = 8

/** The signed-in admin changes their own password (client feedback #41/#96). */
export function ChangePasswordCard() {
  const { t } = useTranslation()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const change = useMutation({
    mutationFn: () => changeOwnPassword(current, next),
    onSuccess: () => {
      setCurrent('')
      setNext('')
      setConfirm('')
      toast.success(t('set_pw_changed'))
    },
    onError: (e) => setErr(apiErrorText(e, 'set_pw_change_failed')),
  })
  const submit = () => {
    setErr(null)
    if (next.length < MIN_PASSWORD) return setErr(t('set_pw_min', { n: MIN_PASSWORD }))
    if (next !== confirm) return setErr(t('set_pw_mismatch'))
    if (next === current) return setErr(t('set_pw_same'))
    change.mutate()
  }
  return (
    <div className="acard" style={{ marginBottom: 18 }}>
      <div className="panelhead">
        <Icon name="id" size={17} />
        <h3>{t('set_my_account')}</h3>
      </div>
      <div className="pad">
        <div className="g3">
          <div>
            <label className="alabel">{t('set_current_pw')}</label>
            <input className="afield" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} />
          </div>
          <div>
            <label className="alabel">{t('usr_new_pw')}</label>
            <input className="afield" type="password" autoComplete="new-password" placeholder={t('usr_pw_ph', { n: MIN_PASSWORD })} maxLength={72} value={next} onChange={(e) => setNext(e.target.value)} />
          </div>
          <div>
            <label className="alabel">{t('set_confirm_pw')}</label>
            <input className="afield" type="password" autoComplete="new-password" maxLength={72} value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          </div>
        </div>
        {err && <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 12 }}>{err}</div>}
        <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 16 }}>
          <button className="abtn primary" onClick={submit} disabled={change.isPending || !current || !next}>
            <Icon name="shield" size={15} /> {t('set_change_pw')}
          </button>
          <span className="muted" style={{ fontSize: 12.5 }}>
            {t('set_pw_2fa_hint')}
          </span>
        </div>
      </div>
    </div>
  )
}
