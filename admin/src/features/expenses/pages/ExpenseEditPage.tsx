import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, LoadingSpinner, ErrorState } from '@/components'
import { ApiError } from '@/lib/api-client'
import { useExpense, useCreateExpense, useUpdateExpense } from '../hooks/useExpenses'
import type { ExpenseInput, ExpenseCategory } from '../api/expenses'

const CATEGORIES: ExpenseCategory[] = [
  'salary',
  'rent',
  'utilities',
  'inventory',
  'marketing',
  'fees',
  'other',
]

// dateInput converts an ISO timestamp to a yyyy-mm-dd value for <input type=date>.
function dateInput(iso?: string): string {
  return iso ? iso.slice(0, 10) : ''
}

// todayInput is today's date as a yyyy-mm-dd default for new expenses.
function todayInput(): string {
  return new Date().toISOString().slice(0, 10)
}

export default function ExpenseEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isNew = !id

  const { data, isLoading, isError, refetch } = useExpense(id)
  const create = useCreateExpense()
  const update = useUpdateExpense(id ?? '')

  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [category, setCategory] = useState<ExpenseCategory>('salary')
  const [categoryOther, setCategoryOther] = useState('')
  const [note, setNote] = useState('')
  const [incurredAt, setIncurredAt] = useState(todayInput())
  const [error, setError] = useState('')

  // Populate the form once the existing expense loads (edit mode).
  useEffect(() => {
    const e = data?.data
    if (!e) return
    setAmount(String(e.amount))
    setCurrency(e.currency)
    setCategory(e.category)
    setCategoryOther(e.categoryOther ?? '')
    setNote(e.note ?? '')
    setIncurredAt(dateInput(e.incurredAt))
  }, [data])

  if (!isNew && isLoading) return <LoadingSpinner />
  if (!isNew && (isError || !data?.data)) {
    return (
      <div className="page">
        <ErrorState message={t('exp_load_one_error')} onRetry={() => refetch()} />
      </div>
    )
  }

  const saving = create.isPending || update.isPending

  const save = () => {
    const v = Number(amount)
    if (!Number.isFinite(v) || v <= 0) {
      setError(t('exp_err_amount'))
      return
    }
    if (!currency.trim()) {
      setError(t('exp_err_currency'))
      return
    }
    if (category === 'other' && !categoryOther.trim()) {
      setError(t('exp_err_other'))
      return
    }
    if (!incurredAt) {
      setError(t('exp_err_date'))
      return
    }
    setError('')
    const input: ExpenseInput = {
      amount: v,
      currency: currency.trim().toUpperCase(),
      category,
      categoryOther: category === 'other' ? categoryOther.trim() : undefined,
      note: note.trim() || undefined,
      incurredAt: new Date(`${incurredAt}T00:00:00`).toISOString(),
    }
    const onError = (e: unknown) => setError(e instanceof ApiError ? e.message : t('exp_save_error'))
    if (isNew) {
      create.mutate(input, { onSuccess: () => navigate('/expenses'), onError })
    } else {
      update.mutate(input, { onSuccess: () => navigate('/expenses'), onError })
    }
  }

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_expenses'), isNew ? t('exp_new') : t('exp_edit')]}
        title={isNew ? t('exp_new') : t('exp_edit')}
        sub={t('exp_edit_sub')}
      >
        <button className="abtn" onClick={() => navigate('/expenses')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary" onClick={save} disabled={saving}>
          <Icon name="check" size={15} /> {t('save')}
        </button>
      </PageHead>

      {error && (
        <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginBottom: 14 }}>{error}</div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('exp_section_amount')}</h3>
            <div className="g2">
              <div>
                <label className="alabel">{t('exp_amount')}</label>
                <input
                  className="afield"
                  type="number"
                  min="0"
                  step="0.01"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                />
              </div>
              <div>
                <label className="alabel">{t('exp_currency')}</label>
                <input
                  className="afield"
                  type="text"
                  maxLength={3}
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                />
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 16 }}>
              {t('exp_date')}
            </label>
            <input
              className="afield"
              type="date"
              value={incurredAt}
              onChange={(e) => setIncurredAt(e.target.value)}
            />
          </div>
        </div>

        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('exp_section_detail')}</h3>
            <label className="alabel">{t('exp_category')}</label>
            <select
              className="select"
              style={{ width: '100%' }}
              value={category}
              onChange={(e) => setCategory(e.target.value as ExpenseCategory)}
            >
              {CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {t(`exp_cat_${c}`)}
                </option>
              ))}
            </select>

            {category === 'other' && (
              <>
                <label className="alabel" style={{ marginTop: 16 }}>
                  {t('exp_category_label')}
                </label>
                <input
                  className="afield"
                  type="text"
                  value={categoryOther}
                  placeholder={t('exp_category_label_ph')}
                  onChange={(e) => setCategoryOther(e.target.value)}
                />
              </>
            )}

            <label className="alabel" style={{ marginTop: 16 }}>
              {t('exp_note')}
            </label>
            <textarea
              className="afield"
              rows={3}
              value={note}
              placeholder={t('exp_note_ph')}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
