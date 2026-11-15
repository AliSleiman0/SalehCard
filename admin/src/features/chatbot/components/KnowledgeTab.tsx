import { useEffect, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ErrorState, Icon, LoadingSpinner } from '@/components'
import { toast } from '@/stores/toast'
import { apiErrorText } from '@/lib/apiErrorText'
import { formatDateTime } from '@/lib/utils'
import {
  getKnowledge,
  knowledgeChars,
  paragraphCount,
  saveKnowledge,
  type KnowledgeInput,
} from '../api/chatbot'

const KEY = ['admin', 'chatbot', 'knowledge']

/** The AI assistant's knowledge: one plain-text box (any language — the
 *  assistant answers in the customer's language) plus a second box the iOS
 *  app never gets. Paragraphs separated by a blank line are what an answer
 *  cites, so one topic per paragraph works best. */
export function KnowledgeTab({ canManage }: { canManage: boolean }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const q = useQuery({
    queryKey: KEY,
    queryFn: async () => {
      const res = await getKnowledge()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
  })
  const [draft, setDraft] = useState<KnowledgeInput>({ general: '', androidOnly: '' })
  const [error, setError] = useState('')

  useEffect(() => {
    if (q.data) setDraft({ general: q.data.general, androidOnly: q.data.androidOnly })
  }, [q.data])

  const save = useMutation({
    mutationFn: async (input: KnowledgeInput) => {
      const res = await saveKnowledge(input)
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Could not save')
      return res.data
    },
    onSuccess: (data) => {
      qc.setQueryData(KEY, data)
      setError('')
      toast.success(t('bot_toast_saved'))
    },
    onError: (e) => setError(apiErrorText(e, 'bot_save_failed')),
  })

  if (q.isLoading) return <LoadingSpinner />
  if (q.isError || !q.data)
    return <ErrorState message={t('bot_load_error')} onRetry={() => q.refetch()} />

  const max = q.data.maxChars
  const used = knowledgeChars(draft)
  const over = used > max
  const dirty = draft.general !== q.data.general || draft.androidOnly !== q.data.androidOnly
  const pct = Math.min(100, Math.round((used / max) * 100))

  const box = (
    key: keyof KnowledgeInput,
    label: string,
    hint: string,
    rows: number,
    placeholder: string
  ) => (
    <div style={{ marginBottom: 18 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 }}>
        <label className="alabel" htmlFor={`kn-${key}`}>
          {label}
        </label>
        <span className="faint" style={{ fontSize: 12 }}>
          {t('bot_paragraphs', { n: paragraphCount(draft[key]) })}
        </span>
      </div>
      <div className="muted" style={{ fontSize: 12.5, marginBottom: 6 }}>
        {hint}
      </div>
      <textarea
        id={`kn-${key}`}
        className="afield"
        dir="auto"
        rows={rows}
        readOnly={!canManage}
        placeholder={placeholder}
        value={draft[key]}
        onChange={(e) => setDraft((d) => ({ ...d, [key]: e.target.value }))}
        style={{ fontSize: 13.5, lineHeight: 1.6, resize: 'vertical' }}
      />
    </div>
  )

  return (
    <div className="acard">
      <div className="pad">
        <div className="muted" style={{ fontSize: 13, lineHeight: 1.6, marginBottom: 16, maxWidth: 760 }}>
          <Trans i18nKey="bot_intro" components={{ b: <b /> }} />
        </div>

        {box(
          'general',
          t('bot_knowledge'),
          t('bot_knowledge_hint'),
          18,
          t('bot_knowledge_ph')
        )}
        {box(
          'androidOnly',
          t('bot_android'),
          t('bot_android_hint'),
          7,
          t('bot_android_ph')
        )}

        <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
          <div style={{ flex: '1 1 260px', minWidth: 200 }}>
            <div
              style={{
                height: 6,
                borderRadius: 3,
                background: 'var(--border)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${pct}%`,
                  height: '100%',
                  background: over ? 'var(--danger)' : pct > 85 ? 'var(--warn)' : 'var(--accent)',
                }}
              />
            </div>
            <div
              className={over ? undefined : 'faint'}
              style={{ fontSize: 12, marginTop: 6, color: over ? 'var(--danger)' : undefined }}
            >
              {t('bot_chars', { used: used.toLocaleString(), max: max.toLocaleString() })}
              {over && t('bot_over')}
              {q.data.updatedAt && !q.data.updatedAt.startsWith('0001') && (
                <>
                  {' · '}
                  {q.data.updatedBy
                    ? t('bot_last_saved_by', { when: formatDateTime(q.data.updatedAt), who: q.data.updatedBy })
                    : t('bot_last_saved', { when: formatDateTime(q.data.updatedAt) })}
                </>
              )}
            </div>
          </div>
          {canManage && (
            <div style={{ display: 'flex', gap: 10 }}>
              <button
                className="abtn"
                disabled={!dirty || save.isPending}
                onClick={() => {
                  setDraft({ general: q.data.general, androidOnly: q.data.androidOnly })
                  setError('')
                }}
              >
                {t('bot_discard')}
              </button>
              <button
                className="abtn primary"
                disabled={!dirty || over || save.isPending}
                onClick={() => save.mutate(draft)}
              >
                <Icon name="check" size={15} /> {save.isPending ? t('bot_saving') : t('sup_save')}
              </button>
            </div>
          )}
        </div>
        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 12 }}>
            {error}
          </div>
        )}
      </div>
    </div>
  )
}
