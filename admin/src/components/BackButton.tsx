import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { hasInAppHistory, useBackTarget, useGoBack } from '@/lib/goBack'
import { Icon } from './Icon'

/**
 * The page-header "‹ Back" button: previous in-app page, else `fallback` (the
 * logical parent route). Rendered by <PageHead back="…"> so it sits in the same
 * slot on every page — pass `back` there rather than using this directly.
 */
export function BackButton({ fallback }: { fallback: string }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const target = useBackTarget(fallback)
  const goBack = useGoBack(fallback)
  // Opened directly on the page the fallback points at (e.g. a role whose first
  // permitted page is this one): Back would be a no-op, so don't offer it.
  if (!hasInAppHistory() && target === pathname) return null
  return (
    <button type="button" className="abtn back-btn" onClick={goBack}>
      <Icon name="chevleft" size={15} /> {t('back')}
    </button>
  )
}
