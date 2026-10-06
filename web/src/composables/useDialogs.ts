import { reactive } from 'vue'

export interface DialogOptions {
  title: string
  message: string
  /** Optional monospace detail line (e.g. a file name). */
  detail?: string
  confirmLabel?: string
  /** Style the confirm button as destructive. */
  danger?: boolean
}

interface DialogState extends DialogOptions {
  open: boolean
  /** When false only a single "Close" button is shown. */
  cancellable: boolean
}

const state = reactive<DialogState>({ open: false, cancellable: false, title: '', message: '' })
let resolver: ((confirmed: boolean) => void) | undefined

function show(options: DialogOptions, cancellable: boolean): Promise<boolean> {
  resolver?.(false)
  Object.assign(state, { detail: undefined, confirmLabel: undefined, danger: false }, options, { open: true, cancellable })
  return new Promise((resolve) => (resolver = resolve))
}

/** Promise based confirm/alert dialogs rendered by <DialogHost/>. */
export function useDialogs() {
  return {
    state,
    confirm: (options: DialogOptions) => show(options, true),
    alert: (title: string, message: string) => show({ title, message }, false).then(() => undefined),
    resolve(confirmed: boolean) {
      state.open = false
      resolver?.(confirmed)
      resolver = undefined
    },
  }
}
