import { create } from 'zustand'
import { isRTL } from '@/lib/utils'
import type { Locale } from '@/types'

interface LocaleState {
  locale: Locale
  setLocale(l: Locale): void
}

// Not persisted — same handoff requirement as /web: force English on every load,
// Arabic/Turkish available at runtime via the language switcher.
export const useLocaleStore = create<LocaleState>((set) => ({
  locale: 'en',
  setLocale(l: Locale) {
    set({ locale: l })
    import('@/i18n/config').then((m) => m.default.changeLanguage(l))
    document.documentElement.lang = l
    document.documentElement.dir = isRTL(l) ? 'rtl' : 'ltr'
  },
}))
