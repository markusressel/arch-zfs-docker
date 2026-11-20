<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger'
    size?: 'normal' | 'small'
    /** Stretch to the full width of the container. */
    block?: boolean
    type?: 'button' | 'submit'
    href?: string
    /** Internal navigation target (renders a router link instead of a plain anchor). */
    to?: RouteLocationRaw
    download?: boolean
  }>(),
  { variant: 'secondary', size: 'normal', type: 'button' },
)

// A primary action is a filled green button, as on GitHub; the others are tonal.
const color = computed(() => (props.variant === 'primary' ? 'success' : props.variant === 'danger' ? 'error' : undefined))
</script>

<template>
  <v-btn
    :color="color"
    :variant="variant === 'primary' ? 'flat' : 'tonal'"
    :size="size === 'small' ? 'small' : 'default'"
    :block="block"
    :type="to || href ? undefined : type"
    :to="to"
    :href="to ? undefined : href"
    :download="download || undefined"
    class="text-none"
  >
    <slot />
  </v-btn>
</template>
