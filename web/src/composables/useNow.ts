import { onScopeDispose, ref, type Ref } from 'vue'

const now = ref(Date.now())
let subscribers = 0
let timer: ReturnType<typeof setInterval> | undefined

/** Shared, once-per-second ticking timestamp (ms). The timer only runs while something uses it. */
export function useNow(): Ref<number> {
  if (subscribers++ === 0) {
    now.value = Date.now()
    timer = setInterval(() => (now.value = Date.now()), 1000)
  }
  onScopeDispose(() => {
    if (--subscribers === 0) clearInterval(timer)
  })
  return now
}
