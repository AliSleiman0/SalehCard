import i18n from '@/i18n/config'
import { formatDate } from '@/lib/utils'
import type { AdminKyc } from '../api/kyc'

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
  if (Number.isNaN(new Date(iso).getTime())) return iso
  return formatDate(iso)
}

/** Map an admin KYC submission to the flat view the list renders. Labels and
 *  dates are in the admin's language at the time of the call. */
export function adaptKyc(k: AdminKyc): KycView {
  return {
    id: k.id,
    fullName: k.fullName,
    contact: k.userContact,
    dateOfBirth: dateLabel(k.dateOfBirth),
    placeOfBirth: k.placeOfBirth,
    placeOfResidence: k.placeOfResidence,
    documentLabel: i18n.t(`kyc_doc_${k.documentType}`, { defaultValue: k.documentType }),
    documentNumber: k.documentNumber,
    documentFrontUrl: k.documentFrontUrl ?? '',
    documentBackUrl: k.documentBackUrl ?? '',
    status: k.status,
    rejectionReason: k.rejectionReason ?? '',
    submitted: dateLabel(k.createdAt),
    raw: k,
  }
}
