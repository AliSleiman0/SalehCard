import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Checkbox, EmptyState, ErrorState, Icon, LoadingSpinner, Modal, PageHead } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { createTodo, deleteTodo, listTodos, updateTodo, type Todo, type TodoInput } from '../api/todos'
import { dateInputToIso, dueState } from '../lib/due'

const KEY = ['admin', 'todos']

// Due badge per state, using the theme's .st-* status classes (they follow
// light/dark mode). Upcoming dates are neutral.
const DUE_CLASS: Record<string, { cls: string; label: string }> = {
  overdue: { cls: 'st st-danger', label: 'Overdue' },
  today: { cls: 'st st-warn', label: 'Today' },
  upcoming: { cls: 'st st-mute', label: '' },
}

function fmtDate(iso: string, withTime = false): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const date = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  return withTime ? `${date} · ${d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}` : date
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
  const onErr = (e: unknown) => toast.error(e instanceof ApiError ? e.message : 'Something went wrong')
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
              ? `Done ${fmtDate(x.doneAt, true)}${x.doneBy ? ` by ${x.doneBy}` : ''}`
              : `Added ${fmtDate(x.createdAt, true)}${x.createdBy ? ` by ${x.createdBy}` : ''}`}
          </div>
        </div>
        {x.dueAt && !x.done && badge && (
          <span className={badge.cls} style={{ whiteSpace: 'nowrap' }} title={fmtDate(x.dueAt)}>
            <i className="d" /> {badge.label ? `${badge.label} · ` : ''}
            {fmtDate(x.dueAt)}
          </span>
        )}
        <button
          className="abtn xs ghost"
          aria-label="Delete to-do"
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
        crumbs={[t('grp_overview'), t('nav_todos')]}
        title={t('nav_todos')}
        sub={`${open.length} open · shared with every admin`}
      />

      <div className="acard pad" style={{ marginBottom: 18 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,2fr) minmax(0,2fr) 170px auto', gap: 10 }} className="todo-add">
          <input
            className="afield"
            placeholder="What needs doing?"
            aria-label="To-do"
            value={title}
            maxLength={200}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && submit()}
          />
          <input
            className="afield"
            placeholder="Note (optional)"
            aria-label="Note"
            value={note}
            maxLength={1000}
            onChange={(e) => setNote(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && submit()}
          />
          <input className="afield" type="date" aria-label="Due date" value={due} onChange={(e) => setDue(e.target.value)} />
          <button className="abtn primary" onClick={submit} disabled={!title.trim() || add.isPending}>
            <Icon name="plus" size={15} /> Add
          </button>
        </div>
      </div>

      {q.isLoading ? (
        <LoadingSpinner />
      ) : q.isError ? (
        <ErrorState message="Couldn't load the to-do list." onRetry={() => void q.refetch()} />
      ) : todos.length === 0 ? (
        <EmptyState icon="checkc" title="Nothing to do" sub="Add a task above — every admin sees the same list." />
      ) : (
        <>
          <div className="acard" style={{ marginBottom: 18 }}>
            <div className="panelhead">
              <Icon name="checkc" size={17} />
              <h3>Open ({open.length})</h3>
            </div>
            {open.length === 0 ? (
              <div className="muted" style={{ padding: '14px 18px' }}>
                All done.
              </div>
            ) : (
              open.map(row)
            )}
          </div>
          {done.length > 0 && (
            <div className="acard">
              <div className="panelhead">
                <Icon name="check" size={17} />
                <h3>Done — last 30 days ({done.length})</h3>
              </div>
              {done.map(row)}
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)} maxWidth={420}>
          <div style={{ padding: 22 }}>
            <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Delete this to-do?</h3>
            <div className="muted" style={{ fontSize: 13, marginBottom: 6, wordBreak: 'break-word' }}>
              “{confirmDelete.title}”
            </div>
            <div className="faint" style={{ fontSize: 12.5, marginBottom: 18 }}>
              Every admin shares this list. The deletion is recorded in the activity log.
            </div>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                Cancel
              </button>
              <button
                className="abtn primary"
                style={{ background: 'var(--danger)' }}
                onClick={() => {
                  remove.mutate(confirmDelete.id)
                  setConfirmDelete(null)
                }}
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
