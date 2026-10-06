import { ref } from 'vue'

const jobName = ref<string | null>(null)

/** Which build job's log is currently shown in the log modal (null = closed). */
export function useLogViewer() {
  return {
    jobName,
    open: (name: string) => (jobName.value = name),
    close: () => (jobName.value = null),
  }
}
