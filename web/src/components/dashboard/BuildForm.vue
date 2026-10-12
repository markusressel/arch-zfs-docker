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

      <FormField
        label="ZFS Version (advanced)"
        for="zfs-input"
        hint="Leave empty for the version currently packaged in the AUR. Other releases (e.g. 2.3.9) override the PKGBUILD and may fail to build if it changed."
      >
        <input id="zfs-input" v-model="zfsVersion" type="text" list="zfs-versions" autocomplete="off" placeholder="AUR default" />
        <datalist id="zfs-versions">
          <option v-for="v in zfsVersions" :key="v" :value="v" />
        </datalist>
      </FormField>

      <FormField label="Kernel Variant" for="variant-select">
        <select id="variant-select" v-model="variant">
          <option value="">Standard (linux)</option>
          <option value="lts">Long Term Support (linux-lts)</option>
        </select>
      </FormField>

      <FormField label="Kernel support">
        <label class="check">
          <input v-model="allowUnsupported" type="checkbox" />
          Build even if OpenZFS does not support this kernel yet
        </label>
        <template #hint>
          Off by default: OpenZFS refuses kernels newer than its release supports. Such modules are untested, and every client that
          syncs installs them. Only for testing.
        </template>
      </FormField>

      <AppButton type="submit" variant="primary" block :disabled="submitting">
        {{ submitting ? 'Starting job...' : 'Start Build Job' }}
      </AppButton>
    </form>
  </AppCard>
</template>

<style scoped>
.check {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  cursor: pointer;
}

.check input {
  accent-color: var(--status-warning);
}
</style>
