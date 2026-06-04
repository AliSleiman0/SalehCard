import { create } from 'zustand'
import { isRTL } from '@/lib/utils'

type Locale = 'en' | 'ar' | 'tr'

export const LOCALE_NAMES = {
  en: 'English',
  ar: 'العربية',
  tr: 'Türkçe',
} as const

interface LocaleState {
  locale: Locale
  setLocale(l: Locale): void
}

// Not persisted — handoff requirement: English default on every load.
export const useLocaleStore = create<LocaleState>((set) => ({
  locale: 'en',
  setLocale(l: Locale) {
    set({ locale: l })
    // Lazily import i18n to avoid init order issues
    import('@/i18n/config').then((m) => {
      m.default.changeLanguage(l)
    })
    document.documentElement.lang = l
    document.documentElement.dir = isRTL(l) ? 'rtl' : 'ltr'
  },
}))
