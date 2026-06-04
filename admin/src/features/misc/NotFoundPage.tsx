import { useNavigate } from 'react-router-dom'
import { Icon, PageHead, EmptyState } from '@/components'

export default function NotFoundPage() {
  const navigate = useNavigate()
  return (
    <div className="page">
      <PageHead title="Page not found" sub="The screen you're looking for doesn't exist." />
      <div className="acard pad">
        <EmptyState icon="search" title="404 — Not found" sub="Head back to the dashboard." />
        <div style={{ display: 'flex', justifyContent: 'center', marginTop: 12 }}>
          <button className="abtn primary" onClick={() => navigate('/')}>
            <Icon name="grid" size={15} /> Go to dashboard
          </button>
        </div>
      </div>
    </div>
  )
}
