import { useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Button, useToast } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { useChangePassword } from '@/features/auth/hooks/useChangePassword'
import { changePasswordErrorKey, validateChangePassword } from '@/features/auth/lib/changePassword'

// PasswordField is a labelled password input with a show/hide toggle. The
// toggle sits on the inline-end edge so it flips with RTL.
function PasswordField({
  label,
  value,
  onChange,
  autoComplete,
  hint,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  autoComplete: string
  hint?: string
}) {
  const { t } = useTranslation()
  const id = useId()
  const [shown, setShown] = useState(false)
  return (
    <div className="col" style={{ gap: 6 }}>
      <label className="label" htmlFor={id}>
        {label}
      </label>
      <div style={{ position: 'relative' }}>
        <input
          id={id}
          className="field"
          type={shown ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          autoComplete={autoComplete}
          dir="auto"
          style={{ width: '100%', paddingInlineEnd: 46 }}
        />
        <button
          type="button"
          onClick={() => setShown((s) => !s)}
          aria-label={shown ? t('pw_hide') : t('pw_show')}
          aria-pressed={shown}
          title={shown ? t('pw_hide') : t('pw_show')}
          style={{
            position: 'absolute',
            insetInlineEnd: 6,
            top: '50%',
            transform: 'translateY(-50%)',
            background: 'transparent',
            border: 0,
            padding: 6,
            cursor: 'pointer',
            color: 'var(--text-dim)',
            display: 'grid',
            placeItems: 'center',
          }}
        >
          <Icon name={shown ? 'eyeoff' : 'eye'} size={18} />
        </button>
      </div>
      {hint && <span className="tiny faint">{hint}</span>}
    </div>
  )
}

// ChangePasswordForm: current + new + confirm, validated client-side (min 8
// characters, max 72 bytes, confirmation matches) and re-checked by the API.
// Other sessions stay signed in, so the UI never promises otherwise.
function ChangePasswordForm() {
  const { t } = useTranslation()
  const toast = useToast()
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [noPassword, setNoPassword] = useState(false)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (change.isPending) return
    const problem = validateChangePassword({ current, next, confirm })
    if (problem) {
      setError(t(problem))
      return
    }
    setError(null)
    setNoPassword(false)
    change.mutate(
      { currentPassword: current, newPassword: next },
      {
        onSuccess: () => {
          toast(t('pw_changed'), 'check')
          setCurrent('')
          setNext('')
          setConfirm('')
        },
        onError: (err) => {
          const key = changePasswordErrorKey(err.code)
          setNoPassword(err.code === 'NO_PASSWORD_SET')
          setError(key ? t(key) : err.message || t('pw_err_generic'))
        },
      },
    )
  }

  return (
    <form className="panel card-pad col" style={{ gap: 16, maxWidth: 520 }} onSubmit={submit} noValidate>
      <div className="col" style={{ gap: 4 }}>
        <h3 className="h3" style={{ margin: 0 }}>
          {t('change_password')}
        </h3>
        <p className="small muted" style={{ margin: 0 }}>
          {t('pw_intro')}
        </p>
      </div>
      <PasswordField
        label={t('pw_current')}
        value={current}
        onChange={setCurrent}
        autoComplete="current-password"
      />
      <PasswordField
        label={t('pw_new')}
        value={next}
        onChange={setNext}
        autoComplete="new-password"
        hint={t('pw_rules')}
      />
      <PasswordField label={t('pw_confirm')} value={confirm} onChange={setConfirm} autoComplete="new-password" />
      {error && (
        <div className="col" style={{ gap: 4 }} role="alert">
          <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
            {error}
          </span>
          {noPassword && <span className="tiny faint">{t('pw_no_password_hint')}</span>}
        </div>
      )}
      <div>
        <Button type="submit" variant="primary" size="lg" loading={change.isPending}>
          {t('change_password')}
        </Button>
      </div>
    </form>
  )
}

export default function SecurityPage() {
  const { t } = useTranslation()
  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="row between" style={{ marginBottom: 24 }}>
        <h1 className="h1">{t('security')}</h1>
      </div>
      <div className="cols-acct">
        <AcctSidebar active="security" />
        <div className="col" style={{ gap: 16 }}>
          <ChangePasswordForm />
        </div>
      </div>
    </div>
  )
}
