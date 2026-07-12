// downscaleImage shrinks a picked photo to <= maxDim on its longest side and
// re-encodes as JPEG, so phone camera shots don't blow the server's 10 MB cap.
// Falls back to the original file on any decode error (e.g. HEIC the browser
// can't read) — the server re-validates and re-encodes regardless.
export async function downscaleImage(
  file: File,
  maxDim = 2000,
  quality = 0.85,
): Promise<Blob> {
  try {
    const bitmap = await createImageBitmap(file)
    const scale = Math.min(1, maxDim / Math.max(bitmap.width, bitmap.height))
    if (scale >= 1) {
      bitmap.close()
      return file
    }
    const w = Math.round(bitmap.width * scale)
    const h = Math.round(bitmap.height * scale)
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      bitmap.close()
      return file
    }
    ctx.drawImage(bitmap, 0, 0, w, h)
    bitmap.close()
    const blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, 'image/jpeg', quality),
    )
    return blob ?? file
  } catch {
    return file
  }
}
