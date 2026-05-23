export function resolveImageUrl(url: string | undefined | null): string {
  if (!url)
    return ''
  if (url.startsWith('@/assets/')) {
    return url.replace('@/assets/', '/assets/')
  }
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  return url
}
