import { describe, it, expect, vi, beforeEach } from 'vitest'
import { uploadProductImage } from './products'
import { apiClient } from '@/lib/api-client'

vi.mock('@/lib/api-client', () => ({
  apiClient: { upload: vi.fn() },
}))

const uploadMock = vi.mocked(apiClient.upload)

describe('uploadProductImage', () => {
  beforeEach(() => uploadMock.mockReset())

  it('posts the file as multipart under the "image" field to the images endpoint', async () => {
    uploadMock.mockResolvedValue({
      success: true,
      data: { imageUrl: 'http://cdn/x.jpg', thumbnailUrl: 'http://cdn/x_thumb.jpg' },
    })

    const file = new File([new Uint8Array([1, 2, 3])], 'card.png', { type: 'image/png' })
    const res = await uploadProductImage(file)

    expect(uploadMock).toHaveBeenCalledTimes(1)
    const [path, form] = uploadMock.mock.calls[0]
    expect(path).toBe('/api/admin/products/images')
    expect(form).toBeInstanceOf(FormData)
    expect((form as FormData).get('image')).toBe(file)

    // Returns the envelope's URLs unchanged.
    expect(res.data).toEqual({ imageUrl: 'http://cdn/x.jpg', thumbnailUrl: 'http://cdn/x_thumb.jpg' })
  })
})
