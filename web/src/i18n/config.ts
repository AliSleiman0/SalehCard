import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from './locales/en.json'
import ar from './locales/ar.json'
import tr from './locales/tr.json'

i18n.use(initReactI18next).init({
  resources: {
    en: { common: en },
    ar: { common: ar },
    tr: { common: tr },
  },
  ns: ['common'],
  defaultNS: 'common',
  // Handoff: force English default on every load (no localStorage detector override).
  lng: 'en',
  fallbackLng: 'en',
  returnNull: false,
  interpolation: {
    escapeValue: false,
  },
})

export default i18n
