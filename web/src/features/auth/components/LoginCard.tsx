import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button, Input } from '@/components'
import { useLogin } from '@/features/auth/hooks/useLogin'
import { useLoginPhone } from '@/features/auth/hooks/useLoginPhone'
import { useRequestOtp } from '@/features/auth/hooks/useRequestOtp'
import { useVerifyOtp } from '@/features/auth/hooks/useVerifyOtp'
import { PhoneField } from './PhoneField'
import { OtpInput } from './OtpInput'
import { toE164Lebanon, isPlausibleLebanonPhone } from '@/features/auth/lib/phone'

// LoginCard is the sign-in form body. `tab` selects phone (default) vs email;
// the phone tab further toggles between a one-time code and phone+password.
export function LoginCard({ tab }: { tab: 'phone' | 'email' }) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const login = useLogin()
  const loginPhone = useLoginPhone()
  const requestOtp = useRequestOtp()
  const verifyOtp = useVerifyOtp()

  const [mode, setMode] = useState<'code' | 'password'>('code')
  const [codeSent, setCodeSent] = useState(false)
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const onDone = () => navigate('/dashboard')
  const fail = (err: Error) => setError(err.message || t('auth_failed'))
  const pending =
    login.isPending || loginPhone.isPending || requestOtp.isPending || verifyOtp.isPending

  // ---- email tab ----
  if (tab === 'email') {
    const submit = (e: FormEvent) => {
      e.preventDefault()
      setError(null)
      if (!email.includes('@')) return setError(t('invalid_email'))
      if (password.length < 8) return setError(t('password_too_short'))
      login.mutate({ email, password }, { onSuccess: onDone, onError: fail })
    }
    return (
      <form className="col" style={{ gap: 14 }} onSubmit={submit}>
        <Input
          label={t('email')}
          type="email"
          placeholder="you@email.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <Input
          label={t('password')}
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          error={error ?? undefined}
        />
        <Button variant="primary" size="lg" block type="submit" loading={pending}>
          {t('login')}
        </Button>
      </form>
    )
  }

  // ---- phone tab: password sub-mode ----
  if (mode === 'password') {
    const submit = (e: FormEvent) => {
      e.preventDefault()
      setError(null)
      if (!isPlausibleLebanonPhone(phone)) return setError(t('invalid_phone'))
      if (password.length < 8) return setError(t('password_too_short'))
      loginPhone.mutate(
        { phone: toE164Lebanon(phone), password },
        { onSuccess: onDone, onError: fail },
      )
    }
    return (
      <form className="col" style={{ gap: 14 }} onSubmit={submit}>
        <PhoneField label={t('phone')} value={phone} onChange={setPhone} autoFocus />
        <Input
          label={t('password')}
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          error={error ?? undefined}
        />
        <Button variant="primary" size="lg" block type="submit" loading={pending}>
          {t('login')}
        </Button>
        <a
          className="small clickable center"
          style={{ fontWeight: 700, color: 'var(--brand-1)' }}
          onClick={() => {
            setMode('code')
            setError(null)
          }}
        >
          {t('use_code_instead')}
        </a>
      </form>
    )
  }

  // ---- phone tab: code sub-mode ----
  const sendCode = (e?: FormEvent) => {
    e?.preventDefault()
    setError(null)
    if (!isPlausibleLebanonPhone(phone)) return setError(t('invalid_phone'))
    requestOtp.mutate(
      { phone: toE164Lebanon(phone) },
      { onSuccess: () => setCodeSent(true), onError: fail },
    )
  }

  const verify = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (code.length < 6) return setError(t('enter_code'))
    verifyOtp.mutate(
      { phone: toE164Lebanon(phone), code },
      { onSuccess: onDone, onError: fail },
    )
  }

  if (!codeSent) {
    return (
      <form className="col" style={{ gap: 14 }} onSubmit={sendCode}>
        <PhoneField
          label={t('phone')}
          value={phone}
          onChange={setPhone}
          error={error ?? undefined}
          autoFocus
        />
        <Button variant="primary" size="lg" block type="submit" loading={pending}>
          {t('send_code')}
        </Button>
        <a
          className="small clickable center"
          style={{ fontWeight: 700, color: 'var(--brand-1)' }}
          onClick={() => {
            setMode('password')
            setError(null)
          }}
        >
          {t('use_password_instead')}
        </a>
      </form>
    )
  }

  return (
    <form className="col" style={{ gap: 16 }} onSubmit={verify}>
      <p className="small muted center" style={{ margin: 0 }}>
        {t('code_sent_to')} <b>+961 {phone}</b>
      </p>
      <OtpInput value={code} onChange={setCode} hasError={!!error} />
      {error && (
        <span className="tiny center" style={{ color: 'var(--danger)', fontWeight: 600 }}>
          {error}
        </span>
      )}
      <Button variant="primary" size="lg" block type="submit" loading={pending}>
        {t('verify')}
      </Button>
      <div className="row center" style={{ gap: 14 }}>
        <a
          className="small clickable"
          style={{ fontWeight: 700 }}
          onClick={() => {
            setCodeSent(false)
            setCode('')
            setError(null)
          }}
        >
          {t('back')}
        </a>
        <a
          className="small clickable"
          style={{ fontWeight: 700, color: 'var(--brand-1)' }}
          onClick={() => sendCode()}
        >
          {t('resend_code')}
        </a>
      </div>
    </form>
  )
}
