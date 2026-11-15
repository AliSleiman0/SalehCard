import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Checkbox, EmptyState, ErrorState, Icon, LoadingSpinner, Modal, PageHead } from '@/components'
import { apiErrorText } from '@/lib/apiErrorText'
import { formatDate, formatDateTime } from '@/lib/utils'
import { toast } from '@/stores/toast'
import { createTodo, deleteTodo, listTodos, updateTodo, type Todo, type TodoInput } from '../api/todos'
import { dateInputToIso, dueState } from '../lib/due'

const KEY = ['admin', 'todos']

// Due badge per state, using the theme's .st-* status classes (they follow
// light/dark mode). Upcoming dates are neutral. Labels are translation keys.
const DUE_CLASS: Record<string, { cls: string; label: string }> = {
  overdue: { cls: 'st st-danger', label: 'todo_overdue' },
  today: { cls: 'st st-warn', label: 'today' },
  upcoming: { cls: 'st st-mute', label: '' },
}

/** Shared team to-do list (client feedback 2026-11-12 #18). Every admin can
 *  add, tick and delete; deletes ask for confirmation and are audited. */
export default function TodosPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const [note, setNote] = useState('')
  const [due, setDue] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<Todo | null>(null)
  // Per-row in-flight state, so one row's request doesn't grey out the others
  // and a double-click can't send the same toggle/delete twice.
  const [busy, setBusy] = useState<Set<string>>(new Set())
  const mark = (id: string, on: boolean) =>
    setBusy((s) => {
      const next = new Set(s)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })

  const q = useQuery({
    queryKey: KEY,
    queryFn: async () => {
      const res = await listTodos()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
    refetchInterval: 30_000, // a shared list: pick up teammates' changes
  })
  const onErr = (e: unknown) => toast.error(apiErrorText(e, 'cmn_something_wrong'))
  const refresh = () => void qc.invalidateQueries({ queryKey: KEY })

  const add = useMutation({
    mutationFn: (input: TodoInput) => createTodo(input),
    onSuccess: () => {
      setTitle('')
      setNote('')
      setDue('')
      refresh()
    },
    onError: onErr,
  })
  const patch = useMutation({
    mutationFn: ({ id, input }: { id: string; input: TodoInput }) => updateTodo(id, input),
    onMutate: ({ id }) => mark(id, true),
    onSettled: (_d, _e, { id }) => {
      mark(id, false)
      refresh()
    },
    onError: onErr,
  })
  const remove = useMutation({
    mutationFn: (id: string) => deleteTodo(id),
    onMutate: (id) => mark(id, true),
    onSettled: (_d, _e, id) => {
      mark(id, false)
      refresh()
    },
    onError: onErr,
  })

  const submit = () => {
    // Enter and the Add button both land here — ignore repeats while saving.
    if (!title.trim() || add.isPending) return
    add.mutate({ title: title.trim(), note: note.trim() || undefined, dueAt: dateInputToIso(due) })
  }

  const todos = q.data ?? []
  const open = todos.filter((x) => !x.done)
  const done = todos.filter((x) => x.done)

  const row = (x: Todo) => {
    const ds = x.done ? 'none' : dueState(x.dueAt)
    const badge = DUE_CLASS[ds]
    const rowBusy = busy.has(x.id)
    return (
      <div
        key={x.id}
        style={{ display: 'flex', gap: 12, alignItems: 'flex-start', padding: '12px 18px', borderTop: '1px solid var(--border)' }}
      >
        <div style={{ paddingTop: 2, opacity: rowBusy ? 0.5 : 1 }}>
          <Checkbox
            on={x.done}
            onClick={() => {
              if (!rowBusy) patch.mutate({ id: x.id, input: { done: !x.done } })
            }}
          />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontWeight: 700,
              fontSize: 14,
              textDecoration: x.done ? 'line-through' : undefined,
              color: x.done ? 'var(--text-dim)' : undefined,
              wordBreak: 'break-word',
            }}
          >
            {x.title}
          </div>
          {x.note && (
            <div className="muted" style={{ fontSize: 12.5, marginTop: 2, whiteSpace: 'pre-line' }}>
              {x.note}
            </div>
          )}
          <div className="muted" style={{ fontSize: 11.5, marginTop: 4 }}>
            {x.done && x.doneAt
              ? x.doneBy
                ? t('todo_done_by', { when: formatDateTime(x.doneAt), who: x.doneBy })
                : t('todo_done_at', { when: formatDateTime(x.doneAt) })
              : x.createdBy
                ? t('todo_added_by', { when: formatDateTime(x.createdAt), who: x.createdBy })
                : t('todo_added_at', { when: formatDateTime(x.createdAt) })}
          </div>
        </div>
        {x.dueAt && !x.done && badge && (
          <span className={badge.cls} style={{ whiteSpace: 'nowrap' }} title={formatDate(x.dueAt)}>
            <i className="d" /> {badge.label ? `${t(badge.label)} · ` : ''}
            {formatDate(x.dueAt)}
          </span>
        )}
        <button
          className="abtn xs ghost"
          aria-label={t('todo_delete_aria')}
          onClick={() => setConfirmDelete(x)}
          disabled={rowBusy}
        >
          <Icon name="trash" size={13} />
        </button>
      </div>
    )
  }

  return (
    <div className="page">
      <PageHead
        back="/"
        crumbs={[t('grp_overview'), t('nav_todos')]}
        title={t('nav_todos')}
        sub={t('todo_sub', { n: open.length })}
      />

      <div className="acard pad" style={{ marginBottom: 18 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,2fr) minmax(0,2fr) 170px auto', gap: 10 }} className="todo-add">
          <input
            className="afield"
            placeholder={t('todo_title_ph')}
            aria-label={t('todo_title_aria')}
            value={title}
            maxLength={200}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && submit()}
          />
          <input
            className="afield"
            placeholder={t('todo_note_ph')}
            aria-label={t('exp_note')}
            value={note}
            maxLength={1000}
            onChange={(e) => setNote(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && submit()}
          />
          <input className="afield" type="date" aria-label={t('todo_due_aria')} value={due} onChange={(e) => setDue(e.target.value)} />
          <button className="abtn primary" onClick={submit} disabled={!title.trim() || add.isPending}>
            <Icon name="plus" size={15} /> {t('add')}
          </button>
        </div>
      </div>

      {q.isLoading ? (
        <LoadingSpinner />
      ) : q.isError ? (
        <ErrorState message={t('todo_load_error')} onRetry={() => void q.refetch()} />
      ) : todos.length === 0 ? (
        <EmptyState icon="checkc" title={t('todo_empty')} sub={t('todo_empty_sub')} />
      ) : (
        <>
          <div className="acard" style={{ marginBottom: 18 }}>
            <div className="panelhead">
              <Icon name="checkc" size={17} />
              <h3>{t('todo_open_n', { n: open.length })}</h3>
            </div>
            {open.length === 0 ? (
              <div className="muted" style={{ padding: '14px 18px' }}>
                {t('todo_all_done')}
              </div>
            ) : (
              open.map(row)
            )}
          </div>
          {done.length > 0 && (
            <div className="acard">
              <div className="panelhead">
                <Icon name="check" size={17} />
                <h3>{t('todo_done_n', { n: done.length })}</h3>
              </div>
              {done.map(row)}
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)} maxWidth={420}>
          <div style={{ padding: 22 }}>
            <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>{t('todo_delete_title')}</h3>
            <div className="muted" style={{ fontSize: 13, marginBottom: 6, wordBreak: 'break-word' }}>
              {t('cmn_quote', { s: confirmDelete.title })}
            </div>
            <div className="faint" style={{ fontSize: 12.5, marginBottom: 18 }}>
              {t('todo_delete_note')}
            </div>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                {t('cancel')}
              </button>
              <button
                className="abtn primary"
                style={{ background: 'var(--danger)' }}
                onClick={() => {
                  remove.mutate(confirmDelete.id)
                  setConfirmDelete(null)
                }}
              >
                <Icon name="trash" size={14} /> {t('delete')}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
