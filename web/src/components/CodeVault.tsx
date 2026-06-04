import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon } from './Icon'
import { Button } from './Button'
import { useToast } from './Toast'

export function CodeVault({
  label,
  code,
  pin,
}: {
  label?: string
  code: string
  pin?: string
}) {
  const { t } = useTranslation()
  const [shown, setShown] = useState(false)
  const toast = useToast()

  const copy = (val: string) => {
    navigator.clipboard?.writeText(val).catch(() => {})
    toast(t('copied'), 'copy')
  }

  return (
    <div className="col" style={{ gap: 10 }}>
      {label && (
        <div className="row between">
          <span className="small" style={{ fontWeight: 700 }}>
            {label}
          </span>
          <span className="badge badge-secure">
            <Icon name="shield" size={12} />
            {t('secure_vault')}
          </span>
        </div>
      )}
      <div className={`vault ${shown ? '' : 'masked'}`}>
        <span className="code">{code}</span>
        <div className="row" style={{ gap: 8 }}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShown((s) => !s)}
            style={{ padding: '9px 12px' }}
          >
            <Icon name={shown ? 'eyeoff' : 'eye'} size={16} />
            {shown ? t('hide') : t('reveal')}
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={() => copy(code)}
            style={{ padding: '9px 14px' }}
          >
            <Icon name="copy" size={16} />
            {t('copy')}
          </Button>
        </div>
      </div>
      {pin && (
        <div className="vault" style={{ fontSize: 15 }}>
          <span className="muted small" style={{ letterSpacing: 0 }}>
            {t('pin')}
          </span>
          <span className="row" style={{ gap: 10 }}>
            <span className="code" style={{ filter: shown ? 'none' : 'blur(6px)' }}>
              {pin}
            </span>
            <Button variant="ghost" size="sm" onClick={() => copy(pin)} style={{ padding: 9 }}>
              <Icon name="copy" size={15} />
            </Button>
          </span>
        </div>
      )}
    </div>
  )
}
