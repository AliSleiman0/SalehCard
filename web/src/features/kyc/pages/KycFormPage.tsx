import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Panel, Button, Input, Segmented, FileUpload, useToast } from '@/components'
import { useKycProfile } from '../hooks/useKycProfile'
import { useSubmitKyc } from '../hooks/useSubmitKyc'
import { useUploadKycDocument } from '../hooks/useUploadKycDocument'
import { backRequired, type KycDocumentType } from '../types'

export default function KycFormPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const { data } = useKycProfile()
  const submit = useSubmitKyc()
  const uploadDoc = useUploadKycDocument()

  const prior = data?.submission
  const [fullName, setFullName] = useState(prior?.fullName ?? '')
  const [dob, setDob] = useState(prior?.dateOfBirth ?? '')
  const [placeOfBirth, setPlaceOfBirth] = useState(prior?.placeOfBirth ?? '')
  const [placeOfResidence, setPlaceOfResidence] = useState(prior?.placeOfResidence ?? '')
  const [docType, setDocType] = useState<KycDocumentType>(prior?.documentType ?? 'id_card')
  const [docNumber, setDocNumber] = useState(prior?.documentNumber ?? '')
  const [frontUrl, setFrontUrl] = useState<string | null>(prior?.documentFrontUrl ?? null)
  const [backUrl, setBackUrl] = useState<string | null>(prior?.documentBackUrl ?? null)
  const [uploadingSide, setUploadingSide] = useState<'front' | 'back' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)

  const needsBack = backRequired(docType)

  const pickDoc = (side: 'front' | 'back', file: File) => {
    setUploadError(null)
    setUploadingSide(side)
    uploadDoc.mutate(file, {
      onSuccess: (url) => {
        if (side === 'front') setFrontUrl(url)
        else setBackUrl(url)
        setUploadingSide(null)
      },
      onError: (err) => {
        setUploadError(err.message)
        setUploadingSide(null)
      },
    })
  }

  const send = () => {
    setError(null)
    if (!fullName.trim()) return setError(t('kyc_err_name'))
    if (!dob) return setError(t('kyc_err_dob'))
    if (!placeOfBirth.trim() || !placeOfResidence.trim()) return setError(t('kyc_err_place'))
    if (!docNumber.trim()) return setError(t('kyc_err_docnum'))
    if (!frontUrl) return setError(t('kyc_err_front'))
    if (needsBack && !backUrl) return setError(t('kyc_err_back'))

    submit.mutate(
      {
        fullName: fullName.trim(),
        dateOfBirth: dob,
        placeOfBirth: placeOfBirth.trim(),
        placeOfResidence: placeOfResidence.trim(),
        documentType: docType,
        documentNumber: docNumber.trim(),
        documentFrontUrl: frontUrl,
        documentBackUrl: needsBack ? (backUrl ?? undefined) : undefined,
      },
      {
        onSuccess: () => {
          toast(t('kyc_submitted'), 'cart')
          navigate('/kyc')
        },
        onError: (err) => setError(err.message || t('auth_failed')),
      },
    )
  }

  return (
    <div className="wrap" style={{ padding: '26px 0 50px', maxWidth: 560 }}>
      <div className="row" style={{ gap: 10, marginBottom: 16 }}>
        <a className="small clickable faint" onClick={() => navigate('/kyc')}>
          {t('kyc_gate_title')}
        </a>
        <span className="faint">/</span>
        <span className="small" style={{ fontWeight: 700 }}>
          {t('kyc_form_title')}
        </span>
      </div>
      <h1 className="h1" style={{ marginBottom: 8 }}>
        {t('kyc_form_title')}
      </h1>
      <p className="muted" style={{ marginBottom: 22 }}>
        {t('kyc_form_sub')}
      </p>

      <Panel style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Input label={t('kyc_full_name')} value={fullName} onChange={(e) => setFullName(e.target.value)} />
        <Input label={t('kyc_dob')} type="date" value={dob} onChange={(e) => setDob(e.target.value)} />
        <Input label={t('kyc_place_birth')} value={placeOfBirth} onChange={(e) => setPlaceOfBirth(e.target.value)} />
        <Input
          label={t('kyc_place_residence')}
          value={placeOfResidence}
          onChange={(e) => setPlaceOfResidence(e.target.value)}
        />

        <div className="col" style={{ gap: 8 }}>
          <span className="label" style={{ margin: 0 }}>
            {t('kyc_doc_type')}
          </span>
          <Segmented
            options={[
              { value: 'id_card' as const, label: t('kyc_doc_id') },
              { value: 'passport' as const, label: t('kyc_doc_passport') },
              { value: 'license' as const, label: t('kyc_doc_license') },
            ]}
            value={docType}
            onChange={(v) => setDocType(v)}
          />
        </div>

        <Input label={t('kyc_doc_number')} value={docNumber} onChange={(e) => setDocNumber(e.target.value)} />

        <FileUpload
          label={t('kyc_doc_front')}
          value={frontUrl}
          uploading={uploadingSide === 'front'}
          onSelect={(f) => pickDoc('front', f)}
          onClear={() => setFrontUrl(null)}
        />
        {needsBack && (
          <FileUpload
            label={t('kyc_doc_back')}
            value={backUrl}
            uploading={uploadingSide === 'back'}
            onSelect={(f) => pickDoc('back', f)}
            onClear={() => setBackUrl(null)}
          />
        )}
        {uploadError && (
          <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
            {uploadError}
          </span>
        )}

        {error && (
          <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
            {error}
          </span>
        )}

        <Button
          variant="primary"
          size="lg"
          block
          onClick={send}
          loading={submit.isPending}
          disabled={uploadingSide !== null}
        >
          <Icon name="shield" size={18} />
          {t('kyc_submit')}
        </Button>
      </Panel>
    </div>
  )
}
