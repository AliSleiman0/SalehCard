import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Button, useToast } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { SavedIdCard } from '@/features/auth/components/SavedIdCard'
import { useUpdateProfile } from '@/features/auth/hooks/useUpdateProfile'
import { useAuthStore } from '@/stores/auth'

export default function SavedIDsPage() {
  const { t } = useTranslation()
  const toast = useToast()
  const user = useAuthStore((s) => s.user)
  const ids = user?.savedPlayerIds ?? []
  const updateProfile = useUpdateProfile()

  const [adding, setAdding] = useState(false)
  const [draft, setDraft] = useState('')

  const addId = () => {
    const v = draft.trim()
    if (!v) return
    if (ids.includes(v)) {
      toast(t('saved_players'), 'user')
      return
    }
    updateProfile.mutate(
      { savedPlayerIds: [...ids, v] },
      {
        onSuccess: () => {
          setDraft('')
          setAdding(false)
        },
        onError: (err) => toast(err.message, 'user'),
      },
    )
  }

  const removeId = (id: string) =>
    updateProfile.mutate(
      { savedPlayerIds: ids.filter((x) => x !== id) },
      { onError: (err) => toast(err.message, 'user') },
    )

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="row between" style={{ marginBottom: 24 }}>
        <h1 className="h1">{t('saved_players')}</h1>
      </div>
      <div className="cols-acct">
        <AcctSidebar active="savedids" />
        <div className="col" style={{ gap: 16 }}>
          <div className="grid" style={{ gridTemplateColumns: 'repeat(2,1fr)', gap: 16 }}>
            {ids.map((id) => (
              <SavedIdCard
                key={id}
                value={id}
                onDelete={updateProfile.isPending ? undefined : () => removeId(id)}
              />
            ))}
            {adding ? (
              <div className="panel card-pad" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <input
                  className="field"
                  autoFocus
                  placeholder={t('id_ph')}
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && addId()}
                />
                <div className="row" style={{ gap: 8 }}>
                  <Button variant="primary" size="sm" onClick={addId} disabled={updateProfile.isPending}>
                    {t('save_changes')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setAdding(false)
                      setDraft('')
                    }}
                  >
                    {t('cancel')}
                  </Button>
                </div>
              </div>
            ) : (
              <button
                className="panel card-pad clickable"
                style={{
                  border: '1.5px dashed var(--border-strong)',
                  display: 'grid',
                  placeItems: 'center',
                  minHeight: 96,
                  background: 'transparent',
                }}
                onClick={() => setAdding(true)}
              >
                <div className="col center" style={{ gap: 8, color: 'var(--text-dim)' }}>
                  <Icon name="plusc" size={30} />
                  <span style={{ fontWeight: 700 }}>{t('add_id')}</span>
                </div>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
