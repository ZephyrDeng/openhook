<script>
  import { 
    FileText, Route, Blocks, KeyRound, Truck, Filter, CopyX, Settings,
    ChevronLeft, ChevronRight, LogOut, Shield, UserCircle
  } from 'lucide-svelte'

  let {
    currentPage = $bindable('templates'),
    collapsed = $bindable(false),
    authState = null,
    onLogout = () => {},
  } = $props()

  const navItems = [
    { id: 'templates', label: '消息模板', icon: FileText },
    { id: 'routes', label: '路由', icon: Route },
    { id: 'middlewares', label: '中间件', icon: Blocks },
    { id: 'tokens', label: '令牌', icon: KeyRound },
    { id: 'deliveries', label: '投递日志', icon: Truck },
    { id: 'filters', label: '过滤器', icon: Filter },
    { id: 'dedup', label: '去重规则', icon: CopyX },
  ]

  let currentUser = $derived(authState?.user || {})
  let displayName = $derived(authState?.admin ? '管理员' : (currentUser.name || currentUser.login || 'GitHub 用户'))
</script>

<aside class="flex flex-col h-screen bg-[var(--color-bg-secondary)] border-r border-[var(--color-border-subtle)] transition-all duration-250" style:width={collapsed ? '64px' : '240px'}>
  <!-- Logo -->
  <div class="flex items-center gap-3 h-14 px-4 border-b border-[var(--color-border-subtle)] flex-shrink-0">
    <div class="w-8 h-8 rounded-lg bg-[var(--color-accent)] flex items-center justify-center flex-shrink-0">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
    </div>
    {#if !collapsed}
      <span class="font-semibold text-sm text-[var(--color-text-primary)] whitespace-nowrap">OpenHook</span>
    {/if}
    <button
      class="ml-auto text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors flex-shrink-0"
      onclick={() => collapsed = !collapsed}
      title={collapsed ? '展开' : '收起'}
    >
      {#if collapsed}
        <ChevronRight size={16} />
      {:else}
        <ChevronLeft size={16} />
      {/if}
    </button>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 py-3 px-2 overflow-y-auto">
    {#each navItems as item}
      <button
        class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-all duration-150 mb-0.5 {currentPage === item.id ? 'bg-[var(--color-accent)]/8 text-[var(--color-accent)] font-medium' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)] hover:text-[var(--color-text-primary)]'}"
        onclick={() => currentPage = item.id}
        title={collapsed ? item.label : ''}
      >
        <item.icon size={18} strokeWidth={1.5} />
        {#if !collapsed}
          <span class="whitespace-nowrap">{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  {#if authState?.authenticated}
    <div class="p-2 border-t border-[var(--color-border-subtle)]">
      <div class="flex items-center gap-2 px-2 py-2 rounded-md bg-[var(--color-bg-primary)]">
        <div class="w-8 h-8 rounded-md bg-[var(--color-bg-tertiary)] flex items-center justify-center overflow-hidden flex-shrink-0">
          {#if currentUser.avatarUrl}
            <img src={currentUser.avatarUrl} alt="" class="w-full h-full object-cover" />
          {:else if authState.admin}
            <Shield size={16} class="text-[var(--color-accent)]" />
          {:else}
            <UserCircle size={16} class="text-[var(--color-text-secondary)]" />
          {/if}
        </div>
        {#if !collapsed}
          <div class="min-w-0 flex-1">
            <div class="text-xs font-medium text-[var(--color-text-primary)] truncate">{displayName}</div>
            <div class="text-[11px] text-[var(--color-text-tertiary)] truncate">{authState.admin ? 'Admin Token' : 'GitHub'}</div>
          </div>
          <button
            class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            onclick={onLogout}
            title="退出"
            aria-label="退出"
          >
            <LogOut size={14} />
          </button>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Settings -->
  <div class="p-2 border-t border-[var(--color-border-subtle)]">
    <button
      class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-all duration-150 {currentPage === 'settings' ? 'bg-[var(--color-accent)]/8 text-[var(--color-accent)] font-medium' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)] hover:text-[var(--color-text-primary)]'}"
      onclick={() => currentPage = 'settings'}
      title={collapsed ? '设置' : ''}
    >
      <Settings size={18} strokeWidth={1.5} />
      {#if !collapsed}
        <span class="whitespace-nowrap">设置</span>
      {/if}
    </button>
  </div>
</aside>
