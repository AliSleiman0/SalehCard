import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { EmptyState, ErrorState, Icon, LoadingSpinner, Modal, Toggle } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import {
  ANDROID_ONLY_TAG,
  createKB,
  deleteKB,
  kbDocChars,
  listKB,
  updateKB,
  type KBDoc,
  type KBDocInput,
} from '../api/chatbot'

const KEY = ['admin', 'chatbot', 'kb']

const toInput = (d: KBDoc, patch: Partial<KBDocInput> = {}): KBDocInput => ({
  slug: d.slug ?? '',
  title: d.title,
  body: d.body,
  tags: d.tags ?? [],
  sortOrder: d.sortOrder,
  enabled: d.enabled,
  ...patch,
})

/** The AI assistant's knowledge base: the only material it may answer from. */
export function KnowledgeBaseTab({
  canManage,
  creating,
  onCreated,
}: {
  canManage: boolean
  creating: boolean
  onCreated: () => void
}) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState<KBDoc | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<KBDoc | null>(null)

  const q = useQuery({
    queryKey: KEY,
    queryFn: async () => {
      const res = await listKB()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
  })
  const refresh = () => void qc.invalidateQueries({ queryKey: KEY })
  const onErr = (e: unknown) =>
    toast.error(e instanceof ApiError ? e.message : 'Something went wrong')

  const toggle = useMutation({
    mutationFn: (d: KBDoc) => updateKB(d.id, toInput(d, { enabled: !d.enabled })),
    onSuccess: refresh,
    onError: onErr,
  })
  const del = useMutation({
    mutationFn: (id: string) => deleteKB(id),
    onSuccess: () => {
      setConfirmDelete(null)
      toast.success('Article deleted')
      refresh()
    },
    onError: onErr,
  })

  const data = q.data
  const pct = data ? Math.min(100, Math.round((data.usedChars / data.maxChars) * 100)) : 0
  const editorOpen = creating || !!editing

  return (
    <>
      <div className="acard" style={{ padding: '14px 18px', marginBottom: 14 }}>
        <div className="muted" style={{ fontSize: 13, lineHeight: 1.6 }}>
          The AI assistant answers typed questions <b>only</b> from the enabled articles below, and
          politely refuses anything they don’t cover (it then points the customer to support). It
          cannot see customer accounts, orders or balances. Turn it on in Settings → AI help
          assistant.
        </div>
        {data && (
          <div style={{ marginTop: 12 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 12.5,
                marginBottom: 6,
              }}
            >
              <span className="muted">Knowledge-base size (enabled articles)</span>
              <span style={{ fontWeight: 700 }} data-testid="kb-budget">
                {data.usedChars.toLocaleString()} / {data.maxChars.toLocaleString()} characters ·{' '}
                {data.docs.length}/{data.maxDocs} articles
              </span>
            </div>
            <div
              style={{
                height: 6,
                borderRadius: 4,
                background: 'var(--border)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${pct}%`,
                  height: '100%',
                  background: pct > 90 ? 'var(--danger)' : 'var(--accent, var(--primary))',
                }}
              />
            </div>
          </div>
        )}
      </div>

      <div className="acard" style={{ padding: 0 }}>
        {q.isLoading ? (
          <LoadingSpinner />
        ) : q.isError ? (
          <ErrorState message="Could not load the knowledge base." onRetry={() => q.refetch()} />
        ) : !data || data.docs.length === 0 ? (
          <EmptyState
            icon="file"
            title="No articles yet"
            sub="Write short articles about how the app works (buying, top-ups, verification, orders, refunds…). The assistant only answers from these."
          />
        ) : (
          data.docs.map((d) => {
            const androidOnly = (d.tags ?? []).includes(ANDROID_ONLY_TAG)
            return (
              <div
                key={d.id}
                style={{
                  display: 'flex',
                  gap: 12,
                  alignItems: 'flex-start',
                  padding: '12px 18px',
                  borderBottom: '1px solid var(--border)',
                  opacity: d.enabled ? 1 : 0.55,
                }}
              >
                <Icon name="file" size={15} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 700 }}>{d.title.en || d.title.ar}</div>
                  {d.title.en && d.title.ar && (
                    <div className="faint" dir="rtl" style={{ fontSize: 12.5 }}>
                      {d.title.ar}
                    </div>
                  )}
                  <div
                    className="muted"
                    style={{
                      fontSize: 13,
                      marginTop: 4,
                      whiteSpace: 'pre-wrap',
                      display: '-webkit-box',
                      WebkitLineClamp: 3,
                      WebkitBoxOrient: 'vertical',
                      overflow: 'hidden',
                    }}
                  >
                    {d.body.en || d.body.ar}
                  </div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 6 }}>
                    {androidOnly && <span className="bdg">Hidden on iOS</span>}
                    {(d.tags ?? [])
                      .filter((t) => t !== ANDROID_ONLY_TAG)
                      .map((t) => (
                        <span key={t} className="bdg">
                          {t}
                        </span>
                      ))}
                    <span className="faint" style={{ fontSize: 12 }}>
                      {kbDocChars(d).toLocaleString()} chars
                    </span>
                  </div>
                </div>
                <span className="faint" style={{ fontSize: 12 }}>
                  #{d.sortOrder}
                </span>
                {canManage && (
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <Toggle on={d.enabled} onClick={() => toggle.mutate(d)} />
                    <button className="abtn xs" onClick={() => setEditing(d)}>
                      <Icon name="edit" size={13} /> Edit
                    </button>
                    <button className="abtn xs danger" onClick={() => setConfirmDelete(d)}>
                      <Icon name="trash" size={13} />
                    </button>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>

      {editorOpen && (
        <KBEditor
          doc={editing ?? undefined}
          onClose={() => {
            setEditing(null)
            onCreated()
          }}
          onSaved={() => {
            setEditing(null)
            onCreated()
            refresh()
          }}
        />
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)}>
          <div style={{ padding: 22 }}>
            <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>
              Delete this article?
            </h3>
            <p className="muted" style={{ fontSize: 13.5 }}>
              “{confirmDelete.title.en || confirmDelete.title.ar}” will be removed from the
              knowledge base and the assistant will stop answering from it. This can’t be undone —
              disable it instead to keep it.
            </p>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                Cancel
              </button>
              <button
                className="abtn danger"
                disabled={del.isPending}
                onClick={() => del.mutate(confirmDelete.id)}
              >
                <Icon name="trash" size={14} /> Delete
              </button>
            </div>
          </div>
        </Modal>
      )}
    </>
  )
}

function KBEditor({
  doc,
  onClose,
  onSaved,
}: {
  doc?: KBDoc
  onClose: () => void
  onSaved: () => void
}) {
  const [titleEn, setTitleEn] = useState(doc?.title.en ?? '')
  const [titleAr, setTitleAr] = useState(doc?.title.ar ?? '')
  const [bodyEn, setBodyEn] = useState(doc?.body.en ?? '')
  const [bodyAr, setBodyAr] = useState(doc?.body.ar ?? '')
  const otherTags = (doc?.tags ?? []).filter((t) => t !== ANDROID_ONLY_TAG)
  const [tags, setTags] = useState(otherTags.join(', '))
  const [androidOnly, setAndroidOnly] = useState((doc?.tags ?? []).includes(ANDROID_ONLY_TAG))
  const [sortOrder, setSortOrder] = useState(doc?.sortOrder ?? 0)
  const [enabled, setEnabled] = useState(doc?.enabled ?? true)
  const [error, setError] = useState('')

  const save = useMutation({
    mutationFn: (input: KBDocInput) => (doc ? updateKB(doc.id, input) : createKB(input)),
    onSuccess: onSaved,
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not save the article.'),
  })

  const submit = () => {
    setError('')
    if (!titleEn.trim() && !titleAr.trim()) return setError('Enter a title in English or Arabic.')
    if (!bodyEn.trim() && !bodyAr.trim())
      return setError('Enter the article text in English or Arabic.')
    const tagList = tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    if (androidOnly) tagList.push(ANDROID_ONLY_TAG)
    save.mutate({
      slug: doc?.slug ?? '',
      title: { en: titleEn, ar: titleAr },
      body: { en: bodyEn, ar: bodyAr },
      tags: tagList,
      sortOrder: Number(sortOrder) || 0,
      enabled,
    })
  }

  const chars = kbDocChars({
    title: { en: titleEn, ar: titleAr },
    body: { en: bodyEn, ar: bodyAr },
  })

  return (
    <Modal onClose={onClose} maxWidth={760}>
      <div style={{ padding: 22, maxHeight: '86vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {doc ? 'Edit article' : 'New article'}
        </h3>
        <p className="muted" style={{ fontSize: 12.5, marginBottom: 14 }}>
          Write it the way you would explain it to a customer: the exact screens and steps, what
          they can and can’t do. Don’t include prices, codes or anything private — the assistant may
          quote any of it.
        </p>
        <div className="g2">
          <div>
            <label className="alabel">Title (English)</label>
            <input
              className="afield"
              value={titleEn}
              onChange={(e) => setTitleEn(e.target.value)}
            />
          </div>
          <div>
            <label className="alabel">Title (Arabic)</label>
            <input
              className="afield"
              dir="rtl"
              value={titleAr}
              onChange={(e) => setTitleAr(e.target.value)}
            />
          </div>
          <div>
            <label className="alabel">Article (English)</label>
            <textarea
              className="afield"
              rows={10}
              value={bodyEn}
              onChange={(e) => setBodyEn(e.target.value)}
            />
          </div>
          <div>
            <label className="alabel">Article (Arabic)</label>
            <textarea
              className="afield"
              rows={10}
              dir="rtl"
              value={bodyAr}
              onChange={(e) => setBodyAr(e.target.value)}
            />
          </div>
        </div>
        <div className="faint" style={{ fontSize: 12, marginTop: 6 }}>
          {chars.toLocaleString()} characters (max 4,000 per language)
        </div>

        <label className="alabel" style={{ marginTop: 12 }}>
          Tags (comma-separated, for your own organisation)
        </label>
        <input
          className="afield"
          placeholder="wallet, top-up"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
        />

        <div
          style={{
            display: 'flex',
            gap: 20,
            alignItems: 'center',
            flexWrap: 'wrap',
            marginTop: 14,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Toggle on={enabled} onClick={() => setEnabled((v) => !v)} />
            <span style={{ fontSize: 13.5 }}>Used by the assistant</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Toggle on={androidOnly} onClick={() => setAndroidOnly((v) => !v)} />
            <span style={{ fontSize: 13.5 }}>Hide from the iOS app</span>
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
        <div className="faint" style={{ fontSize: 12, marginTop: 8 }}>
          Turn on “Hide from the iOS app” for anything about crypto, Whish or money transfer — the
          iOS app must not mention them.
        </div>

        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 14 }}>
            {error}
          </div>
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
