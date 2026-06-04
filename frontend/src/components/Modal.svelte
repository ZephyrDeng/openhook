<script>
  let { show = false, title = '', onclose, children } = $props()

  function handleKeydown(e) {
    if (e.key === 'Escape') onclose?.()
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if show}
  <div class="fixed inset-0 z-[100] flex items-center justify-center" role="dialog" aria-modal="true" aria-labelledby={title ? 'modal-title' : undefined}>
    <button type="button" class="absolute inset-0 bg-black/30" aria-label="关闭弹窗" onclick={() => onclose?.()}></button>
    <div class="relative bg-[var(--color-bg-secondary)] rounded-lg border border-[var(--color-border-subtle)] shadow-[0_8px_32px_rgba(0,0,0,0.12)] w-full max-w-lg mx-4 animate-[fadeIn_200ms_ease-out]">
      {#if title}
        <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border-subtle)]">
          <h3 id="modal-title" class="text-base font-semibold text-[var(--color-text-primary)]">{title}</h3>
          <button class="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors" aria-label="关闭弹窗" onclick={() => onclose?.()}>
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

<style>
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.97); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
