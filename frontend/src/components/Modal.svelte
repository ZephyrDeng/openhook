<script>
  import { fade, scale } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'

  let { show = false, title = '', onclose, children } = $props()

  function handleKeydown(e) {
    if (e.key === 'Escape') onclose?.()
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if show}
  <div class="fixed inset-0 z-[100] flex items-center justify-center" role="dialog" aria-modal="true" aria-labelledby={title ? 'modal-title' : undefined}>
    <button
      type="button"
      class="absolute inset-0 bg-black/25 transition-opacity"
      aria-label="关闭弹窗"
      onclick={() => onclose?.()}
      transition:fade={{ duration: 200 }}
    ></button>
    <div
      class="relative bg-[var(--color-bg-secondary)] rounded-lg border border-[var(--color-border-subtle)] shadow-[0_8px_32px_rgba(0,0,0,0.12)] w-full max-w-lg mx-4"
      transition:scale={{ start: 0.96, duration: 200, easing: cubicOut }}
    >
      {#if title}
        <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border-subtle)]">
          <h3 id="modal-title" class="text-base font-semibold text-[var(--color-text-primary)]">{title}</h3>
          <button
            class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-tertiary)] transition-colors"
            aria-label="关闭弹窗"
            onclick={() => onclose?.()}
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
          </button>
        </div>
      {/if}
      <div class="p-5">
        {@render children?.()}
      </div>
    </div>
  </div>
{/if}
