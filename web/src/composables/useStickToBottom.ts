import { ref, type Ref } from 'vue'
import { isNearBottom } from '../utils/stickToBottom'

/**
 * "Smart" auto-scroll for a growing, scrollable element (terminal output):
 * follows new content while the user is at the bottom, stops following as soon as they
 * scroll up, and resumes when they return to the bottom or call `scrollToBottom`.
 *
 * Wire `onScroll` to the element's scroll event and call `contentChanged` after its content
 * was updated in the DOM (e.g. from a `watch(..., { flush: 'post' })`).
 */
export function useStickToBottom(el: Ref<HTMLElement | null>, threshold = 80) {
  const following = ref(true)

  function onScroll() {
    if (el.value) following.value = isNearBottom(el.value, threshold)
  }

  function scrollToBottom() {
    if (!el.value) return
    el.value.scrollTop = el.value.scrollHeight
    following.value = true
  }

  function contentChanged() {
    if (following.value) scrollToBottom()
  }

  return { following, onScroll, scrollToBottom, contentChanged }
}
