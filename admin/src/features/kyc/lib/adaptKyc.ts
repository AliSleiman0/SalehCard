import type { AdminKyc, KycDocumentType } from '../api/kyc'

const DOC_LABEL: Record<KycDocumentType, string> = {
  passport: 'Passport',
  id_card: 'ID card',
  license: "Driver's license",
}

/** The flat shape the KYC list renders. */
export interface KycView {
  id: string
  fullName: string
  contact: string
  dateOfBirth: string
  placeOfBirth: string
  placeOfResidence: string
  documentLabel: string
  documentNumber: string
  documentFrontUrl: string
  documentBackUrl: string
  status: AdminKyc['status']
  rejectionReason: string
  submitted: string
  raw: AdminKyc
}

/** Short "Mon D, YYYY" label for an ISO date, or the raw value when unparseable. */
function dateLabel(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

/** Map an admin KYC submission to the flat view the list renders. */
export function adaptKyc(k: AdminKyc): KycView {
  return {
    id: k.id,
    fullName: k.fullName,
    contact: k.userContact,
    dateOfBirth: dateLabel(k.dateOfBirth),
    placeOfBirth: k.placeOfBirth,
    placeOfResidence: k.placeOfResidence,
    documentLabel: DOC_LABEL[k.documentType] ?? k.documentType,
    documentNumber: k.documentNumber,
    documentFrontUrl: k.documentFrontUrl ?? '',
    documentBackUrl: k.documentBackUrl ?? '',
    status: k.status,
    rejectionReason: k.rejectionReason ?? '',
    submitted: dateLabel(k.createdAt),
    raw: k,
  }
}
