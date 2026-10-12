import { onScopeDispose } from 'vue'

/**
 * Calls `task` repeatedly until the calling component/scope is disposed. `intervalMs` is read before
 * every wait, so the rate can change with the state (fast while builds run). The task also runs
 * right away when the browser tab becomes visible again.
 */
export function usePolling(task: () => unknown, intervalMs: number | (() => number)): void {
  const next = () => (typeof intervalMs === 'function' ? intervalMs() : intervalMs)
  let timer: ReturnType<typeof setTimeout>

  const schedule = () => {
    timer = setTimeout(() => {
      if (!document.hidden) task()
      schedule()
    }, next())
  }
  const onVisible = () => {
    if (!document.hidden) {
      clearTimeout(timer)
      task()
      schedule()
    }
  }

  schedule()
  document.addEventListener('visibilitychange', onVisible)
  onScopeDispose(() => {
    clearTimeout(timer)
    document.removeEventListener('visibilitychange', onVisible)
  })
}
