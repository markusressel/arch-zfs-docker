<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePackages } from '../../composables/usePackages'
import { useSettings } from '../../composables/useSettings'
import { KNOWN_VARIANTS, variantLabel } from '../../utils/jobs'
import { publishedModule } from '../../utils/packages'
import { formatFingerprint, installCommand, pacmanSnippet, trustKeyCommands } from '../../utils/pacman'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import CodeBlock from '../ui/CodeBlock.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'

const { settings } = useSettings()
const { packages } = usePackages()
const origin = import.meta.env.MODE === 'demo' ? 'https://zfs.example.com' : window.location.origin
const signing = computed(() => settings.value?.signing)
/** The published key; only then can clients require signatures. */
const key = computed(() => {
  const s = signing.value
  return s?.enabled && s.keyPath && s.fingerprint ? { path: s.keyPath, fingerprint: s.fingerprint } : null
})
const snippet = computed(() => pacmanSnippet(settings.value?.repoName, origin, !!key.value))
const keyCommands = computed(() => (key.value ? trustKeyCommands(origin, key.value.path, key.value.fingerprint) : ''))
/** Step numbers shift by one when the key step is shown. */
const step = (n: number) => (key.value ? n + 1 : n)

const variant = ref('')
const variants = KNOWN_VARIANTS.map((v) => ({
  value: v,
  label: variantLabel(v),
}))
const published = computed(() => publishedModule(packages.value, variant.value))
</script>

<template>
  <AppCard title="Use this repository">
    <template #subtitle>Install the ZFS packages built here on any Arch Linux machine with pacman.</template>
    <template #actions>
      <AppButton size="small" :to="{ name: 'dashboard' }">Package catalog</AppButton>
    </template>
    <ol class="steps">
      <li v-if="key" class="step">
        <div class="step-head">
          <span class="num">1</span>
          <div>
            <h3>Trust the signing key</h3>
            <p class="muted small">
              Before you run this, compare the fingerprint with your own record of the key:
              this page comes from the same server as the key.
            </p>
          </div>
        </div>
        <p class="small fingerprint">Fingerprint <code>{{ formatFingerprint(key.fingerprint) }}</code></p>
        <CodeBlock :text="keyCommands" lang="shell" caption="Terminal" />
      </li>
      <li class="step">
        <div class="step-head">
          <span class="num">{{ step(1) }}</span>
          <div>
            <h3>Add the repository</h3>
            <p class="muted small">Append this section to <code>/etc/pacman.conf</code>.</p>
          </div>
        </div>
        <CodeBlock :text="snippet" lang="ini" caption="/etc/pacman.conf" />
      </li>
      <li class="step">
        <div class="step-head">
          <span class="num">{{ step(2) }}</span>
          <div>
            <h3>Install ZFS for your kernel</h3>
            <p class="muted small">Pick the kernel you run (<code>uname -r</code> ends in <code>-lts</code> for linux-lts).</p>
          </div>
        </div>
        <SegmentedControl v-model="variant" :options="variants" label="Kernel" />
        <CodeBlock :text="installCommand(variant)" lang="shell" caption="Terminal" />
        <p v-if="published" class="small available">
          Available now:
          <strong>{{ published.packageName }} {{ published.version }}</strong>
          for kernel
          <RouterLink :to="{ name: 'dashboard', query: { q: published.kernelVersion } }" title="Show in the package catalog">
            <code>{{ published.kernelVersion }}</code>
          </RouterLink>
        </p>
        <p v-else class="small missing">
          No {{ variantLabel(variant) }} package is published yet. The upstream sync builds it, or
          <RouterLink :to="{ name: 'builds' }">start a build</RouterLink>.
        </p>
        <p class="muted small">Kernel updates then arrive with <code>sudo pacman -Syu</code> once the matching ZFS build is published.</p>
      </li>
    </ol>
    <p v-if="key" class="note small">
      Packages and database are signed (<code>SigLevel = Required</code>): pacman installs nothing without a valid
      signature of this key.
    </p>
    <p v-else-if="signing?.enabled" class="note small missing">
      Signing is configured, but no signed build has published the key yet. Start a build; until it is done, the
      repository can only be used unsigned.
    </p>
    <p v-else class="note small">The packages are not signed (<code>SigLevel = Optional TrustAll</code>): serve the repository over HTTPS or a network you trust.</p>
  </AppCard>
</template>

<style scoped>
.steps {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 24px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.step {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.step > :deep(.code-block) {
  align-self: stretch;
}

.step-head {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.num {
  flex: none;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(var(--v-theme-primary), 0.12);
  border: 1px solid rgba(var(--v-theme-primary), 0.4);
  color: var(--accent-blue);
  font-size: 0.75rem;
  font-weight: 600;
}

h3 {
  margin: 2px 0 2px;
  font-size: 0.95rem;
}

p {
  margin: 0;
}

.available {
  color: var(--text-secondary);
}

.available a,
.missing a {
  color: var(--accent-blue);
  text-decoration: none;
}

.available a:hover,
.missing a:hover {
  text-decoration: underline;
}

.available a code {
  color: inherit;
}

.missing {
  color: var(--status-warning);
}

.fingerprint {
  color: var(--text-secondary);
}

.fingerprint code {
  overflow-wrap: anywhere;
}

.note {
  margin-top: 20px;
  padding-top: 14px;
  border-top: 1px solid var(--border-color);
  color: var(--text-muted);
}
</style>
