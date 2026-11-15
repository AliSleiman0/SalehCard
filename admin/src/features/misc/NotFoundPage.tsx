import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, PageHead, EmptyState } from '@/components'

export default function NotFoundPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  return (
    <div className="page">
      <PageHead back="/" title={t('cmn_nf_title')} sub={t('cmn_nf_sub')} />
      <div className="acard pad">
        <EmptyState icon="search" title={t('cmn_nf_empty')} sub={t('cmn_nf_empty_sub')} />
        <div style={{ display: 'flex', justifyContent: 'center', marginTop: 12 }}>
          <button className="abtn primary" onClick={() => navigate('/')}>
            <Icon name="grid" size={15} /> {t('cmn_nf_go')}
          </button>
        </div>
      </div>
    </div>
  )
}
