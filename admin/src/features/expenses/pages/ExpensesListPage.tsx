import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { useExpenses, useExpenseSummary, useDeleteExpense } from '../hooks/useExpenses'
import { adaptExpense, amountLabel, type ExpenseView } from '../lib/adaptExpense'
import { listExpenses, type AdminExpense, type ExpenseCategory } from '../api/expenses'
import { downloadCsv } from '@/lib/utils'

const CATEGORIES: ExpenseCategory[] = [
  'salary',
  'rent',
  'utilities',
  'inventory',
  'marketing',
  'fees',
  'other',
]

/** ISO bounds for a yyyy-mm-dd <input type=date> value (empty → undefined). */
function startOfDayISO(d: string): string | undefined {
  return d ? new Date(`${d}T00:00:00`).toISOString() : undefined
}
function endOfDayISO(d: string): string | undefined {
  return d ? new Date(`${d}T23:59:59.999`).toISOString() : undefined
}

export default function ExpensesListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [category, setCategory] = useState<'' | ExpenseCategory>('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [toDelete, setToDelete] = useState<ExpenseView | null>(null)
  const [exporting, setExporting] = useState(false)

  // Resolve a category to its display label (custom text for "other").
  const catLabel = (c: ExpenseCategory, other?: string) =>
    c === 'other' ? other || t('exp_cat_other') : t(`exp_cat_${c}`)

  const filters = {
    category: category || undefined,
    from: startOfDayISO(from),
    to: endOfDayISO(to),
  }

  const { data, isLoading, isError, refetch } = useExpenses({ page, ...filters })
  const summary = useExpenseSummary(filters)

  const rows = useMemo(() => (data?.data ?? []).map(adaptExpense), [data])
  const meta = data?.meta
  const totals = summary.data?.data?.totals ?? []

  // Aggregate the per-(currency,category) breakdown to per-category (USD-only
  // deployments collapse to one row per category).
  const catBreakdown = useMemo(() => {
    const byCategory = summary.data?.data?.byCategory ?? []
    const map = new Map<string, { category: ExpenseCategory; currency: string; total: number }>()
    for (const b of byCategory) {
      const key = `${b.category}|${b.currency}`
      const existing = map.get(key)
      if (existing) existing.total += b.total
      else map.set(key, { category: b.category, currency: b.currency, total: b.total })
    }
    return [...map.values()].sort((a, b) => b.total - a.total)
  }, [summary.data])

  const resetPage = () => setPage(1)

  // Export the full filtered result set as a CSV (opens in Excel). The backend
  // caps limit at 100, so page through to the total (cf. InventoryPage.exportCodes).
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const all: AdminExpense[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listExpenses({ page: p, limit: 100, ...filters })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = [t('exp_date'), t('exp_category'), t('exp_note'), t('exp_amount'), 'Currency']
      const csvRows = all.map((e) => {
        const v = adaptExpense(e)
        return [v.dateLabel, catLabel(v.category, v.categoryOther), v.note || '', v.raw.amount, v.raw.currency]
      })
      downloadCsv(`expenses-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_expenses')]}
        title={t('nav_expenses')}
        sub={meta ? `${meta.total.toLocaleString()} ${t('exp_records')}` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
        </button>
        <button className="abtn primary" onClick={() => navigate('/expenses/new')}>
          <Icon name="plus" size={15} /> {t('exp_new')}
        </button>
      </PageHead>

      {/* Totals strip — per-currency grand total over all filtered rows. */}
      {totals.length > 0 && (
        <div className="formgrid" style={{ marginBottom: 16 }}>
          {totals.map((tot) => (
            <div key={tot.currency} className="acard pad">
              <div className="alabel">
                {t('exp_total')} ({tot.currency})
              </div>
              <div className="mono" style={{ fontSize: 26, fontWeight: 800 }}>
                {amountLabel(tot.total, tot.currency)}
              </div>
              <div className="muted" style={{ fontSize: 12.5, marginTop: 4 }}>
                {tot.count} {t('exp_records')}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="acard">
        <div className="toolbar">
          <select
            className="select"
            value={category}
            onChange={(e) => {
              setCategory(e.target.value as '' | ExpenseCategory)
              resetPage()
            }}
          >
            <option value="">{t('exp_all_categories')}</option>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {t(`exp_cat_${c}`)}
              </option>
            ))}
          </select>
          <label className="tb-field">
            <span>{t('exp_from')}</span>
            <input
              className="tb-date"
              type="date"
              value={from}
              onChange={(e) => {
                setFrom(e.target.value)
                resetPage()
              }}
            />
          </label>
          <label className="tb-field">
            <span>{t('exp_to')}</span>
            <input
              className="tb-date"
              type="date"
              value={to}
              onChange={(e) => {
                setTo(e.target.value)
                resetPage()
              }}
            />
          </label>
          <div className="tb-spacer" />
          <button
            className="abtn"
            onClick={handleExport}
            disabled={exporting || rows.length === 0}
          >
            <Icon name="download" size={15} /> {t('exp_export')}
          </button>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('exp_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState icon="coins" title={t('exp_empty')} />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{t('exp_date')}</th>
                    <th>{t('exp_category')}</th>
                    <th>{t('exp_note')}</th>
                    <th className="num">{t('exp_amount')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((e) => (
                    <tr
                      key={e.id}
                      className="clickable"
                      onClick={() => navigate(`/expenses/${e.id}/edit`)}
                    >
                      <td className="muted" style={{ fontSize: 12.5 }}>
                        {e.dateLabel}
                      </td>
                      <td className="strong">{catLabel(e.category, e.categoryOther)}</td>
                      <td className="muted">{e.note || '—'}</td>
                      <td className="num strong">{e.amountLabel}</td>
                      <td onClick={(ev) => ev.stopPropagation()}>
                        <div className="row-actions">
                          <span className="iact" onClick={() => navigate(`/expenses/${e.id}/edit`)}>
                            <Icon name="edit" size={15} />
                          </span>
                          <span className="iact danger" onClick={() => setToDelete(e)}>
                            <Icon name="trash" size={15} />
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {catBreakdown.length > 0 && (
              <div className="ahint" style={{ padding: '12px 16px', display: 'flex', flexWrap: 'wrap', gap: 14 }}>
                {catBreakdown.map((b) => (
                  <span key={`${b.category}|${b.currency}`}>
                    <b>{t(`exp_cat_${b.category}`)}:</b> {amountLabel(b.total, b.currency)}
                  </span>
                ))}
              </div>
            )}

            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('nav_expenses')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toDelete && (
        <DeleteModal expense={toDelete} label={catLabel(toDelete.category, toDelete.categoryOther)} onClose={() => setToDelete(null)} />
      )}
    </div>
  )
}

/** Confirmation dialog for deleting an expense. */
function DeleteModal({
  expense,
  label,
  onClose,
}: {
  expense: ExpenseView
  label: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const del = useDeleteExpense()

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>{t('exp_delete_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          {t('exp_delete_confirm')}{' '}
          <b>
            {label} · {expense.amountLabel}
          </b>
          ? {t('exp_delete_irreversible')}
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>{t('exp_delete_error')}</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={del.isPending}>
            {t('cancel')}
          </button>
          <button
            className="abtn danger"
            disabled={del.isPending}
            onClick={() => del.mutate(expense.id, { onSuccess: onClose })}
          >
            <Icon name="trash" size={15} /> {t('delete')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
