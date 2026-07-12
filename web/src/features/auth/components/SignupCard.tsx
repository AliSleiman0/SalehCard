import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button, Input } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useRegister } from '@/features/auth/hooks/useRegister'
import { useRequestOtp } from '@/features/auth/hooks/useRequestOtp'
import { useVerifyOtp } from '@/features/auth/hooks/useVerifyOtp'
import { PhoneField } from './PhoneField'
import { OtpInput } from './OtpInput'
import { toE164Lebanon, isPlausibleLebanonPhone } from '@/features/auth/lib/phone'

function StepDots({ step, total }: { step: number; total: number }) {
  return (
    <div className="row center" style={{ gap: 8, marginBottom: 4 }}>
      {Array.from({ length: total }).map((_, i) => (
        <span
          key={i}
          style={{
            width: i === step ? 22 : 8,
            height: 8,
            borderRadius: 99,
            background: i === step ? 'var(--brand-1)' : 'var(--border-strong)',
            transition: 'width .2s',
          }}
        />
      ))}
    </div>
  )
}

// SignupCard is the registration form body. Phone signup is a 3-step wizard
// (name+phone → password → OTP verify, which creates the account); email signup
// keeps the classic single form.
export function SignupCard({ tab }: { tab: 'phone' | 'email' }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const locale = useLocaleStore((s) => s.locale)

  const register = useRegister()
  const requestOtp = useRequestOtp()
  const verifyOtp = useVerifyOtp()

  const [step, setStep] = useState(0)
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [code, setCode] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)

  const onDone = () => navigate('/dashboard')
  const fail = (err: Error) => setError(err.message || t('auth_failed'))
  const pending = register.isPending || requestOtp.isPending || verifyOtp.isPending

  // ---- email tab ----
  if (tab === 'email') {
    const submit = (e: FormEvent) => {
      e.preventDefault()
      setError(null)
      if (!name.trim()) return setError(t('enter_name'))
      if (!email.includes('@')) return setError(t('invalid_email'))
      if (password.length < 8) return setError(t('password_too_short'))
      register.mutate(
        { name: name.trim(), email, password, locale },
        { onSuccess: onDone, onError: fail },
      )
    }
    return (
      <form className="col" style={{ gap: 14 }} onSubmit={submit}>
        <Input label={t('full_name')} value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" />
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
          autoComplete="new-password"
          error={error ?? undefined}
        />
        <Button variant="primary" size="lg" block type="submit" loading={pending}>
          {t('register')}
        </Button>
      </form>
    )
  }

  // ---- phone wizard ----
  const next0 = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!name.trim()) return setError(t('enter_name'))
    if (!isPlausibleLebanonPhone(phone)) return setError(t('invalid_phone'))
    setStep(1)
  }

  const next1 = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (password.length < 8) return setError(t('password_too_short'))
    if (password !== confirm) return setError(t('passwords_no_match'))
    requestOtp.mutate(
      { phone: toE164Lebanon(phone) },
      { onSuccess: () => setStep(2), onError: fail },
    )
  }

  const finish = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (code.length < 6) return setError(t('enter_code'))
    verifyOtp.mutate(
      { phone: toE164Lebanon(phone), code, password, name: name.trim() },
      { onSuccess: onDone, onError: fail },
    )
  }

  return (
    <div className="col" style={{ gap: 16 }}>
      <StepDots step={step} total={3} />
      {step === 0 && (
        <form className="col" style={{ gap: 14 }} onSubmit={next0}>
          <Input
            label={t('full_name')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoComplete="name"
            autoFocus
          />
          <PhoneField label={t('phone')} value={phone} onChange={setPhone} error={error ?? undefined} />
          <Button variant="primary" size="lg" block type="submit">
            {t('next')}
          </Button>
        </form>
      )}
      {step === 1 && (
        <form className="col" style={{ gap: 14 }} onSubmit={next1}>
          <Input
            label={t('create_password')}
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            autoFocus
          />
          <Input
            label={t('confirm_password')}
            type="password"
            placeholder="••••••••"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            error={error ?? undefined}
          />
          <Button variant="primary" size="lg" block type="submit" loading={pending}>
            {t('send_code')}
          </Button>
          <a
            className="small clickable center"
            style={{ fontWeight: 700 }}
            onClick={() => {
              setStep(0)
              setError(null)
            }}
          >
            {t('back')}
          </a>
        </form>
      )}
      {step === 2 && (
        <form className="col" style={{ gap: 16 }} onSubmit={finish}>
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
                setStep(1)
                setCode('')
                setError(null)
              }}
            >
              {t('back')}
            </a>
            <a
              className="small clickable"
              style={{ fontWeight: 700, color: 'var(--brand-1)' }}
              onClick={() =>
                requestOtp.mutate({ phone: toE164Lebanon(phone) }, { onError: fail })
              }
            >
              {t('resend_code')}
            </a>
          </div>
        </form>
      )}
    </div>
  )
}
