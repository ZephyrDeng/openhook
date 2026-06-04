<script>
  import { routes, templates, middlewares as mwApi } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import { ArrowLeft, Save, Plus, Trash2, Send, X } from 'lucide-svelte'
  import Modal from '../components/Modal.svelte'

  let { route = null, onBack, allowMiddlewares = false } = $props()

  let isEdit = $derived(!!route)

  let tplList = $state([])
  let mwList = $state([])

  function buildForm(r) {
    return {
      name: r?.name || '',
      templateId: r?.templateId || '',
      targetUrls: r?.targetUrls?.length ? [...r.targetUrls] : [''],
      headers: r?.headers ? Object.entries(r.headers).map(([k, v]) => ({ key: k, value: v })) : [],
      middlewareIds: r?.middlewareIds ? [...r.middlewareIds] : [],
      mode: r?.mode || 'envelope',
      enabled: r?.enabled ?? true,
    }
  }

  let form = $state(buildForm(null))

  let saving = $state(false)
  let showTestModal = $state(false)
  let testData = $state('{\n  "title": "Test Alert",\n  "severity": "info",\n  "service": "test"\n}')
  let testResult = $state(null)
  let testLoading = $state(false)

  async function loadRefs() {
    try {
      const [tRes, mRes] = await Promise.all([
        templates.list(),
        allowMiddlewares ? mwApi.list() : Promise.resolve({ data: [] }),
      ])
      tplList = tRes.data || []
      mwList = mRes.data || []
    } catch (e) {
      toast.error('加载引用数据失败')
    }
  }

  function addTargetUrl() {
    form.targetUrls = [...form.targetUrls, '']
  }

  function removeTargetUrl(idx) {
    form.targetUrls = form.targetUrls.filter((_, i) => i !== idx)
  }

  function addHeader() {
    form.headers = [...form.headers, { key: '', value: '' }]
  }

  function removeHeader(idx) {
    form.headers = form.headers.filter((_, i) => i !== idx)
  }

  function toggleMiddleware(id) {
    if (form.middlewareIds.includes(id)) {
      form.middlewareIds = form.middlewareIds.filter((m) => m !== id)
    } else {
      form.middlewareIds = [...form.middlewareIds, id]
    }
  }

  async function save() {
    saving = true
    try {
      const body = {
        name: form.name,
        templateId: form.templateId,
        targetUrls: form.targetUrls.filter((u) => u.trim()),
        headers: Object.fromEntries(form.headers.filter((h) => h.key.trim()).map((h) => [h.key, h.value])),
        middlewareIds: allowMiddlewares ? form.middlewareIds : [],
        mode: form.mode,
        enabled: form.enabled,
      }
      if (isEdit) {
        await routes.update(route.routeId, body)
        toast.success('路由已更新')
      } else {
        await routes.create(body)
        toast.success('路由已创建')
      }
      onBack()
    } catch (e) {
      toast.error('保存失败: ' + e.message)
    } finally {
      saving = false
    }
  }

  async function doTest() {
    if (!isEdit) {
      toast.error('请先保存路由')
      return
    }
    testLoading = true
    testResult = null
    try {
      const data = JSON.parse(testData)
      const res = await routes.deliver(route.routeId, data)
      testResult = res.data
    } catch (e) {
      testResult = { error: e.message }
    } finally {
      testLoading = false
    }
  }

  $effect(() => {
    loadRefs()
  })

  $effect(() => {
    const next = buildForm(route)
    form.name = next.name
    form.templateId = next.templateId
    form.targetUrls = next.targetUrls
    form.headers = next.headers
    form.middlewareIds = next.middlewareIds
    form.mode = next.mode
    form.enabled = next.enabled
  })
</script>

<div class="page-shell">
  <div class="page-header">
    <div class="flex items-center gap-3">
      <button class="btn btn-ghost p-2" onclick={onBack}>
        <ArrowLeft size={18} />
      </button>
      <div>
        <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">{isEdit ? '编辑路由' : '新建路由'}</h1>
      </div>
    </div>
    <div class="desktop-actions">
      {#if isEdit}
        <button class="btn btn-secondary" onclick={() => showTestModal = true}>
          <Send size={16} />
          测试投递
        </button>
      {/if}
      <button class="btn btn-secondary" onclick={onBack}>取消</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        <Save size={16} />
        {saving ? '保存中...' : '保存'}
      </button>
    </div>
  </div>

  <div class="page-content">
    <div class="max-w-2xl space-y-5">
      <!-- Name -->
      <div>
        <label for="route-name" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">路由名称</label>
        <input id="route-name" type="text" class="input" bind:value={form.name} placeholder="例如: generic-alert-route" />
      </div>

      <!-- Template -->
      <div>
        <label for="route-template" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">消息模板</label>
        <select id="route-template" class="input" bind:value={form.templateId}>
          <option value="">选择模板...</option>
          {#each tplList as tpl}
            <option value={tpl.templateId}>{tpl.templateName} · {tpl.visibility || 'private'} ({tpl.templateId})</option>
          {/each}
        </select>
      </div>

      <!-- Target URLs -->
      <div>
        <div class="flex items-center justify-between mb-1.5">
          <span class="text-sm font-medium text-[var(--color-text-primary)]">目标 Webhook 地址</span>
          <button class="text-xs text-[var(--color-accent)] hover:underline flex items-center gap-0.5" onclick={addTargetUrl}>
            <Plus size={12} />添加地址
          </button>
        </div>
        <div class="space-y-2">
          {#each form.targetUrls as url, idx}
            <div class="dynamic-field-row">
              <input type="text" class="input flex-1" bind:value={form.targetUrls[idx]} aria-label={`目标 Webhook 地址 ${idx + 1}`} placeholder="https://example.com/webhook" />
              {#if form.targetUrls.length > 1}
                <button class="icon-button p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]" onclick={() => removeTargetUrl(idx)}>
                  <Trash2 size={14} />
                </button>
              {/if}
            </div>
          {/each}
        </div>
      </div>

      <!-- Headers -->
      <div>
        <div class="flex items-center justify-between mb-1.5">
          <span class="text-sm font-medium text-[var(--color-text-primary)]">自定义请求头</span>
          <button class="text-xs text-[var(--color-accent)] hover:underline flex items-center gap-0.5" onclick={addHeader}>
            <Plus size={12} />添加请求头
          </button>
        </div>
        <div class="space-y-2">
          {#each form.headers as h, idx}
            <div class="dynamic-field-row stack-mobile">
              <input type="text" class="input flex-1" bind:value={form.headers[idx].key} aria-label={`请求头名称 ${idx + 1}`} placeholder="Header 名" />
              <input type="text" class="input flex-1" bind:value={form.headers[idx].value} aria-label={`请求头值 ${idx + 1}`} placeholder="Header 值" />
              <button class="icon-button p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]" onclick={() => removeHeader(idx)}>
                <Trash2 size={14} />
              </button>
            </div>
          {/each}
        </div>
      </div>

      {#if allowMiddlewares}
        <div>
          <span class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">中间件</span>
          {#if mwList.length === 0}
            <p class="text-sm text-[var(--color-text-tertiary)]">暂无中间件，请先创建中间件</p>
          {:else}
            <div class="flex flex-wrap gap-2">
              {#each mwList as mw}
                <button
                  class="px-3 py-1.5 rounded-md text-xs font-medium border transition-all {form.middlewareIds.includes(mw.middlewareId) ? 'bg-[var(--color-accent)]/10 border-[var(--color-accent)] text-[var(--color-accent)]' : 'bg-[var(--color-bg-primary)] border-[var(--color-border-default)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-default)]'}"
                  onclick={() => toggleMiddleware(mw.middlewareId)}
                >
                  {mw.name}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Mode -->
      <div>
        <span class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">投递模式</span>
        <div class="mode-grid">
          <label class="flex items-center gap-2 px-3 py-2 rounded-md border border-[var(--color-border-default)] cursor-pointer hover:border-[var(--color-accent)] transition-colors {form.mode === 'envelope' ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/5' : ''}">
            <input type="radio" bind:group={form.mode} value="envelope" class="accent-[var(--color-accent)]" />
            <span class="text-sm">envelope（包装消息）</span>
          </label>
          <label class="flex items-center gap-2 px-3 py-2 rounded-md border border-[var(--color-border-default)] cursor-pointer hover:border-[var(--color-accent)] transition-colors {form.mode === 'raw' ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/5' : ''}">
            <input type="radio" bind:group={form.mode} value="raw" class="accent-[var(--color-accent)]" />
            <span class="text-sm">raw（原始内容）</span>
          </label>
        </div>
      </div>

      <!-- Enabled -->
      <div class="flex items-center gap-2">
        <input type="checkbox" id="enabled" bind:checked={form.enabled} class="w-4 h-4 accent-[var(--color-accent)]" />
        <label for="enabled" class="text-sm text-[var(--color-text-primary)]">启用此路由</label>
      </div>
    </div>
  </div>

  <div class="mobile-sticky-actions {isEdit ? 'three-actions' : ''}">
    {#if isEdit}
      <button class="btn btn-secondary" onclick={() => showTestModal = true}>
        <Send size={16} />
        测试
      </button>
    {/if}
    <button class="btn btn-secondary" onclick={onBack}>取消</button>
    <button class="btn btn-primary primary-action" onclick={save} disabled={saving}>
      <Save size={16} />
      {saving ? '保存中' : '保存'}
    </button>
  </div>
</div>

<!-- Test Modal -->
<Modal show={showTestModal} title="测试投递" onclose={() => showTestModal = false}>
  <div class="space-y-4">
    <div>
      <label for="route-test-data" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">测试数据 (JSON)</label>
      <textarea id="route-test-data" class="input font-mono text-xs" style="min-height: 120px;" bind:value={testData}></textarea>
    </div>

    {#if testResult}
      <div>
        <span class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">投递结果</span>
        {#if testResult.error}
          <div class="badge badge-error">{testResult.error}</div>
        {:else}
          <div class="space-y-2">
            {#each testResult as result}
              <div class="flex items-center gap-2 text-sm">
                {#if result.code === 0}
                  <span class="badge badge-success">成功</span>
                {:else}
                  <span class="badge badge-error">失败</span>
                {/if}
                <span class="text-[var(--color-text-secondary)]">{result.targetUrl}</span>
                <span class="text-[var(--color-text-tertiary)]">{result.message}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <div class="flex justify-end gap-2">
      <button class="btn btn-secondary" onclick={() => showTestModal = false}>关闭</button>
      <button class="btn btn-primary" onclick={doTest} disabled={testLoading}>
        <Send size={14} />
        {testLoading ? '投递中...' : '投递测试'}
      </button>
    </div>
  </div>
</Modal>
