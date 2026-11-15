import i18n, { type BackendModule } from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from './locales/en.json'

/** Arabic and Turkish load on first use, so English sign-ins don't download them. */
const lazyLocales: BackendModule = {
  type: 'backend',
  init() {},
  read(lng, _ns, done) {
    const load = lng === 'ar' ? import('./locales/ar.json') : lng === 'tr' ? import('./locales/tr.json') : null
    if (!load) return done(null, {})
    load.then((m) => done(null, m.default)).catch((e) => done(e, null))
  },
}

i18n.use(lazyLocales).use(initReactI18next).init({
  resources: { en: { common: en } },
  partialBundledLanguages: true,
  ns: ['common'],
  defaultNS: 'common',
  // Handoff: force English on every load (same as /web), no detector override.
  lng: 'en',
  fallbackLng: 'en',
  returnNull: false,
  interpolation: { escapeValue: false },
})

export default i18n
