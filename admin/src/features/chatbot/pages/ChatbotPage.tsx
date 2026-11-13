import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { EmptyState, ErrorState, Icon, LoadingSpinner, Modal, PageHead, Toggle } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { useCan } from '@/stores/auth'
import {
  createEntry,
  deleteEntry,
  groupEntries,
  listEntries,
  updateEntry,
  type ChatEntry,
  type ChatEntryInput,
} from '../api/chatbot'

const KEY = ['admin', 'chatbot']

/** Help-bot Q&A manager (client feedback 2026-11-12 #74). Predefined questions
 *  and answers only (no AI): customers tap a question on the storefront and get
 *  its answer, then its follow-up questions. */
export default function ChatbotPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const can = useCan()
  const canManage = can('chatbot.manage')
  const [editing, setEditing] = useState<{ entry?: ChatEntry; parentId?: string } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<{ entry: ChatEntry; followUps: number } | null>(null)

  const q = useQuery({
    queryKey: KEY,
    queryFn: async () => {
      const res = await listEntries()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
  })
  const refresh = () => void qc.invalidateQueries({ queryKey: KEY })
  const onErr = (e: unknown) => toast.error(e instanceof ApiError ? e.message : 'Something went wrong')

  const toggle = useMutation({
    mutationFn: (e: ChatEntry) =>
      updateEntry(e.id, {
        parentId: e.parentId ?? '',
        question: e.question,
        answer: e.answer,
        keywords: e.keywords ?? [],
        sortOrder: e.sortOrder,
        enabled: !e.enabled,
      }),
    onSuccess: refresh,
    onError: onErr,
  })
  const del = useMutation({
    mutationFn: (id: string) => deleteEntry(id),
    onSuccess: () => {
      setConfirmDelete(null)
      toast.success('Question deleted')
      refresh()
    },
    onError: onErr,
  })

  const groups = groupEntries(q.data ?? [])

  const row = (e: ChatEntry, followUp: boolean, followUps = 0) => (
    <div
      key={e.id}
      style={{
        display: 'flex',
        gap: 12,
        alignItems: 'flex-start',
        padding: '12px 18px',
        paddingInlineStart: followUp ? 44 : 18,
        borderBottom: '1px solid var(--border)',
        opacity: e.enabled ? 1 : 0.55,
      }}
    >
      <Icon name={followUp ? 'chevright' : 'send'} size={15} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontWeight: 700 }}>{e.question.en || e.question.ar}</div>
        {e.question.en && e.question.ar && (
          <div className="faint" dir="rtl" style={{ fontSize: 12.5 }}>
            {e.question.ar}
          </div>
        )}
        <div className="muted" style={{ fontSize: 13, marginTop: 4, whiteSpace: 'pre-wrap' }}>
          {e.answer.en || e.answer.ar}
        </div>
        {e.keywords && e.keywords.length > 0 && (
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 6 }}>
            {e.keywords.map((k) => (
              <span key={k} className="bdg">
                {k}
              </span>
            ))}
          </div>
        )}
      </div>
      <span className="faint" style={{ fontSize: 12 }}>
        #{e.sortOrder}
      </span>
      {canManage && (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <Toggle on={e.enabled} onClick={() => toggle.mutate(e)} />
          {!followUp && (
            <button className="abtn xs" onClick={() => setEditing({ parentId: e.id })} title="Add a follow-up">
              <Icon name="plus" size={13} /> Follow-up
            </button>
          )}
          <button className="abtn xs" onClick={() => setEditing({ entry: e })}>
            <Icon name="edit" size={13} /> Edit
          </button>
          <button className="abtn xs danger" onClick={() => setConfirmDelete({ entry: e, followUps })}>
            <Icon name="trash" size={13} />
          </button>
        </div>
      )}
    </div>
  )

  return (
    <div>
      <PageHead
        back="/"
        title={t('nav_chatbot')}
        sub="Predefined questions and answers for the storefront help bot (web + app). Customers tap a question to see its answer and follow-ups."
      >
        {canManage && (
          <button className="abtn primary" onClick={() => setEditing({})}>
            <Icon name="plus" size={15} /> New question
          </button>
        )}
      </PageHead>
      <div className="acard" style={{ padding: 0 }}>
        {q.isLoading ? (
          <LoadingSpinner />
        ) : q.isError ? (
          <ErrorState message="Could not load the questions." onRetry={() => q.refetch()} />
        ) : groups.length === 0 ? (
          <EmptyState
            icon="send"
            title="No questions yet"
            sub="Add the questions customers ask most (top-ups, delivery times, KYC…). The help button stays hidden until there is at least one."
          />
        ) : (
          groups.map((g) => (
            <div key={g.top.id}>
              {row(g.top, !!g.top.parentId, g.children.length)}
              {g.children.map((c) => row(c, true))}
            </div>
          ))
        )}
      </div>

      {editing && (
        <EntryEditor
          entry={editing.entry}
          parentId={editing.entry?.parentId ?? editing.parentId ?? ''}
          tops={(q.data ?? []).filter((e) => !e.parentId && e.id !== editing.entry?.id)}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null)
            refresh()
          }}
        />
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)}>
          <div style={{ padding: 22 }}>
            <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>Delete this question?</h3>
            <p className="muted" style={{ fontSize: 13.5 }}>
              “{confirmDelete.entry.question.en || confirmDelete.entry.question.ar}”
              {confirmDelete.followUps > 0 &&
                ` and its ${confirmDelete.followUps} follow-up question${confirmDelete.followUps === 1 ? '' : 's'}`}{' '}
              will be removed from the help bot. This can’t be undone.
            </p>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                Cancel
              </button>
              <button
                className="abtn danger"
                disabled={del.isPending}
                onClick={() => del.mutate(confirmDelete.entry.id)}
              >
                <Icon name="trash" size={14} /> Delete
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

function EntryEditor({
  entry,
  parentId: initialParent,
  tops,
  onClose,
  onSaved,
}: {
  entry?: ChatEntry
  parentId: string
  tops: ChatEntry[]
  onClose: () => void
  onSaved: () => void
}) {
  const [parentId, setParentId] = useState(initialParent)
  const [qEn, setQEn] = useState(entry?.question.en ?? '')
  const [qAr, setQAr] = useState(entry?.question.ar ?? '')
  const [aEn, setAEn] = useState(entry?.answer.en ?? '')
  const [aAr, setAAr] = useState(entry?.answer.ar ?? '')
  const [keywords, setKeywords] = useState((entry?.keywords ?? []).join(', '))
  const [sortOrder, setSortOrder] = useState(entry?.sortOrder ?? 0)
  const [enabled, setEnabled] = useState(entry?.enabled ?? true)
  const [error, setError] = useState('')

  const save = useMutation({
    mutationFn: (input: ChatEntryInput) => (entry ? updateEntry(entry.id, input) : createEntry(input)),
    onSuccess: onSaved,
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not save the question.'),
  })

  const submit = () => {
    setError('')
    if (!qEn.trim() && !qAr.trim()) return setError('Enter the question in English or Arabic.')
    if (!aEn.trim() && !aAr.trim()) return setError('Enter the answer in English or Arabic.')
    save.mutate({
      parentId,
      question: { en: qEn, ar: qAr },
      answer: { en: aEn, ar: aAr },
      keywords: keywords
        .split(',')
        .map((k) => k.trim())
        .filter(Boolean),
      sortOrder: Number(sortOrder) || 0,
      enabled,
    })
  }

  return (
    <Modal onClose={onClose} maxWidth={640}>
      <div style={{ padding: 22, maxHeight: '84vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 16 }}>
          {entry ? 'Edit question' : parentId ? 'New follow-up question' : 'New question'}
        </h3>

        <label className="alabel">Shown</label>
        <select className="afield" value={parentId} onChange={(e) => setParentId(e.target.value)}>
          <option value="">As a top-level question</option>
          {tops.map((t) => (
            <option key={t.id} value={t.id}>
              After: {t.question.en || t.question.ar}
            </option>
          ))}
        </select>

        <div className="g2" style={{ marginTop: 12 }}>
          <div>
            <label className="alabel">Question (English)</label>
            <input className="afield" value={qEn} onChange={(e) => setQEn(e.target.value)} />
          </div>
          <div>
            <label className="alabel">Question (Arabic)</label>
            <input className="afield" dir="rtl" value={qAr} onChange={(e) => setQAr(e.target.value)} />
          </div>
          <div>
            <label className="alabel">Answer (English)</label>
            <textarea className="afield" rows={5} value={aEn} onChange={(e) => setAEn(e.target.value)} />
          </div>
          <div>
            <label className="alabel">Answer (Arabic)</label>
            <textarea className="afield" rows={5} dir="rtl" value={aAr} onChange={(e) => setAAr(e.target.value)} />
          </div>
        </div>

        <label className="alabel" style={{ marginTop: 12 }}>
          Keywords (comma-separated, help customers find it when they type)
        </label>
        <input
          className="afield"
          placeholder="wallet, top up, شحن"
          value={keywords}
          onChange={(e) => setKeywords(e.target.value)}
        />

        <div style={{ display: 'flex', gap: 20, alignItems: 'center', marginTop: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Toggle on={enabled} onClick={() => setEnabled((v) => !v)} />
            <span style={{ fontSize: 13.5 }}>Shown to customers</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span className="alabel" style={{ margin: 0 }}>
              Order
            </span>
            <input
              className="afield"
              type="number"
              style={{ width: 90 }}
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
            />
          </div>
        </div>

        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 14 }}>{error}</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
          <button className="abtn" onClick={onClose}>
            Cancel
          </button>
          <button className="abtn primary" disabled={save.isPending} onClick={submit}>
            <Icon name="check" size={15} /> Save
          </button>
        </div>
      </div>
    </Modal>
  )
}
