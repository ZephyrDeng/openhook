<script>
  import { toast } from '../stores/toast.js'
  import { Settings, KeyRound, Save } from 'lucide-svelte'

  let token = $state(localStorage.getItem('openhook-token') || '')
  let apiBase = $state(localStorage.getItem('openhook-api-base') || '')

  function save() {
    if (token.trim()) {
      localStorage.setItem('openhook-token', token.trim())
    } else {
      localStorage.removeItem('openhook-token')
    }
    if (apiBase.trim()) {
      localStorage.setItem('openhook-api-base', apiBase.trim())
    } else {
      localStorage.removeItem('openhook-api-base')
    }
    toast.success('设置已保存，刷新页面后生效')
  }
</script>

<div class="flex flex-col h-full">
  <div class="px-6 py-4 border-b border-[var(--color-border-subtle)]">
    <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">设置</h1>
    <p class="text-sm text-[var(--color-text-secondary)] mt-0.5">配置 API 连接和管理令牌</p>
  </div>

  <div class="flex-1 overflow-auto p-6">
    <div class="max-w-xl space-y-6">
      <div class="card space-y-5">
        <div class="flex items-center gap-2">
          <KeyRound size={18} class="text-[var(--color-accent)]" />
          <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">API 配置</h3>
        </div>

        <div>
          <label for="settings-admin-token" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">管理令牌 (X-OpenHook-Admin-Token)</label>
          <input id="settings-admin-token" type="password" class="input" bind:value={token} placeholder="输入你的管理令牌" />
          <p class="text-xs text-[var(--color-text-tertiary)] mt-1">设置后用于保护写 API 的鉴权。留空表示不发送鉴权头。</p>
        </div>

        <div>
          <label for="settings-api-base" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">API 基础地址</label>
          <input id="settings-api-base" type="text" class="input" bind:value={apiBase} placeholder="默认: 当前域名" />
          <p class="text-xs text-[var(--color-text-tertiary)] mt-1">例如: http://localhost:8080。留空则使用当前域名。</p>
        </div>

        <div class="pt-2">
          <button class="btn btn-primary" onclick={save}>
            <Save size={14} />
            保存设置
          </button>
        </div>
      </div>

      <div class="card">
        <h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-3">关于 OpenHook</h3>
        <div class="space-y-2 text-sm text-[var(--color-text-secondary)]">
          <p>OpenHook 是一个 Go 编写的 Webhook 转发服务，支持消息模板、路由转发、JavaScript 中间件、投递日志等功能。</p>
          <p>前端控制台使用 Svelte 5 + Tailwind CSS 构建，编译为纯静态文件。</p>
          <div class="flex items-center gap-4 mt-3 pt-3 border-t border-[var(--color-border-subtle)]">
            <span class="text-xs text-[var(--color-text-tertiary)]">版本: 0.1.0</span>
            <a href="https://github.com/ZephyrDeng/openhook" target="_blank" class="text-xs text-[var(--color-accent)] hover:underline">GitHub</a>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
