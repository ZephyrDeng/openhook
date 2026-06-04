<script>
  import { toast } from '../stores/toast.js'
  import { X, CheckCircle, AlertCircle } from 'lucide-svelte'

  const icons = {
    success: CheckCircle,
    error: AlertCircle,
    info: AlertCircle,
  }

  const styles = {
    success: 'border-l-3 border-l-[var(--color-success)]',
    error: 'border-l-3 border-l-[var(--color-error)]',
    info: 'border-l-3 border-l-[var(--color-info)]',
  }
</script>

<div class="fixed top-4 right-4 z-[200] flex flex-col gap-2">
  {#each $toast as t (t.id)}
    <div
      class="flex items-center gap-3 px-4 py-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-subtle)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] min-w-[280px] max-w-[400px] transition-all duration-200 ease-out"
      style="animation: slideIn 200ms ease-out"
    >
      <svelte:component this={icons[t.type]} size={18} class={t.type === 'success' ? 'text-[var(--color-success)]' : t.type === 'error' ? 'text-[var(--color-error)]' : 'text-[var(--color-info)]'} />
      <span class="text-sm text-[var(--color-text-primary)] flex-1">{t.message}</span>
      <button class="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors" onclick={() => toast.remove(t.id)}>
        <X size={14} />
      </button>
    </div>
  {/each}
</div>

<style>
  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateX(100%);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }
</style>
