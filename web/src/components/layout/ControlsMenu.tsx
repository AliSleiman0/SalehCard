import { useTranslation } from 'react-i18next'
import { Icon } from '@/components'
import { useLocaleStore, LOCALE_NAMES } from '@/stores/locale'
import { useCurrencyStore } from '@/stores/currency'
import { useThemeStore } from '@/stores/theme'
import { SYM, type Currency } from '@/lib/utils'

const LOCALE_KEYS = Object.keys(LOCALE_NAMES) as (keyof typeof LOCALE_NAMES)[]
const CURRENCIES: Currency[] = ['USD', 'TRY']

export function ControlsMenu() {
  const { t } = useTranslation()
  const { locale, setLocale } = useLocaleStore()
  const { currency, setCurrency } = useCurrencyStore()
  const { theme, setTheme } = useThemeStore()

  return (
    <div className="pop" onClick={(e) => e.stopPropagation()} style={{ minWidth: 210 }}>
      <div className="eyebrow" style={{ padding: '6px 12px' }}>
        {t('language')}
      </div>
      {LOCALE_KEYS.map((k) => (
        <button key={k} className={locale === k ? 'on' : ''} onClick={() => setLocale(k)}>
          <Icon name="globe" size={16} /> {LOCALE_NAMES[k]}
          {locale === k && <span className="spacer" />}
          {locale === k && <Icon name="check" size={15} />}
        </button>
      ))}

      <hr className="divider" style={{ margin: '6px 0' }} />

      <div className="eyebrow" style={{ padding: '6px 12px' }}>
        {t('currency')}
      </div>
      <div className="row" style={{ gap: 6, padding: 6 }}>
        {CURRENCIES.map((c) => (
          <button
            key={c}
            className={'btn btn-sm ' + (currency === c ? 'btn-primary' : 'btn-ghost')}
            style={{ flex: 1 }}
            onClick={() => setCurrency(c)}
          >
            {SYM[c]} {c}
          </button>
        ))}
      </div>

      <hr className="divider" style={{ margin: '6px 0' }} />

      <div className="row" style={{ gap: 6, padding: 6 }}>
        <button
          className={'btn btn-sm ' + (theme === 'light' ? 'btn-primary' : 'btn-ghost')}
          style={{ flex: 1 }}
          onClick={() => setTheme('light')}
        >
          <Icon name="sun" size={15} /> Light
        </button>
        <button
          className={'btn btn-sm ' + (theme === 'dark' ? 'btn-primary' : 'btn-ghost')}
          style={{ flex: 1 }}
          onClick={() => setTheme('dark')}
        >
          <Icon name="moon" size={15} /> Dark
        </button>
      </div>
    </div>
  )
}
