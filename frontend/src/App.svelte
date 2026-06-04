<script>
  import { onMount } from 'svelte'
  import Sidebar from './components/Sidebar.svelte'
  import Toast from './components/Toast.svelte'
  import Templates from './pages/Templates.svelte'
  import TemplateEditor from './pages/TemplateEditor.svelte'
  import Routes from './pages/Routes.svelte'
  import RouteEditor from './pages/RouteEditor.svelte'
  import Middlewares from './pages/Middlewares.svelte'
  import Tokens from './pages/Tokens.svelte'
  import Deliveries from './pages/Deliveries.svelte'
  import Settings from './pages/Settings.svelte'
  import { auth } from './stores/api.js'
  import { toast } from './stores/toast.js'
  import { Blocks, FileText, Github, KeyRound, LogIn, Route, Settings as SettingsIcon, Truck } from 'lucide-svelte'

  let currentPage = $state('templates')
  let sidebarCollapsed = $state(false)
  let editingTemplate = $state(null)
  let editingRoute = $state(null)
  let authLoading = $state(true)
  let authState = $state({ authenticated: false, authRequired: false, githubEnabled: false })
  let authError = $state('')
  let adminToken = $state('')
  let loggingIn = $state(false)
  let isLoginRoute = $state(false)
  const isAdmin = $derived(authState?.admin || authState?.authRequired === false)
  const mobileNavItems = $derived([
    { id: 'templates', label: '模板', icon: FileText },
    { id: 'routes', label: '路由', icon: Route },
    ...(isAdmin ? [
      { id: 'middlewares', label: '中间件', icon: Blocks },
      { id: 'tokens', label: '令牌', icon: KeyRound },
      { id: 'deliveries', label: '日志', icon: Truck },
    ] : []),
    { id: 'settings', label: '设置', icon: SettingsIcon },
  ])

  onMount(() => {
    adminToken = localStorage.getItem('openhook-token') || ''
    isLoginRoute = window.location.pathname === '/login'
    loadAuth()
  })

  async function loadAuth() {
    authLoading = true
    authError = ''
    try {
      const res = await auth.me()
      authState = res.data || { authenticated: false, authRequired: false, githubEnabled: false }
      if (authState.authRequired && !authState.authenticated) {
        moveTo('/login')
      } else if (isLoginRoute) {
        moveTo('/')
      }
    } catch (e) {
      authError = e.message
      authState = { authenticated: false, authRequired: true, githubEnabled: false }
      moveTo('/login')
    } finally {
      authLoading = false
    }
  }

  async function handleAdminLogin(event) {
    event.preventDefault()
    const token = adminToken.trim()
    if (!token) {
      authError = '请输入管理令牌'
      return
    }
    loggingIn = true
    authError = ''
    localStorage.setItem('openhook-token', token)
    try {
      const res = await auth.me()
      const next = res.data || { authenticated: false, authRequired: true, githubEnabled: authState.githubEnabled }
      if (!next.authenticated) {
        authError = '管理令牌无效'
        return
      }
      authState = next
      moveTo('/')
      toast.success('已登录')
    } catch (e) {
      authError = e.message
    } finally {
      loggingIn = false
    }
  }

  async function handleLogout() {
    try {
      await auth.logout()
      localStorage.removeItem('openhook-token')
      adminToken = ''
      authState = { authenticated: false, authRequired: authState.authRequired, githubEnabled: authState.githubEnabled }
      toast.success('已退出登录')
      moveTo(authState.authRequired ? '/login' : '/')
    } catch (e) {
      toast.error('退出失败: ' + e.message)
    }
  }

  function startGitHubLogin() {
    window.location.href = '/login/github?returnTo=/'
  }

  function moveTo(path) {
    if (window.location.pathname !== path) {
      window.history.replaceState({}, '', path)
    }
    isLoginRoute = path === '/login'
  }

  function handleNavigate(page) {
    currentPage = page
    editingTemplate = null
    editingRoute = null
  }

  function handleEditTemplate(template) {
    editingTemplate = template
    currentPage = 'template-editor'
  }

  function handleNewTemplate() {
    editingTemplate = null
    currentPage = 'template-editor'
  }

  function handleBackToTemplates() {
    editingTemplate = null
    currentPage = 'templates'
  }

  function handleEditRoute(route) {
    editingRoute = route
    currentPage = 'route-editor'
  }

  function handleNewRoute() {
    editingRoute = null
    currentPage = 'route-editor'
  }

  function handleBackToRoutes() {
    editingRoute = null
    currentPage = 'routes'
  }
</script>

<Toast />

{#if authLoading}
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)]">
    <div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if authState.authRequired && !authState.authenticated}
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)] px-4">
    <div class="w-full max-w-sm card">
      <div class="flex items-center gap-3 mb-6">
        <div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center">
          <KeyRound size={20} class="text-white" />
        </div>
        <div>
          <h1 class="text-lg font-semibold text-[var(--color-text-primary)]">登录 OpenHook</h1>
          <p class="text-sm text-[var(--color-text-secondary)]">{authState.githubEnabled ? 'GitHub 自动注册登录' : '管理令牌'}</p>
        </div>
      </div>
      {#if authError}
        <div class="mb-4 text-sm text-[var(--color-error)] bg-[var(--color-error-bg)] rounded-md px-3 py-2">{authError}</div>
      {/if}
      {#if authState.githubEnabled}
        <button class="btn btn-primary w-full" onclick={startGitHubLogin}>
          <Github size={16} />
          GitHub 注册或登录
        </button>
        <div class="mt-2 text-xs text-[var(--color-text-tertiary)] font-mono">/login/github</div>
        <div class="flex items-center gap-3 my-4">
          <div class="h-px flex-1 bg-[var(--color-border-subtle)]"></div>
          <span class="text-xs text-[var(--color-text-tertiary)]">或</span>
          <div class="h-px flex-1 bg-[var(--color-border-subtle)]"></div>
        </div>
      {/if}
      <form class="space-y-3" onsubmit={handleAdminLogin}>
        <div>
          <label for="admin-token-login" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">管理令牌</label>
          <input
            id="admin-token-login"
            type="password"
            class="input"
            bind:value={adminToken}
            autocomplete="current-password"
            placeholder="X-OpenHook-Admin-Token"
          />
        </div>
        <button class="btn btn-primary w-full" type="submit" disabled={loggingIn}>
          <LogIn size={16} />
          {loggingIn ? '登录中' : '管理令牌登录'}
        </button>
      </form>
    </div>
  </div>
{:else}
  <div class="app-shell">
    <div class="desktop-sidebar">
      <Sidebar bind:currentPage bind:collapsed={sidebarCollapsed} {authState} onLogout={handleLogout} />
    </div>

    <main class="app-main">
      {#if currentPage === 'templates'}
        <Templates onEdit={handleEditTemplate} onNew={handleNewTemplate} />
      {:else if currentPage === 'template-editor'}
        <TemplateEditor template={editingTemplate} onBack={handleBackToTemplates} />
      {:else if currentPage === 'routes'}
        <Routes onEdit={handleEditRoute} onNew={handleNewRoute} />
      {:else if currentPage === 'route-editor'}
        <RouteEditor route={editingRoute} onBack={handleBackToRoutes} allowMiddlewares={authState.admin || authState.authRequired === false} />
      {:else if currentPage === 'middlewares'}
        <Middlewares />
      {:else if currentPage === 'tokens'}
        <Tokens />
      {:else if currentPage === 'deliveries'}
        <Deliveries />
      {:else if currentPage === 'settings'}
        <Settings />
      {/if}
    </main>

    <nav class="mobile-bottom-nav" aria-label="主导航">
      {#each mobileNavItems as item}
        <button
          class="mobile-bottom-nav-item {currentPage === item.id ? 'active' : ''}"
          onclick={() => handleNavigate(item.id)}
          aria-label={item.label}
          aria-current={currentPage === item.id ? 'page' : undefined}
        >
          <item.icon size={18} strokeWidth={1.8} />
          <span>{item.label}</span>
        </button>
      {/each}
    </nav>
  </div>
{/if}
