import { onScopeDispose } from 'vue'

/** Calls `task` every `intervalMs` until the calling component/scope is disposed. */
export function usePolling(task: () => unknown, intervalMs: number): void {
  const timer = setInterval(task, intervalMs)
  onScopeDispose(() => clearInterval(timer))
}
