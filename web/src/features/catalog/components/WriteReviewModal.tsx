import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal, Button, useToast } from '@/components'
import { StarInput } from './StarInput'
import { useSubmitReview } from '../hooks/useSubmitReview'

// WriteReviewModal collects a 1–5★ rating + optional note (the note field only
// appears for 1–2★, mirroring the app). ALREADY_REVIEWED closes with a toast and
// the CTA flips via the invalidated my-review query.
export function WriteReviewModal({
  productId,
  open,
  onClose,
}: {
  productId: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  const toast = useToast()
  const submit = useSubmitReview()
  const [rating, setRating] = useState(5)
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  const showNote = rating > 0 && rating <= 2

  const send = () => {
    setError(null)
    if (rating < 1) return setError(t('review_pick_rating'))
    submit.mutate(
      { productId, rating, body: body.trim() },
      {
        onSuccess: () => {
          toast(t('review_submitted'), 'cart')
          onClose()
        },
        onError: (err) => {
          if (err.code === 'ALREADY_REVIEWED') {
            toast(t('already_reviewed'), 'user')
            onClose()
            return
          }
          setError(err.message || t('auth_failed'))
        },
      },
    )
  }

  return (
    <Modal open={open} onClose={onClose} title={t('write_review')}>
      <div className="col" style={{ gap: 16 }}>
        <div className="col center" style={{ gap: 10 }}>
          <StarInput value={rating} onChange={setRating} />
        </div>
        {showNote && (
          <div className="col" style={{ gap: 6 }}>
            <label className="label">{t('review_note_label')}</label>
            <textarea
              className="field"
              rows={4}
              placeholder={t('review_note_ph')}
              value={body}
              onChange={(e) => setBody(e.target.value)}
              style={{ resize: 'vertical' }}
            />
          </div>
        )}
        {error && (
          <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
            {error}
          </span>
        )}
        <Button variant="primary" size="lg" block onClick={send} loading={submit.isPending}>
          {t('submit_review')}
        </Button>
      </div>
    </Modal>
  )
}
