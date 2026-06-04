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
  import Guide from './pages/Guide.svelte'
  import { auth } from './stores/api.js'
  import { toast } from './stores/toast.js'
  import { Blocks, BookOpen, CheckCircle2, FileText, Github, KeyRound, LogIn, Route, Settings as SettingsIcon, Truck, Workflow } from 'lucide-svelte'

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
    { id: 'guide', label: '指南', icon: BookOpen },
    ...(isAdmin ? [
      { id: 'middlewares', label: '中间件', icon: Blocks },
      { id: 'tokens', label: '令牌', icon: KeyRound },
      { id: 'deliveries', label: '日志', icon: Truck },
    ] : []),
    { id: 'settings', label: '设置', icon: SettingsIcon },
  ])

  // Page transition key
  let pageKey = $state(0)

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
    localStorage.removeItem('openhook-token')
    adminToken = ''
    window.location.href = '/auth/github/start?returnTo=/'
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
    pageKey++
  }

  function handleEditTemplate(template) {
    editingTemplate = template
    currentPage = 'template-editor'
    pageKey++
  }

  function handleNewTemplate() {
    editingTemplate = null
    currentPage = 'template-editor'
    pageKey++
  }

  function handleBackToTemplates() {
    editingTemplate = null
    currentPage = 'templates'
    pageKey++
  }

  function handleEditRoute(route) {
    editingRoute = route
    currentPage = 'route-editor'
    pageKey++
  }

  function handleNewRoute() {
    editingRoute = null
    currentPage = 'route-editor'
    pageKey++
  }

  function handleBackToRoutes() {
    editingRoute = null
    currentPage = 'routes'
    pageKey++
  }
</script>

<Toast />

{#if authLoading}
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)]">
    <div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if authState.authRequired && !authState.authenticated}
  <div class="login-hero page-transition">
    <section class="login-hero-content" aria-label="OpenHook 登录">
      <div class="login-hero-copy">
        <div class="login-brand-mark">
          <Workflow size={24} />
        </div>
        <p class="login-eyebrow">Webhook forwarding console</p>
        <h1>OpenHook</h1>
        <p class="login-lede">把告警 payload 渲染成可读消息，再按路由投递到企业微信、Telegram、QQ 或任意 Webhook。</p>
        <div class="login-flow" aria-label="核心链路">
          <div>
            <span>01</span>
            <strong>模板</strong>
            <small>{'{'}{'{'}data.title{'}'}{'}'}</small>
          </div>
          <div>
            <span>02</span>
            <strong>路由</strong>
            <small>routeId + targetUrls</small>
          </div>
          <div>
            <span>03</span>
            <strong>投递</strong>
            <small>delivery logs</small>
          </div>
        </div>
      </div>

      <div class="login-panel" aria-label="登录面板">
        <div class="login-panel-header">
          <div class="login-panel-icon">
            <KeyRound size={20} />
          </div>
          <div>
            <h2>登录控制台</h2>
            <p>{authState.githubEnabled ? 'GitHub OAuth 或管理令牌' : '管理令牌'}</p>
          </div>
        </div>

        {#if authError}
          <div class="login-error animate-[shakeIn_200ms_ease-out]">{authError}</div>
        {/if}

        {#if authState.githubEnabled}
          <button class="btn btn-primary login-primary-action" onclick={startGitHubLogin}>
            <Github size={16} />
            GitHub 登录
          </button>
          <div class="login-oauth-note">
            <CheckCircle2 size={14} />
            <span>登录完成后自动回到控制台</span>
          </div>
          <div class="login-divider">
            <span>或使用管理令牌</span>
          </div>
        {/if}

        <form class="login-token-form" onsubmit={handleAdminLogin}>
          <div>
            <label for="admin-token-login">管理令牌</label>
            <input
              id="admin-token-login"
              type="password"
              class="input"
              bind:value={adminToken}
              autocomplete="current-password"
              placeholder="X-OpenHook-Admin-Token"
            />
          </div>
          <button class="btn btn-secondary login-token-action" type="submit" disabled={loggingIn}>
            <LogIn size={16} />
            {loggingIn ? '登录中' : '令牌登录'}
          </button>
        </form>
      </div>
    </section>
  </div>
{:else}
  <div class="app-shell">
    <div class="desktop-sidebar">
      <Sidebar bind:currentPage bind:collapsed={sidebarCollapsed} {authState} onLogout={handleLogout} />
    </div>

    <main class="app-main">
      {#key pageKey}
        <div class="page-transition h-full">
          {#if currentPage === 'templates'}
            <Templates onEdit={handleEditTemplate} onNew={handleNewTemplate} />
          {:else if currentPage === 'template-editor'}
            <TemplateEditor template={editingTemplate} onBack={handleBackToTemplates} />
          {:else if currentPage === 'routes'}
            <Routes onEdit={handleEditRoute} onNew={handleNewRoute} />
          {:else if currentPage === 'route-editor'}
            <RouteEditor route={editingRoute} onBack={handleBackToRoutes} allowMiddlewares={authState.admin || authState.authRequired === false} />
          {:else if currentPage === 'guide'}
            <Guide />
          {:else if currentPage === 'middlewares'}
            <Middlewares />
          {:else if currentPage === 'tokens'}
            <Tokens />
          {:else if currentPage === 'deliveries'}
            <Deliveries />
          {:else if currentPage === 'settings'}
            <Settings />
          {/if}
        </div>
      {/key}
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
