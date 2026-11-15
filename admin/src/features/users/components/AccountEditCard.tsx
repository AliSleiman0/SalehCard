import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Icon } from '@/components'
import { apiErrorText } from '@/lib/apiErrorText'
import { toast } from '@/stores/toast'
import { setUserPassword, updateUserProfile } from '../api/users'

const MIN_PASSWORD = 8

/**
 * Lets an admin fix a customer's display name or set them a new password (e.g.
 * a customer who lost access). Setting a password signs the customer out
 * everywhere. Client feedback 2026-11-12 #59.
 */
export function AccountEditCard({ id, name }: { id: string; name: string }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [nameDraft, setNameDraft] = useState(name)
  const [pw, setPw] = useState('')
  const [pwErr, setPwErr] = useState<string | null>(null)
  useEffect(() => setNameDraft(name), [name])

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    void qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
  }
  const saveName = useMutation({
    mutationFn: () => updateUserProfile(id, nameDraft.trim()),
    onSuccess: () => {
      refresh()
      toast.success(t('usr_name_updated'))
    },
    onError: (e) => toast.error(apiErrorText(e, 'usr_name_update_failed')),
  })
  const savePw = useMutation({
    mutationFn: () => setUserPassword(id, pw),
    onSuccess: () => {
      setPw('')
      toast.success(t('usr_pw_set'))
    },
    onError: (e) => setPwErr(apiErrorText(e, 'usr_pw_set_failed')),
  })

  const nameChanged = nameDraft.trim() !== name.trim() && nameDraft.trim().length > 0
  const onSetPw = () => {
    setPwErr(null)
    if (pw.length < MIN_PASSWORD) {
      setPwErr(t('usr_pw_min', { n: MIN_PASSWORD }))
      return
    }
    savePw.mutate()
  }

  return (
    <div className="acard pad">
      <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('usr_edit_account')}</h3>
      <label className="alabel">{t('rd_display_name')}</label>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          className="afield"
          aria-label={t('rd_display_name')}
          value={nameDraft}
          maxLength={80}
          onChange={(e) => setNameDraft(e.target.value)}
        />
        <button className="abtn" disabled={!nameChanged || saveName.isPending} onClick={() => saveName.mutate()}>
          <Icon name="check" size={14} /> {t('sup_save')}
        </button>
      </div>
      <label className="alabel" style={{ marginTop: 14 }}>
        {t('usr_set_new_pw')}
      </label>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          className="afield"
          type="password"
          autoComplete="new-password"
          aria-label={t('usr_new_pw')}
          placeholder={t('usr_pw_ph', { n: MIN_PASSWORD })}
          maxLength={72}
          value={pw}
          onChange={(e) => setPw(e.target.value)}
        />
        <button className="abtn" disabled={!pw || savePw.isPending} onClick={onSetPw}>
          <Icon name="shield" size={14} /> {t('usr_set')}
        </button>
      </div>
      <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
        {t('usr_pw_hint')}
      </div>
      {pwErr && <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 8 }}>{pwErr}</div>}
    </div>
  )
}
