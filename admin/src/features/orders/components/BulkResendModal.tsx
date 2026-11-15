import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Trans, useTranslation } from 'react-i18next'
import { Icon, Modal } from '@/components'
import { orderNumber } from '@/lib/orderNumber'
import { supplierErrorText, supplierText } from '../lib/supplierText'
import {
  listRerouteProviders,
  rerouteBulkOrders,
  MAX_BULK_RESEND,
  type BulkRerouteOutcome,
} from '../api/orders'

// Each outcome's badge colour; its label is the `ord_br_st_<status>` key.
const STATUS_CLASS: Record<BulkRerouteOutcome['status'], string> = {
  completed: 'st-ok',
  sent: 'st-ok',
  refused: 'st-danger',
  needs_ack: 'st-warn',
  no_link: 'st-warn',
  not_eligible: 'st-mute',
  conflict: 'st-mute',
  error: 'st-danger',
}

/** BulkResendModal re-sends up to MAX_BULK_RESEND selected orders to a supplier
 *  in one go (the old admin's bulk "re-order via API"): each order goes to its
 *  product's first usable linked supplier — or to its link at one chosen
 *  supplier — and the per-order outcome is listed. */
export function BulkResendModal({ ids, onClose, onDone }: { ids: string[]; onClose: () => void; onDone: () => void }) {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const { data: provRes } = useQuery({
    queryKey: ['admin', 'orders', 'reroute-providers'],
    queryFn: listRerouteProviders,
    staleTime: 5 * 60 * 1000,
  })
  const providers = provRes?.data ?? []
  const [providerId, setProviderId] = useState(0)
  const [understood, setUnderstood] = useState(false)
  const [ackAtSupplier, setAckAtSupplier] = useState(false)
  const [sendAnyway, setSendAnyway] = useState(false)
  const [error, setError] = useState('')
  const name = (id?: number) => providers.find((p) => p.id === id)?.name ?? (id ? `#${id}` : '')

  const run = useMutation({
    mutationFn: () =>
      rerouteBulkOrders({
        ids,
        ...(providerId ? { providerId } : {}),
        acknowledgeAtSupplier: ackAtSupplier,
        acknowledgeAmbiguous: ackAtSupplier,
        acknowledgeLowBalance: sendAnyway,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
      qc.invalidateQueries({ queryKey: ['admin', 'order'] })
    },
    onError: (e) => setError(supplierErrorText(e, 'ord_br_failed', t, i18n.language)),
  })
  const result = run.data?.data
  const tooMany = ids.length > MAX_BULK_RESEND

  return (
    <Modal onClose={result ? onDone : onClose} maxWidth={680}>
      <div style={{ padding: 22, maxHeight: '86vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {t('ord_br_title', { count: ids.length })}
        </h3>
        {!result ? (
          <>
            <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
              {t('ord_br_body')}
            </p>
            <label className="alabel">{t('ord_sp_supplier')}</label>
            <select
              className="afield"
              aria-label={t('ord_sp_supplier')}
              value={providerId}
              onChange={(e) => setProviderId(Number(e.target.value))}
            >
              <option value={0}>{t('ord_br_first_linked')}</option>
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {t('ord_br_only_linked', { name: p.name })}
                </option>
              ))}
            </select>
            <div className="ahint" style={{ marginTop: 6 }}>
              {providerId
                ? t('ord_br_hint_provider')
                : t('ord_br_hint_first')}
            </div>

            <div
              style={{
                marginTop: 14,
                padding: 12,
                borderRadius: 10,
                border: '1px solid var(--border-strong)',
                background: 'var(--surface-2)',
                fontSize: 13,
              }}
            >
              <b>{t('ord_sp_risk')}</b> {t('ord_br_risk_body')}
              <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
                <input type="checkbox" checked={understood} onChange={(e) => setUnderstood(e.target.checked)} />{' '}
                {t('ord_sp_understand')}
              </label>
              <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center', color: 'var(--danger)' }}>
                <input type="checkbox" checked={ackAtSupplier} onChange={(e) => setAckAtSupplier(e.target.checked)} />{' '}
                {t('ord_br_also_resend')}
              </label>
              <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center' }}>
                <input type="checkbox" checked={sendAnyway} onChange={(e) => setSendAnyway(e.target.checked)} />{' '}
                {t('ord_sp_low_balance_ok')}
              </label>
            </div>
            {tooMany && (
              <div role="alert" style={{ color: 'var(--danger)', fontSize: 13, marginTop: 10 }}>
                {t('ord_br_too_many', { n: MAX_BULK_RESEND })}
              </div>
            )}
            {error && (
              <div role="alert" style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 10 }}>
                {error}
              </div>
            )}
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn" onClick={onClose}>
                {t('cancel')}
              </button>
              <button
                className="abtn primary"
                disabled={!understood || tooMany || run.isPending}
                onClick={() => {
                  setError('')
                  run.mutate()
                }}
              >
                <Icon name="send" size={14} /> {run.isPending ? t('ord_sending') : t('ord_br_go', { n: ids.length })}
              </button>
            </div>
          </>
        ) : (
          <>
            <p style={{ fontSize: 13, marginBottom: 10 }}>
              <Trans
                i18nKey="ord_br_sent_of"
                values={{ sent: result.sent, total: result.total }}
                components={{ b: <b /> }}
              />
            </p>
            <table className="tbl" aria-label={t('ord_br_results_aria')} style={{ width: '100%', fontSize: 13 }}>
              <thead>
                <tr>
                  <th>{t('col_order')}</th>
                  <th>{t('ord_br_result')}</th>
                  <th>{t('ord_br_details')}</th>
                </tr>
              </thead>
              <tbody>
                {result.results.map((r) => {
                  const status = STATUS_CLASS[r.status] ? r.status : 'error'
                  return (
                    <tr key={r.id}>
                      <td className="mono">{orderNumber(r.id)}</td>
                      <td>
                        <span className={`st ${STATUS_CLASS[status]}`}>{t(`ord_br_st_${status}`)}</span>
                      </td>
                      <td>
                        {r.provider ? (
                          <span>
                            {name(r.provider)} <span className="faint mono">#{r.upstreamId}</span>
                            {r.message ? ' — ' : ''}
                          </span>
                        ) : null}
                        {r.message ? supplierText(r.message, t, i18n.language) : null}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn primary" onClick={onDone}>
                {t('ord_done')}
              </button>
            </div>
          </>
        )}
      </div>
    </Modal>
  )
}
