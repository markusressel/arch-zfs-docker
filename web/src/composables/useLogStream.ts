import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { logStreamUrl } from '../api/builds'
import { appendBounded } from '../utils/logBuffer'

/**
 * Streams the log of a build job over SSE. Incoming lines are buffered and flushed once per
 * animation frame, and only the newest MAX_LOG_LINES are kept so a high-throughput
 * compilation cannot freeze the browser. The stream is closed when the job changes or the
 * scope is disposed (which also lets the server cancel the upstream Kubernetes request).
 */
export function useLogStream(jobName: Ref<string | null>) {
  const lines = shallowRef<string[]>([])
  const connected = ref(false)

  let source: EventSource | null = null
  let pending: string[] = []
  let frame = 0

  function flush() {
    frame = 0
    if (pending.length === 0) return
    lines.value = appendBounded(lines.value, pending)
    pending = []
  }

  function push(line: string) {
    pending.push(line)
    if (!frame) frame = requestAnimationFrame(flush)
  }

  function disconnect() {
    source?.close()
    source = null
    connected.value = false
    if (frame) cancelAnimationFrame(frame)
    frame = 0
    pending = []
  }

  function end(message: string) {
    push(message)
    flush()
    source?.close()
    source = null
    connected.value = false
  }

  function connect(name: string) {
    disconnect()
    lines.value = ['Connecting to log stream...']
    source = new EventSource(logStreamUrl(name))
    source.onopen = () => (connected.value = true)
    source.onmessage = (event) => push(event.data)
    source.addEventListener('close', () => end('\n[Stream ended]'))
    source.onerror = () => end('\n[Stream disconnected or job completed]')
  }

  watch(
    jobName,
    (name) => {
      if (name) connect(name)
      else {
        disconnect()
        lines.value = []
      }
    },
    { immediate: true },
  )

  onScopeDispose(disconnect)
  return { lines, connected }
}
