import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Icon } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { changeOwnPassword } from '@/features/users/api/users'

const MIN_PASSWORD = 8

/** The signed-in admin changes their own password (client feedback #41/#96). */
export function ChangePasswordCard() {
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
      toast.success('Your password was changed')
    },
    onError: (e) => setErr(e instanceof ApiError ? e.message : "Couldn't change the password"),
  })
  const submit = () => {
    setErr(null)
    if (next.length < MIN_PASSWORD) return setErr(`New password must be at least ${MIN_PASSWORD} characters.`)
    if (next !== confirm) return setErr('The new passwords do not match.')
    if (next === current) return setErr('Choose a password different from the current one.')
    change.mutate()
  }
  return (
    <div className="acard" style={{ marginBottom: 18 }}>
      <div className="panelhead">
        <Icon name="id" size={17} />
        <h3>My account</h3>
      </div>
      <div className="pad">
        <div className="g3">
          <div>
            <label className="alabel">Current password</label>
            <input className="afield" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} />
          </div>
          <div>
            <label className="alabel">New password</label>
            <input className="afield" type="password" autoComplete="new-password" placeholder="At least 8 characters" maxLength={72} value={next} onChange={(e) => setNext(e.target.value)} />
          </div>
          <div>
            <label className="alabel">Confirm new password</label>
            <input className="afield" type="password" autoComplete="new-password" maxLength={72} value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          </div>
        </div>
        {err && <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 12 }}>{err}</div>}
        <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 16 }}>
          <button className="abtn primary" onClick={submit} disabled={change.isPending || !current || !next}>
            <Icon name="shield" size={15} /> Change password
          </button>
          <span className="muted" style={{ fontSize: 12.5 }}>
            For a second factor, turn on SMS two-factor under Security below.
          </span>
        </div>
      </div>
    </div>
  )
}
