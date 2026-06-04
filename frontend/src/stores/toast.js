import { writable } from 'svelte/store'

let id = 0

function createToastStore() {
  const { subscribe, update } = writable([])

  return {
    subscribe,
    add: (message, type = 'info') => {
      const toastId = ++id
      update((items) => [...items, { id: toastId, message, type }])
      setTimeout(() => {
        update((items) => items.filter((t) => t.id !== toastId))
      }, 4000)
    },
    success: (message) => {
      const toastId = ++id
      update((items) => [...items, { id: toastId, message, type: 'success' }])
      setTimeout(() => {
        update((items) => items.filter((t) => t.id !== toastId))
      }, 4000)
    },
    error: (message) => {
      const toastId = ++id
      update((items) => [...items, { id: toastId, message, type: 'error' }])
      setTimeout(() => {
        update((items) => items.filter((t) => t.id !== toastId))
      }, 5000)
    },
    remove: (toastId) => {
      update((items) => items.filter((t) => t.id !== toastId))
    },
  }
}

export const toast = createToastStore()
