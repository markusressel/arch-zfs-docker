<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { triggerBuild } from '../../api/builds'
import { useBuilds } from '../../composables/useBuilds'
import { useDialogs } from '../../composables/useDialogs'
import { useKernelVersions } from '../../composables/useKernelVersions'
import { useLogViewer } from '../../composables/useLogViewer'
import { useZfsVersions } from '../../composables/useZfsVersions'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import FormField from '../ui/FormField.vue'

// The dashboard and analytics link here with ?kernel=...&variant=...&zfs=...&unsupported=1 to prefill the form.
const query = useRoute().query
const queryValue = (key: string) => (typeof query[key] === 'string' ? (query[key] as string) : '')

const kernelVersion = ref(queryValue('kernel'))
const zfsVersion = ref(queryValue('zfs'))
const variant = ref(queryValue('variant') === 'lts' ? 'lts' : '')
const allowUnsupported = ref(queryValue('unsupported') === '1')
const submitting = ref(false)
const variants = [
  { title: 'Standard (linux)', value: '' },
  { title: 'Long Term Support (linux-lts)', value: 'lts' },
]

const { versions } = useKernelVersions(variant)
const { versions: zfsVersions } = useZfsVersions()
const { refresh } = useBuilds()
const { confirm, alert } = useDialogs()
const logViewer = useLogViewer()

async function start(overwrite: boolean) {
  submitting.value = true
  try {
    const result = await triggerBuild({
      kernelVersion: kernelVersion.value.trim(),
      zfsVersion: zfsVersion.value.trim(),
      variant: variant.value,
      forceBuild: overwrite,
      allowUnsupportedKernel: allowUnsupported.value,
    })
    if (result.kind === 'already-built') {
      const pkgName = variant.value ? `linux-${variant.value}` : 'linux'
      submitting.value = false
      const ok = await confirm({
        title: 'Overwrite existing packages?',
        message: `A build for ${pkgName} ${result.info.kernelVersion}${zfsVersion.value.trim() ? ' with ZFS ' + zfsVersion.value.trim() : ''} already exists in the repository. Starting this job will rebuild and replace the existing files.`,
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
    <v-form class="d-flex flex-column ga-4" @submit.prevent="start(false)">
      <FormField hint="Leave empty for latest, or pick/type a version (e.g. 7.2.8.arch1-2). Must exist in the Arch Linux Archive.">
        <v-combobox
          id="kernel-input"
          :model-value="kernelVersion || null"
          :items="versions"
          label="Target kernel version"
          placeholder="Latest Arch official (default)"
          clearable
          @update:model-value="(v: string | null) => (kernelVersion = v ?? '')"
        />
      </FormField>

      <FormField
        hint="Leave empty for the version currently packaged in the AUR. Other releases (e.g. 2.3.9) override the PKGBUILD and may fail to build if it changed."
      >
        <v-combobox
          id="zfs-input"
          :model-value="zfsVersion || null"
          :items="zfsVersions"
          label="ZFS version (advanced)"
          placeholder="AUR default"
          clearable
          @update:model-value="(v: string | null) => (zfsVersion = v ?? '')"
        />
      </FormField>

      <v-select id="variant-select" v-model="variant" :items="variants" label="Kernel variant" />

      <FormField
        hint="Off by default: OpenZFS refuses kernels newer than its release supports. Such modules are untested, and every client that syncs installs them. Only for testing."
      >
        <v-checkbox v-model="allowUnsupported" label="Build even if OpenZFS does not support this kernel yet" />
      </FormField>

      <AppButton type="submit" variant="primary" block :disabled="submitting">
        {{ submitting ? 'Starting job...' : 'Start Build Job' }}
      </AppButton>
    </v-form>
  </AppCard>
</template>
