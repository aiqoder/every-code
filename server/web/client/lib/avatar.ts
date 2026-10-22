/** 将图片压缩为 128×128 JPEG data URL。 */
export async function compressAvatarToDataURL(file: File, size = 128, quality = 0.85): Promise<string> {
  const bitmap = await createImageBitmap(file)
  const canvas = document.createElement('canvas')
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    bitmap.close()
    throw new Error('无法创建画布')
  }

  const scale = Math.max(size / bitmap.width, size / bitmap.height)
  const w = bitmap.width * scale
  const h = bitmap.height * scale
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, size, size)
  ctx.drawImage(bitmap, (size - w) / 2, (size - h) / 2, w, h)
  bitmap.close()

  return canvas.toDataURL('image/jpeg', quality)
}
