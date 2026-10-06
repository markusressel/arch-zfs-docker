import { ref } from 'vue'

/** Copy helper exposing a short-lived `copied` flag for button feedback. */
export function useClipboard(resetAfterMs = 1500) {
  const copied = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  async function copy(text: string) {
    await navigator.clipboard.writeText(text)
    copied.value = true
    clearTimeout(timer)
    timer = setTimeout(() => (copied.value = false), resetAfterMs)
  }

  return { copied, copy }
}
