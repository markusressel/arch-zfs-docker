<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger'
    size?: 'normal' | 'small'
    /** Stretch to the full width of the container. */
    block?: boolean
    type?: 'button' | 'submit'
    href?: string
    download?: boolean
  }>(),
  { variant: 'secondary', size: 'normal', type: 'button' },
)
</script>

<template>
  <component
    :is="href ? 'a' : 'button'"
    :href="href"
    :download="download || undefined"
    :type="href ? undefined : type"
    class="btn"
    :class="[variant, size, { block }]"
  >
    <slot />
  </component>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  text-decoration: none;
  transition: all 0.15s ease-in-out;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn.small {
  padding: 4px 8px;
  font-size: 0.75rem;
}

.btn.block {
  width: 100%;
}

.primary {
  background: var(--accent-green);
  color: #fff;
  border-color: rgba(240, 246, 252, 0.1);
}

.primary:hover:not(:disabled) {
  background: var(--accent-green-hover);
}

.secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-color: var(--border-color);
}

.secondary:hover:not(:disabled) {
  background: #30363d;
}

.danger {
  background: #da3633;
  color: #fff;
}

.danger:hover:not(:disabled) {
  background: var(--status-error);
}
</style>
