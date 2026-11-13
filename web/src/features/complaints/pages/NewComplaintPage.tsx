import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Card, FileUpload } from '@/components'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { useLocaleStore } from '@/stores/locale'
import {
  COMPLAINT_CATEGORIES,
  uploadComplaintPhoto,
  validateComplaint,
  type ComplaintCategory,
} from '../api/complaints'
import { useCreateComplaint } from '../hooks/useComplaints'

export default function NewComplaintPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const locale = useLocaleStore((s) => s.locale)
  const presetOrder = params.get('orderId') ?? ''
  const presetCategory = params.get('category') as ComplaintCategory | null

  const [category, setCategory] = useState<ComplaintCategory>(
    presetCategory && COMPLAINT_CATEGORIES.includes(presetCategory) ? presetCategory : presetOrder ? 'order' : 'other',
  )
  const [orderId, setOrderId] = useState(presetOrder)
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')
  const [photoUrl, setPhotoUrl] = useState('')
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const [error, setError] = useState('')

  const orders = useOrders()
  const recent = (orders.data ?? []).slice(0, 20)
  const create = useCreateComplaint()

  const upload = async (file: File) => {
    setUploadError('')
    setUploading(true)
    try {
      const res = await uploadComplaintPhoto(file)
      if (res.success && res.data?.url) setPhotoUrl(res.data.url)
      else setUploadError(res.error?.message ?? t('upload_failed'))
    } catch {
      setUploadError(t('upload_failed'))
    } finally {
      setUploading(false)
    }
  }

  const submit = () => {
    const key = validateComplaint({ subject, message })
    if (key) {
      setError(t(key))
      return
    }
    setError('')
    create.mutate(
      {
        category,
        subject: subject.trim(),
        message: message.trim(),
        orderId: category === 'order' && orderId ? orderId : undefined,
        photoUrl: photoUrl || undefined,
      },
      {
        onSuccess: (c) => navigate(`/complaints/${c.id}`, { replace: true }),
        onError: (e) => setError(e instanceof Error ? e.message : t('complaint_send_failed')),
      },
    )
  }

  return (
    <div className="wrap" style={{ padding: '26px 0 50px', maxWidth: 640 }}>
      <a
        className="small clickable faint"
        onClick={() => navigate('/complaints')}
        style={{ display: 'inline-block', marginBottom: 16 }}
      >
        ← {t('complaints')}
      </a>
      <h1 className="h1" style={{ marginBottom: 6 }}>
        {t('complaint_new_title')}
      </h1>
      <p className="small muted" style={{ marginBottom: 20 }}>
        {t('complaint_new_sub')}
      </p>
      <Card style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div className="col" style={{ gap: 8 }}>
          <span className="label">{t('complaint_category')}</span>
          <select className="field" value={category} onChange={(e) => setCategory(e.target.value as ComplaintCategory)}>
            {COMPLAINT_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {t('complaint_cat_' + c)}
              </option>
            ))}
          </select>
        </div>

        {category === 'order' && (
          <div className="col" style={{ gap: 8 }}>
            <span className="label">{t('complaint_order_optional')}</span>
            <select className="field" value={orderId} onChange={(e) => setOrderId(e.target.value)}>
              <option value="">{t('complaint_no_order')}</option>
              {presetOrder && !recent.some((o) => o.id === presetOrder) && (
                <option value={presetOrder}>#{presetOrder.slice(-8)}</option>
              )}
              {recent.map((o) => (
                <option key={o.id} value={o.id}>
                  #{o.id.slice(-8)} · {o.items[0]?.title?.[locale] || o.items[0]?.title?.en || ''} ·{' '}
                  {new Date(o.createdAt).toLocaleDateString()}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="col" style={{ gap: 8 }}>
          <span className="label">{t('complaint_subject')}</span>
          <input
            className="field"
            maxLength={120}
            value={subject}
            placeholder={t('complaint_subject_ph')}
            onChange={(e) => setSubject(e.target.value)}
          />
        </div>

        <div className="col" style={{ gap: 8 }}>
          <span className="label">{t('complaint_message')}</span>
          <textarea
            className="field"
            rows={6}
            maxLength={2000}
            value={message}
            placeholder={t('complaint_message_ph')}
            onChange={(e) => setMessage(e.target.value)}
            style={{ resize: 'vertical', minHeight: 120 }}
          />
        </div>

        <div className="col" style={{ gap: 8 }}>
          <span className="label">{t('complaint_photo_optional')}</span>
          <FileUpload
            label={t('complaint_upload_photo')}
            value={photoUrl || null}
            uploading={uploading}
            error={uploadError || undefined}
            onSelect={upload}
            onClear={() => setPhotoUrl('')}
          />
        </div>

        {error && (
          <div className="small" style={{ color: 'var(--danger)' }}>
            {error}
          </div>
        )}

        <Button variant="primary" block loading={create.isPending} disabled={uploading || create.isPending} onClick={submit}>
          {t('complaint_submit')}
        </Button>
      </Card>
    </div>
  )
}
