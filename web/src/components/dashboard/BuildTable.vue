<script setup lang="ts">
import { ref } from 'vue'
import { useBuilds } from '../../composables/useBuilds'
import { useDialogs } from '../../composables/useDialogs'
import { useLogViewer } from '../../composables/useLogViewer'
import type { JobSummary } from '../../types/api'
import KubectlDialog from '../dialogs/KubectlDialog.vue'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import BuildRow from './BuildRow.vue'

const { builds, finishedCount, refresh, remove, clearFinished } = useBuilds()
const { confirm, alert } = useDialogs()
const logViewer = useLogViewer()

const kubectlJob = ref<JobSummary | null>(null)

async function guarded(action: () => Promise<unknown>, failureTitle: string) {
  try {
    await action()
  } catch (err) {
    await alert(failureTitle, err instanceof Error ? err.message : String(err))
  }
}

async function onDelete(job: JobSummary) {
  const ok = await confirm({
    title: 'Delete build job?',
    message: 'Removes the job and its pod from the cluster. Built packages stay in the repository.',
    detail: job.name,
    confirmLabel: 'Delete',
    danger: true,
  })
  if (ok) await guarded(() => remove(job.name), 'Delete failed')
}

async function onClearFinished() {
  const n = finishedCount.value
  const ok = await confirm({
    title: 'Clear finished jobs?',
    message: `Removes ${n} succeeded/failed job${n === 1 ? '' : 's'} from the cluster. Running jobs and built packages are not affected.`,
    confirmLabel: 'Clear',
    danger: true,
  })
  if (ok) await guarded(clearFinished, 'Cleanup failed')
}
</script>

<template>
  <AppCard title="Build Pipeline History">
    <template #actions>
      <AppButton size="small" :disabled="finishedCount === 0" @click="onClearFinished">Clear finished</AppButton>
      <AppButton size="small" @click="refresh">Refresh</AppButton>
    </template>

    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Job Name</th>
            <th>Status</th>
            <th>Started</th>
            <th>Duration</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="builds.length === 0">
            <td colspan="5" class="empty">No recent build jobs</td>
          </tr>
          <BuildRow
            v-for="job in builds"
            :key="job.name"
            :job="job"
            :history="builds"
            @logs="logViewer.open(job.name)"
            @kubectl="kubectlJob = job"
            @delete="onDelete(job)"
          />
        </tbody>
      </table>
    </div>

    <KubectlDialog v-if="kubectlJob" :job="kubectlJob" @close="kubectlJob = null" />
  </AppCard>
</template>
