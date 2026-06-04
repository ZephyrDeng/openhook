<script>
  import { toast } from '../stores/toast.js'
  import { X, CheckCircle, AlertCircle, Info } from 'lucide-svelte'
  import { fly, fade } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'

  const icons = {
    success: CheckCircle,
    error: AlertCircle,
    info: Info,
  }

  const iconColors = {
    success: 'text-[var(--color-success)]',
    error: 'text-[var(--color-error)]',
    info: 'text-[var(--color-info)]',
  }

  const borderColors = {
    success: 'border-l-[var(--color-success)]',
    error: 'border-l-[var(--color-error)]',
    info: 'border-l-[var(--color-info)]',
  }
</script>

<div class="fixed top-4 right-4 z-[200] flex flex-col gap-2 pointer-events-none">
  {#each $toast as t (t.id)}
    <div
      class="pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border-subtle)] border-l-[3px] {borderColors[t.type]} shadow-[0_4px_12px_rgba(0,0,0,0.08)] min-w-[280px] max-w-[400px]"
      in:fly={{ x: 40, duration: 250, easing: cubicOut }}
      out:fade={{ duration: 150 }}
    >
      <svelte:component this={icons[t.type]} size={18} class="flex-shrink-0 {iconColors[t.type]}" />
      <span class="text-sm text-[var(--color-text-primary)] flex-1">{t.message}</span>
      <button
        class="flex-shrink-0 p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-tertiary)] transition-colors"
        onclick={() => toast.remove(t.id)}
        aria-label="关闭通知"
      >
        <X size={14} />
      </button>
    </div>
  {/each}
</div>
