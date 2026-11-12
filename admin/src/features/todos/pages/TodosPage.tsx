import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Checkbox, EmptyState, ErrorState, Icon, LoadingSpinner, PageHead } from '@/components'
import { ApiError } from '@/lib/api-client'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { createTodo, deleteTodo, listTodos, updateTodo, type Todo, type TodoInput } from '../api/todos'
import { dateInputToIso, dueState } from '../lib/due'

const KEY = ['admin', 'todos']

const DUE_STYLE: Record<string, { bg: string; fg: string; label: string }> = {
  overdue: { bg: 'var(--danger-bg, rgba(229,72,77,.12))', fg: 'var(--danger)', label: 'Overdue' },
  today: { bg: 'var(--warn-bg, rgba(245,166,35,.14))', fg: 'var(--warn, #b7791f)', label: 'Today' },
  upcoming: { bg: 'var(--surface-2)', fg: 'var(--text-dim)', label: '' },
}

function fmtDate(iso: string, withTime = false): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const date = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  return withTime ? `${date} · ${d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}` : date
}

/** Shared team to-do list (client feedback 2026-11-12 #18). */
export default function TodosPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const canManage = useCan()('todos.manage')
  const [title, setTitle] = useState('')
  const [note, setNote] = useState('')
  const [due, setDue] = useState('')

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
  const add = useMutation({
    mutationFn: (input: TodoInput) => createTodo(input),
    onSuccess: () => {
      setTitle('')
      setNote('')
      setDue('')
      void qc.invalidateQueries({ queryKey: KEY })
    },
    onError: onErr,
  })
  const patch = useMutation({
    mutationFn: ({ id, input }: { id: string; input: TodoInput }) => updateTodo(id, input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
    onError: onErr,
  })
  const remove = useMutation({
    mutationFn: (id: string) => deleteTodo(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
    onError: onErr,
  })

  const submit = () => {
    if (!title.trim()) return
    add.mutate({ title: title.trim(), note: note.trim() || undefined, dueAt: dateInputToIso(due) })
  }

  const todos = q.data ?? []
  const open = todos.filter((x) => !x.done)
  const done = todos.filter((x) => x.done)

  const row = (x: Todo) => {
    const ds = x.done ? 'none' : dueState(x.dueAt)
    const st = DUE_STYLE[ds]
    return (
      <div
        key={x.id}
        style={{ display: 'flex', gap: 12, alignItems: 'flex-start', padding: '12px 18px', borderTop: '1px solid var(--border)' }}
      >
        <div style={{ paddingTop: 2 }}>
          <Checkbox
            on={x.done}
            onClick={() => {
              if (canManage) patch.mutate({ id: x.id, input: { done: !x.done } })
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
        {x.dueAt && !x.done && st && (
          <span
            className="bdg"
            style={{ background: st.bg, color: st.fg, whiteSpace: 'nowrap' }}
            title={fmtDate(x.dueAt)}
          >
            <Icon name="clock" size={12} /> {st.label ? `${st.label} · ` : ''}
            {fmtDate(x.dueAt)}
          </span>
        )}
        {canManage && (
          <button
            className="abtn xs ghost"
            aria-label="Delete to-do"
            onClick={() => remove.mutate(x.id)}
            disabled={remove.isPending}
          >
            <Icon name="trash" size={13} />
          </button>
        )}
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

      {canManage && (
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
      )}

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
    </div>
  )
}
