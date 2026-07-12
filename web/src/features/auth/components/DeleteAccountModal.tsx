import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Modal, Button, useToast } from '@/components'
import { useDeleteAccount } from '../hooks/useDeleteAccount'

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

// DeleteAccountModal is a two-stage confirm: consequences → type DELETE.
export function DeleteAccountModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const del = useDeleteAccount()
  const [stage, setStage] = useState<0 | 1>(0)
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [walletErr, setWalletErr] = useState(false)

  const close = () => {
    setStage(0)
    setConfirm('')
    setError(null)
    setWalletErr(false)
    onClose()
  }

  const doDelete = () => {
    if (confirm.trim() !== 'DELETE') return
    setError(null)
    del.mutate(undefined, {
      onSuccess: () => {
        toast(t('del_success'), 'user')
        close()
        navigate('/')
      },
      onError: (err) => {
        setWalletErr(err.code === 'WALLET_NOT_EMPTY')
        setError(messageForCode(err.code, err.message || t('auth_failed'), t))
      },
    })
  }

  return (
    <Modal open={open} onClose={close} title={t('delete_account')}>
      {stage === 0 ? (
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
        </div>
      )}
    </Modal>
  )
}
