<script setup lang="ts">
import { ref } from 'vue'
import { triggerBuild } from '../../api/builds'
import { useBuilds } from '../../composables/useBuilds'
import { useDialogs } from '../../composables/useDialogs'
import { useKernelVersions } from '../../composables/useKernelVersions'
import { useLogViewer } from '../../composables/useLogViewer'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import FormField from '../ui/FormField.vue'

const kernelVersion = ref('')
const variant = ref('')
const submitting = ref(false)

const { versions } = useKernelVersions(variant)
const { refresh } = useBuilds()
const { confirm, alert } = useDialogs()
const logViewer = useLogViewer()

async function start(overwrite: boolean) {
  submitting.value = true
  try {
    const result = await triggerBuild({ kernelVersion: kernelVersion.value.trim(), variant: variant.value, forceBuild: overwrite })
    if (result.kind === 'already-built') {
      const pkgName = variant.value ? `linux-${variant.value}` : 'linux'
      submitting.value = false
      const ok = await confirm({
        title: 'Overwrite existing packages?',
        message: `A build for ${pkgName} ${result.info.kernelVersion} already exists in the repository. Starting this job will rebuild and replace the existing files.`,
        detail: result.info.package.filename,
        confirmLabel: 'Rebuild & overwrite',
        danger: true,
      })
      if (ok) await start(true)
      return
    }
    await refresh()
    logViewer.open(result.job.name)
  } catch (err) {
    await alert('Failed to trigger build', err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AppCard title="Trigger ZFS Build">
    <form @submit.prevent="start(false)">
      <FormField
        label="Target Kernel Version"
        for="kernel-input"
        hint="Leave empty for latest, or pick/type a version (e.g. 7.2.8.arch1-2). Must exist in the Arch Linux Archive."
      >
        <input id="kernel-input" v-model="kernelVersion" type="text" list="kernel-versions" autocomplete="off" placeholder="Latest Arch official (default)" />
        <datalist id="kernel-versions">
          <option v-for="v in versions" :key="v" :value="v" />
        </datalist>
      </FormField>

      <FormField label="Kernel Variant" for="variant-select">
        <select id="variant-select" v-model="variant">
          <option value="">Standard (linux)</option>
          <option value="lts">Long Term Support (linux-lts)</option>
        </select>
      </FormField>

      <AppButton type="submit" variant="primary" block :disabled="submitting">
        {{ submitting ? 'Starting job...' : 'Start Build Job' }}
      </AppButton>
    </form>
  </AppCard>
</template>
