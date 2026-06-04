<script>
  import { toast } from '../stores/toast.js'
  import FormField from '../components/FormField.svelte'
  import { KeyRound, Globe, Save } from 'lucide-svelte'

  let token = $state(localStorage.getItem('openhook-token') || '')
  let apiBaseUrl = $state(localStorage.getItem('openhook-api-base') || '')
  let saving = $state(false)

  function save() {
    saving = true
    if (token.trim()) {
      localStorage.setItem('openhook-token', token.trim())
    } else {
      localStorage.removeItem('openhook-token')
    }
    if (apiBaseUrl.trim()) {
      localStorage.setItem('openhook-api-base', apiBaseUrl.trim())
    } else {
      localStorage.removeItem('openhook-api-base')
    }
    toast.success('设置已保存')
    saving = false
  }
</script>

<div class="page-shell">
  <div class="page-header">
    <div>
      <h1 class="page-title">设置</h1>
      <p class="page-description">配置 API 连接和管理令牌</p>
    </div>
  </div>

  <div class="page-content">
    <div class="max-w-xl space-y-5">
      <div class="card space-y-5 page-transition">
        <div class="flex items-center gap-2">
          <KeyRound size={18} class="text-[var(--color-accent)]" />
          <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">API 配置</h3>
        </div>

        <FormField
          label="管理令牌"
          forId="settings-admin-token"
          helper="X-OpenHook-Admin-Token，用于保护写 API 的鉴权。留空表示不发送鉴权头。"
        >
          <input id="settings-admin-token" type="password" class="input" bind:value={token} placeholder="输入你的管理令牌" />
        </FormField>

        <FormField
          label="API 基础地址"
          forId="settings-api-base"
          helper="留空则使用当前域名。例如: http://localhost:8080"
        >
          <input id="settings-api-base" type="text" class="input" bind:value={apiBaseUrl} placeholder="默认: 当前域名" />
        </FormField>

        <div class="pt-1">
          <button class="btn btn-primary" onclick={save} disabled={saving}>
            <Save size={14} />
            {saving ? '保存中' : '保存设置'}
          </button>
        </div>
      </div>

      <div class="card page-transition">
        <h3 class="text-sm font-semibold text-[var(--color-text-primary)] mb-2">关于 OpenHook</h3>
        <p class="text-sm text-[var(--color-text-secondary)] leading-relaxed">Webhook 转发服务，支持消息模板、路由转发、JavaScript 中间件、投递日志。</p>
        <div class="flex items-center gap-4 mt-3 pt-3 border-t border-[var(--color-border-subtle)]">
          <span class="text-xs text-[var(--color-text-tertiary)]">v0.1.0</span>
          <a href="https://github.com/ZephyrDeng/openhook" target="_blank" class="text-xs text-[var(--color-accent)] hover:underline transition-colors">GitHub</a>
        </div>
      </div>
    </div>
  </div>
</div>
