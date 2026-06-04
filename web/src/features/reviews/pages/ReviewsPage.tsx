import { useTranslation } from 'react-i18next'
import { Panel } from '@/components'

// TODO: standalone reviews feature — wire to review module API when it lands
export default function ReviewsPage() {
  const { t } = useTranslation()

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <Panel className="center" style={{ textAlign: 'center' }}>
        <h2 className="h2">{t('reviews', { defaultValue: 'Reviews' })}</h2>
        <p className="muted" style={{ marginTop: 8 }}>
          Verified reviews appear on each product page.
        </p>
      </Panel>
    </div>
  )
}
