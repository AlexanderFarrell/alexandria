import { onBeforeUnmount, ref, watch } from 'vue'
import { getCoverBlob } from '@/api/books'

export function useBookCover(
  bookId: () => string | undefined,
  hasCover: () => boolean,
) {
  const coverSrc = ref<string | null>(null)
  let objectUrl: string | null = null
  let requestId = 0

  function clearCover() {
    if (objectUrl) {
      URL.revokeObjectURL(objectUrl)
      objectUrl = null
    }
    coverSrc.value = null
  }

  watch([bookId, hasCover], async ([id, canLoad], _, onCleanup) => {
    requestId += 1
    const activeRequestId = requestId
    let cancelled = false

    onCleanup(() => {
      cancelled = true
    })

    clearCover()

    if (!id || !canLoad) return

    try {
      const blob = await getCoverBlob(id)
      if (cancelled || activeRequestId !== requestId) return

      objectUrl = URL.createObjectURL(blob)
      coverSrc.value = objectUrl
    } catch {
      if (!cancelled && activeRequestId === requestId) {
        coverSrc.value = null
      }
    }
  }, { immediate: true })

  onBeforeUnmount(() => {
    clearCover()
  })

  return coverSrc
}
