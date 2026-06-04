<script>
  let {
    label = '',
    forId = '',
    helper = '',
    error = '',
    required = false,
    children,
  } = $props()
</script>

<div class="form-field" class:error={!!error}>
  {#if label}
    <label for={forId} class="form-label">
      {label}
      {#if required}<span class="form-required" aria-label="必填">*</span>{/if}
    </label>
  {/if}
  <div class="form-control">
    {@render children?.()}
  </div>
  {#if error}
    <p class="form-error" role="alert">{error}</p>
  {:else if helper}
    <p class="form-helper">{helper}</p>
  {/if}
</div>

<style>
  .form-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .form-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
    line-height: 1.4;
  }
  .form-required {
    color: var(--color-error);
    margin-left: 2px;
  }
  .form-control {
    display: flex;
    flex-direction: column;
  }
  .form-helper {
    font-size: 12px;
    line-height: 1.5;
    color: var(--color-text-tertiary);
  }
  .form-error {
    font-size: 12px;
    line-height: 1.5;
    color: var(--color-error);
    display: flex;
    align-items: center;
    gap: 4px;
    animation: shakeIn 200ms ease-out;
  }
  .form-field.error :global(.input),
  .form-field.error :global(select.input),
  .form-field.error :global(textarea.input) {
    border-color: var(--color-error);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-error) 12%, transparent);
  }
  @keyframes shakeIn {
    0% { opacity: 0; transform: translateX(-4px); }
    100% { opacity: 1; transform: translateX(0); }
  }
</style>
