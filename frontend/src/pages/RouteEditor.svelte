<script>
  import { routes, templates, middlewares as mwApi } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import FormField from '../components/FormField.svelte'
  import { ArrowLeft, Copy, Save, Plus, Trash2, Send, X, MessagesSquare } from 'lucide-svelte'
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
  let touched = $state({})

  let saving = $state(false)
  let showTestModal = $state(false)
  let testData = $state('{\n  "title": "Test Alert",\n  "severity": "info",\n  "service": "test"\n}')
  let testResult = $state(null)
  let testLoading = $state(false)
  let testDataTouched = $state(false)

  const defaultTestData = {
    title: 'Test Alert',
    severity: 'info',
    service: 'test',
  }

  const providerProfiles = [
    {
      id: 'wecom-markdown',
      provider: 'wecom',
      label: '企微 Markdown',
      targetUrlHint: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...',
      mode: 'raw',
      fields: ['title', 'severity', 'service', 'environment', 'time', 'description'],
      sample: {
        title: 'OpenHook 告警',
        severity: 'info',
        service: 'openhook',
        environment: 'prod',
        time: '2026-06-04 00:00:00',
        description: 'provider smoke',
      },
      match: (tpl) => includesAll(tpl?.content, ['"msgtype":"markdown"', '"markdown"', '{{json data.text}}']),
    },
    {
      id: 'wecom-text',
      provider: 'wecom',
      label: '企微 Text',
      targetUrlHint: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...',
      mode: 'raw',
      fields: ['title', 'severity', 'service', 'environment', 'description', 'mentionedList', 'mentionedMobileList'],
      sample: {
        title: 'OpenHook 告警',
        severity: 'info',
        service: 'openhook',
        environment: 'prod',
        description: 'provider smoke',
        mentionedList: [],
        mentionedMobileList: [],
      },
      match: (tpl) => includesAll(tpl?.content, ['"msgtype":"text"', 'mentioned_list', 'mentioned_mobile_list']),
    },
    {
      id: 'telegram-html',
      provider: 'telegram',
      label: 'Telegram HTML',
      targetUrlHint: 'https://api.telegram.org/bot<TOKEN>/sendMessage',
      mode: 'raw',
      fields: ['chatId', 'title', 'severity', 'service', 'environment', 'description'],
      sample: {
        chatId: '123456789',
        title: 'OpenHook alert',
        severity: 'info',
        service: 'openhook',
        environment: 'prod',
        description: 'provider smoke',
      },
      match: (tpl) => includesAll(tpl?.content, ['"chat_id"', '"parse_mode":"HTML"', '"text"']),
    },
    {
      id: 'telegram-text',
      provider: 'telegram',
      label: 'Telegram Text',
      targetUrlHint: 'https://api.telegram.org/bot<TOKEN>/sendMessage',
      mode: 'raw',
      fields: ['chatId', 'text', 'title', 'severity', 'service', 'environment', 'description'],
      sample: {
        chatId: '123456789',
        title: 'OpenHook alert',
        severity: 'info',
        service: 'openhook',
        environment: 'prod',
        description: 'provider smoke',
      },
      match: (tpl) => includesAll(tpl?.content, ['"chat_id"', '"text"']) && !String(tpl?.content || '').includes('"parse_mode":"HTML"'),
    },
  ]

  function includesAll(value, needles) {
    const text = String(value || '')
    return needles.every((needle) => text.includes(needle))
  }

  function selectedTemplate() {
    return tplList.find((tpl) => tpl.templateId === form.templateId) || null
  }

  function selectedProviderProfile() {
    const tpl = selectedTemplate()
    if (!tpl) return null
    return providerProfiles.find((profile) => profile.match(tpl)) || null
  }

  const activeProvider = $derived(selectedProviderProfile())
  const targetUrlPlaceholder = $derived(activeProvider?.targetUrlHint || 'https://example.com/webhook')

  const publicWebhookUrl = $derived(
    isEdit && typeof window !== 'undefined'
      ? `${window.location.origin}/webhook/routes/${route.routeId}`
      : ''
  )

  const errors = $derived({
    name: touched.name && !form.name.trim() ? '请输入路由名称' : '',
    templateId: touched.templateId && !form.templateId ? '请选择消息模板' : '',
    targetUrls: touched.targetUrls && !form.targetUrls.some(u => u.trim()) ? '至少填写一个目标地址' : '',
  })

  const hasErrors = $derived(Object.values(errors).some(Boolean))

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
    touched.targetUrls = true
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

  function formatJSON(value) {
    return JSON.stringify(value, null, 2)
  }

  function syncProviderTestData(force = false) {
    if (!activeProvider) return
    if (!force && testDataTouched) return
    testData = formatJSON(activeProvider.sample)
    testDataTouched = false
  }

  function handleTemplateChange() {
    touched.templateId = true
    const profile = selectedProviderProfile()
    if (profile) {
      form.mode = profile.mode
      syncProviderTestData(true)
    }
  }

  function applyProviderDefaults() {
    if (!activeProvider) return
    form.mode = activeProvider.mode
    syncProviderTestData(true)
    toast.success(`已套用 ${activeProvider.label} 路由建议`)
  }

  async function save() {
    touched = { name: true, templateId: true, targetUrls: true }
    if (hasErrors) {
      toast.error('请填写必填字段')
      return
    }
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

  function openTestModal() {
    syncProviderTestData(!testDataTouched)
    testResult = null
    showTestModal = true
  }

  function responseSummary(result) {
    if (!activeProvider || !result?.response || typeof result.response !== 'object') {
      return ''
    }
    const resp = result.response
    if (activeProvider.provider === 'wecom' && resp.errcode !== undefined) {
      return `企微 errcode=${resp.errcode}${resp.errmsg ? ` · ${resp.errmsg}` : ''}`
    }
    if (activeProvider.provider === 'telegram' && resp.ok !== undefined) {
      return `Telegram ok=${resp.ok}${resp.description ? ` · ${resp.description}` : ''}`
    }
    return ''
  }

  function resultStatusLabel(result) {
    const summary = responseSummary(result)
    return summary || result.message
  }

  async function copyPublicWebhookUrl() {
    if (!publicWebhookUrl) return
    try {
      await navigator.clipboard.writeText(publicWebhookUrl)
      toast.success('对外 Webhook 地址已复制')
    } catch (e) {
      toast.error('复制失败: ' + e.message)
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
    touched = {}
    testData = formatJSON(defaultTestData)
    testDataTouched = false
  })

  const modeOptions = [
    { value: 'envelope', label: '包装消息 (envelope)', desc: '将内容包装在标准消息信封中发送' },
    { value: 'raw', label: '原始内容 (raw)', desc: '直接发送原始渲染内容' },
  ]
</script>

<div class="page-shell">
  <div class="page-header">
    <div class="flex items-center gap-3">
      <button class="btn btn-ghost p-2" onclick={onBack} title="返回列表" aria-label="返回列表">
        <ArrowLeft size={18} />
      </button>
      <div>
        <div class="breadcrumb">
          <button onclick={onBack}>路由</button>
          <span>/</span>
          <span>{isEdit ? '编辑' : '新建'}</span>
        </div>
        <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">{isEdit ? '编辑路由' : '新建路由'}</h1>
      </div>
    </div>
    <div class="desktop-actions">
      {#if isEdit}
        <button class="btn btn-secondary" onclick={openTestModal}>
          <Send size={16} />
          测试投递
        </button>
      {/if}
      <button class="btn btn-secondary" onclick={onBack}>取消</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        <Save size={16} />
        {saving ? '保存中' : '保存'}
      </button>
    </div>
  </div>

  <div class="page-content">
    <div class="max-w-2xl space-y-5">
      <FormField label="路由名称" forId="route-name" required error={errors.name}>
        <input
          id="route-name"
          type="text"
          class="input"
          bind:value={form.name}
          onblur={() => touched.name = true}
          placeholder="例如: generic-alert-route"
        />
      </FormField>

      <div class="form-section">
        <div class="route-webhook-card">
          <div>
            <span class="form-section-title">对外 Webhook 地址</span>
            <p class="route-webhook-desc">把这个地址提供给上游系统。对方使用 POST JSON 请求，OpenHook 会用当前路由的模板渲染消息并转发。</p>
          </div>
          {#if isEdit}
            <div class="route-webhook-copy-row">
              <code>{publicWebhookUrl}</code>
              <button class="btn btn-secondary" type="button" onclick={copyPublicWebhookUrl}>
                <Copy size={14} />
                复制地址
              </button>
            </div>
          {:else}
            <p class="route-webhook-pending">保存路由后生成 routeId，并在这里显示可复制的对外 Webhook 地址。</p>
          {/if}
          <p class="route-webhook-hint">下方的目标 Webhook 地址填写下游接收方，例如企业微信群机器人、Telegram 网关、QQ 网关或业务系统 Webhook。</p>
        </div>
      </div>

      <FormField label="消息模板" forId="route-template" required error={errors.templateId} helper="选择要使用的消息模板">
        <select id="route-template" class="input" bind:value={form.templateId} onchange={handleTemplateChange}>
          <option value="">选择模板...</option>
          {#each tplList as tpl}
            <option value={tpl.templateId}>{tpl.templateName} · {tpl.visibility === 'public' ? '公开' : '私有'} ({tpl.templateId})</option>
          {/each}
        </select>
      </FormField>

      {#if activeProvider}
        <div class="route-provider-panel">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <MessagesSquare size={16} class="text-[var(--color-accent)]" />
                <span class="text-sm font-semibold text-[var(--color-text-primary)]">{activeProvider.label}</span>
                <span class="badge badge-success">{activeProvider.mode}</span>
              </div>
              <div class="route-provider-hint">{activeProvider.targetUrlHint}</div>
            </div>
            <button type="button" class="btn btn-secondary compact" onclick={applyProviderDefaults}>
              套用建议
            </button>
          </div>
          <div class="route-provider-fields">
            {#each activeProvider.fields as field}
              <span>{field}</span>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Target URLs -->
      <div class="form-section">
        <div class="flex items-center justify-between mb-3">
          <span class="form-section-title">目标 Webhook 地址</span>
          <button class="text-xs text-[var(--color-accent)] hover:text-[var(--color-accent-hover)] flex items-center gap-0.5 transition-colors" onclick={addTargetUrl}>
            <Plus size={12} />添加地址
          </button>
        </div>
        <div class="space-y-2">
          {#each form.targetUrls as url, idx}
            <div class="dynamic-field-row">
              <input
                type="text"
                class="input flex-1"
                bind:value={form.targetUrls[idx]}
                aria-label={`目标 Webhook 地址 ${idx + 1}`}
                placeholder={targetUrlPlaceholder}
              />
              {#if form.targetUrls.length > 1}
                <button
                  class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-error)] transition-colors"
                  onclick={() => removeTargetUrl(idx)}
                  aria-label="删除此地址"
                >
                  <Trash2 size={14} />
                </button>
              {/if}
            </div>
          {/each}
        </div>
        {#if errors.targetUrls}
          <p class="form-error mt-1.5" role="alert">{errors.targetUrls}</p>
        {/if}
      </div>

      <!-- Headers -->
      <div class="form-section">
        <div class="flex items-center justify-between mb-3">
          <span class="form-section-title">自定义请求头</span>
          <button class="text-xs text-[var(--color-accent)] hover:text-[var(--color-accent-hover)] flex items-center gap-0.5 transition-colors" onclick={addHeader}>
            <Plus size={12} />添加请求头
          </button>
        </div>
        {#if form.headers.length === 0}
          <p class="text-sm text-[var(--color-text-tertiary)]">未配置自定义请求头</p>
        {:else}
          <div class="space-y-2">
            {#each form.headers as h, idx}
              <div class="dynamic-field-row stack-mobile">
                <input type="text" class="input flex-1" bind:value={form.headers[idx].key} aria-label={`请求头名称 ${idx + 1}`} placeholder="Header 名 (如 Content-Type)" />
                <input type="text" class="input flex-1" bind:value={form.headers[idx].value} aria-label={`请求头值 ${idx + 1}`} placeholder="Header 值" />
                <button
                  class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-error)] transition-colors"
                  onclick={() => removeHeader(idx)}
                  aria-label="删除此请求头"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      {#if allowMiddlewares}
        <div class="form-section">
          <span class="form-section-title">中间件</span>
          {#if mwList.length === 0}
            <p class="text-sm text-[var(--color-text-tertiary)]">暂无中间件，请先创建中间件</p>
          {:else}
            <div class="flex flex-wrap gap-2">
              {#each mwList as mw}
                <button
                  class="px-3 py-1.5 rounded-md text-xs font-medium border transition-all duration-150 {form.middlewareIds.includes(mw.middlewareId) ? 'bg-[var(--color-accent)]/10 border-[var(--color-accent)] text-[var(--color-accent)]' : 'bg-[var(--color-bg-primary)] border-[var(--color-border-default)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-default)]'}"
                  onclick={() => toggleMiddleware(mw.middlewareId)}
                  type="button"
                >
                  {mw.name}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Mode -->
      <FormField label="投递模式" helper="选择消息发送格式">
        <div class="mode-grid">
          {#each modeOptions as opt}
            <label class="flex items-start gap-2.5 px-3 py-2.5 rounded-md border border-[var(--color-border-default)] cursor-pointer hover:border-[var(--color-accent)] transition-colors duration-150 {form.mode === opt.value ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/5' : ''}">
              <input type="radio" bind:group={form.mode} value={opt.value} class="accent-[var(--color-accent)] mt-0.5" />
              <div>
                <span class="text-sm font-medium">{opt.label}</span>
                <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5">{opt.desc}</p>
              </div>
            </label>
          {/each}
        </div>
      </FormField>

      <div class="flex items-center gap-2 py-2">
        <input type="checkbox" id="enabled" bind:checked={form.enabled} class="w-4 h-4 accent-[var(--color-accent)]" />
        <label for="enabled" class="text-sm text-[var(--color-text-primary)]">启用此路由</label>
      </div>
    </div>
  </div>

  <div class="mobile-sticky-actions {isEdit ? 'three-actions' : ''}">
    {#if isEdit}
      <button class="btn btn-secondary" onclick={openTestModal}>
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
    {#if activeProvider}
      <div class="route-provider-panel compact-panel">
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="text-sm font-semibold text-[var(--color-text-primary)]">{activeProvider.label}</div>
            <div class="route-provider-hint">{activeProvider.targetUrlHint}</div>
          </div>
          <span class="badge badge-success">{activeProvider.mode}</span>
        </div>
      </div>
    {/if}

    <FormField label="测试数据 (JSON)" forId="route-test-data">
      <textarea
        id="route-test-data"
        class="input font-mono text-xs"
        style="min-height: 120px;"
        bind:value={testData}
        oninput={() => testDataTouched = true}
      ></textarea>
    </FormField>

    {#if testResult}
      <div>
        <span class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">投递结果</span>
        {#if testResult.error}
          <div class="badge badge-error">{testResult.error}</div>
        {:else}
          <div class="space-y-2">
            {#each testResult as result}
              <div class="flex items-center gap-2 text-sm flex-wrap">
                {#if result.code === 0}
                  <span class="badge badge-success">成功</span>
                {:else}
                  <span class="badge badge-error">失败</span>
                {/if}
                <span class="text-[var(--color-text-secondary)] break-all">{result.targetUrl}</span>
                <span class="text-[var(--color-text-tertiary)]">{resultStatusLabel(result)}</span>
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
