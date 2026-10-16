import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Modal, Button, useToast } from '@/components'
import { useDeleteAccount } from '../hooks/useDeleteAccount'
import { canForceDelete, forfeitBalance } from '../lib/deleteAccount'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useAuthStore } from '@/stores/auth'

// Maps the API's 409/403 delete-guard codes to friendly inline messages.
function messageForCode(code: string | undefined, fallback: string, t: (k: string) => string): string {
  switch (code) {
    case 'WALLET_NOT_EMPTY':
      return t('del_err_wallet')
    case 'ORDERS_IN_FLIGHT':
      return t('del_err_orders')
    case 'PAYMENTS_PENDING':
      return t('del_err_payments')
    case 'TOPUPS_PENDING':
      return t('del_err_topups')
    case 'ADMIN_ACCOUNT':
      return t('del_err_admin')
    case 'ACCOUNT_SUSPENDED':
      return t('del_err_suspended')
    default:
      return fallback
  }
}

// DeleteAccountModal is a two-stage confirm: consequences → type DELETE. A
// WALLET_NOT_EMPTY / ORDERS_IN_FLIGHT refusal additionally offers "Delete
// anyway", which opens an explicit forfeit confirmation (stage 2) and retries
// with force:true. Pending payments / top-ups never offer it.
export function DeleteAccountModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const del = useDeleteAccount()
  const [stage, setStage] = useState<0 | 1 | 2>(0)
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [errCode, setErrCode] = useState<string | undefined>(undefined)
  const walletErr = errCode === 'WALLET_NOT_EMPTY'
  const forceable = canForceDelete(errCode)

  // The live wallet balance (fallback: the session snapshot) for the forfeit
  // warning; only fetched once the forfeit stage is shown.
  const walletQuery = useWallet(open && stage === 2)
  const sessionBalance = useAuthStore((s) => s.user?.walletBalance)
  const balance = forfeitBalance(walletQuery.data?.balance ?? sessionBalance)

  const close = () => {
    setStage(0)
    setConfirm('')
    setError(null)
    setErrCode(undefined)
    onClose()
  }

  const runDelete = (force: boolean) => {
    setError(null)
    del.mutate(
      { force },
      {
        onSuccess: () => {
          toast(t('del_success'), 'user')
          close()
          navigate('/')
        },
        onError: (err) => {
          setErrCode(err.code)
          setError(messageForCode(err.code, err.message || t('auth_failed'), t))
          // A forced attempt that still fails (payment/top-up pending, network)
          // goes back to the explainer stage with its message.
          if (force) setStage(1)
        },
      },
    )
  }

  const doDelete = () => {
    if (confirm.trim() !== 'DELETE') return
    runDelete(false)
  }

  return (
    <Modal open={open} onClose={close} title={stage === 2 ? t('del_force_title') : t('delete_account')}>
      {stage === 2 ? (
        <div className="col" style={{ gap: 16 }}>
          <ul className="col" style={{ gap: 8, margin: 0, paddingInlineStart: 18 }}>
            <li className="small">
              {balance !== null
                ? t('del_force_balance_amount', { amount: `$${balance.toFixed(2)}` })
                : t('del_force_balance')}
            </li>
            <li className="small">{t('del_force_orders')}</li>
          </ul>
          <div className="row" style={{ gap: 10 }}>
            <Button variant="ghost" size="lg" block disabled={del.isPending} onClick={() => setStage(1)}>
              {t('back')}
            </Button>
            <Button variant="danger" size="lg" block loading={del.isPending} onClick={() => runDelete(true)}>
              {t('del_force_cta')}
            </Button>
          </div>
        </div>
      ) : stage === 0 ? (
        <div className="col" style={{ gap: 16 }}>
          <p className="small muted" style={{ margin: 0 }}>
            {t('del_intro')}
          </p>
          <ul className="col" style={{ gap: 8, margin: 0, paddingInlineStart: 18 }}>
            <li className="small">{t('del_point_permanent')}</li>
            <li className="small">{t('del_point_docs')}</li>
            <li className="small">{t('del_point_orders')}</li>
            <li className="small">{t('del_point_wallet')}</li>
          </ul>
          <div className="row" style={{ gap: 10 }}>
            <Button variant="ghost" size="lg" block onClick={close}>
              {t('cancel')}
            </Button>
            <Button variant="danger" size="lg" block onClick={() => setStage(1)}>
              {t('continue')}
            </Button>
          </div>
        </div>
      ) : (
        <div className="col" style={{ gap: 16 }}>
          <p className="small muted" style={{ margin: 0 }}>
            {t('del_type_confirm')}
          </p>
          <input
            className="field"
            autoFocus
            placeholder="DELETE"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          {error && (
            <div className="col" style={{ gap: 6 }}>
              <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
                {error}
              </span>
              {walletErr && (
                <a
                  className="tiny clickable"
                  style={{ fontWeight: 700, color: 'var(--brand-1)' }}
                  onClick={() => {
                    close()
                    navigate('/wallet')
                  }}
                >
                  {t('go_to_wallet')}
                </a>
              )}
            </div>
          )}
          <Button
            variant="danger"
            size="lg"
            block
            loading={del.isPending}
            disabled={confirm.trim() !== 'DELETE'}
            onClick={doDelete}
          >
            {t('delete_account')}
          </Button>
          {forceable && !del.isPending && (
            <Button variant="outline" size="lg" block onClick={() => setStage(2)} style={{ color: 'var(--danger)', borderColor: 'var(--danger)' }}>
              {t('del_anyway')}
            </Button>
          )}
        </div>
      )}
    </Modal>
  )
}
