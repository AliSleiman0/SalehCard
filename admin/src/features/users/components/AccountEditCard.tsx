import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Icon } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { setUserPassword, updateUserProfile } from '../api/users'

const MIN_PASSWORD = 8

/**
 * Lets an admin fix a customer's display name or set them a new password (e.g.
 * a customer who lost access). Setting a password signs the customer out
 * everywhere. Client feedback 2026-11-12 #59.
 */
export function AccountEditCard({ id, name }: { id: string; name: string }) {
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
      toast.success('Name updated')
    },
    onError: (e) => toast.error(e instanceof ApiError ? e.message : "Couldn't update the name"),
  })
  const savePw = useMutation({
    mutationFn: () => setUserPassword(id, pw),
    onSuccess: () => {
      setPw('')
      toast.success('Password set — the user was signed out of all devices')
    },
    onError: (e) => setPwErr(e instanceof ApiError ? e.message : "Couldn't set the password"),
  })

  const nameChanged = nameDraft.trim() !== name.trim() && nameDraft.trim().length > 0
  const onSetPw = () => {
    setPwErr(null)
    if (pw.length < MIN_PASSWORD) {
      setPwErr(`Password must be at least ${MIN_PASSWORD} characters.`)
      return
    }
    savePw.mutate()
  }

  return (
    <div className="acard pad">
      <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Edit account</h3>
      <label className="alabel">Display name</label>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          className="afield"
          aria-label="Display name"
          value={nameDraft}
          maxLength={80}
          onChange={(e) => setNameDraft(e.target.value)}
        />
        <button className="abtn" disabled={!nameChanged || saveName.isPending} onClick={() => saveName.mutate()}>
          <Icon name="check" size={14} /> Save
        </button>
      </div>
      <label className="alabel" style={{ marginTop: 14 }}>
        Set a new password
      </label>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          className="afield"
          type="password"
          autoComplete="new-password"
          aria-label="New password"
          placeholder="At least 8 characters"
          maxLength={72}
          value={pw}
          onChange={(e) => setPw(e.target.value)}
        />
        <button className="abtn" disabled={!pw || savePw.isPending} onClick={onSetPw}>
          <Icon name="shield" size={14} /> Set
        </button>
      </div>
      <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
        The user is signed out of every device and must use the new password.
      </div>
      {pwErr && <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 8 }}>{pwErr}</div>}
    </div>
  )
}
