import { useMutation } from '@tanstack/react-query'
import { uploadKycDocument } from '../api/kyc'
import { downscaleImage } from '@/lib/downscaleImage'
import { ApiError, unwrap } from '@/lib/api-error'

// useUploadKycDocument downscales a picked photo then uploads it, returning the
// stored image URL. Rate-limited 10/min/IP server-side — surface errors inline.
export function useUploadKycDocument() {
  return useMutation<string, ApiError, File>({
    mutationFn: async (file) => {
      const blob = await downscaleImage(file)
      const res = await uploadKycDocument(blob)
      return unwrap(res, 'Upload failed').imageUrl
    },
  })
}
