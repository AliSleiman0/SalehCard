export type KycStatus = 'unverified' | 'pending' | 'verified' | 'rejected'

export type KycDocumentType = 'passport' | 'id_card' | 'license'

export interface KycSubmission {
  fullName: string
  dateOfBirth: string
  placeOfBirth: string
  placeOfResidence: string
  documentType: KycDocumentType
  documentNumber: string
  documentFrontUrl?: string
  documentBackUrl?: string
  status: string
  rejectionReason?: string
}

export interface KycProfile {
  status: KycStatus
  rejectionReason?: string
  submission?: KycSubmission
}

// KycSubmissionInput is the POST /api/v1/kyc body.
export interface KycSubmissionInput {
  fullName: string
  dateOfBirth: string // ISO yyyy-mm-dd
  placeOfBirth: string
  placeOfResidence: string
  documentType: KycDocumentType
  documentNumber: string
  documentFrontUrl: string
  documentBackUrl?: string
}

// Back photo is required for everything except a passport.
export function backRequired(docType: KycDocumentType): boolean {
  return docType !== 'passport'
}
