<script>
  import { templates } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import FormField from '../components/FormField.svelte'
  import { ArrowLeft, Save, RotateCcw } from 'lucide-svelte'

  let { template = null, onBack } = $props()

  let isEdit = $derived(!!template)

  function buildForm(t) {
    return {
      templateName: t?.templateName || '',
      msgType: t?.msgType || 'markdown',
      visibility: t?.visibility || 'private',
      content: t?.content || '# {{data.title}}\n- severity: {{data.severity}}\n- service: {{data.service}}',
      simulation: t?.simulation ? JSON.stringify(t.simulation, null, 2) : '{\n  "title": "Test Alert",\n  "severity": "critical",\n  "service": "checkout"\n}',
    }
  }

  let form = $state(buildForm(null))
  let touched = $state({})
  let previewResult = $state(null)
  let previewLoading = $state(false)
  let saving = $state(false)

  const errors = $derived({
    templateName: touched.templateName && !form.templateName.trim() ? '请输入模板名称' : '',
    content: touched.content && !form.content.trim() ? '请输入模板内容' : '',
    simulation: touched.simulation && !form.simulation.trim() ? '请输入模拟数据' : '',
  })

  const hasErrors = $derived(Object.values(errors).some(Boolean))

  async function doPreview() {
    previewLoading = true
    try {
      const simData = JSON.parse(form.simulation || '{}')
      const res = await templates.preview({
        content: form.content,
        simulation: simData,
        msgType: form.msgType,
      })
      previewResult = res.data
    } catch (e) {
      previewResult = { error: e.message }
    } finally {
      previewLoading = false
    }
  }

  let previewTimer = null
  function schedulePreview() {
    if (previewTimer) clearTimeout(previewTimer)
    previewTimer = setTimeout(() => {
      if (form.content && form.simulation) doPreview()
    }, 400)
  }

  async function save() {
    touched = { templateName: true, content: true, simulation: true }
    if (hasErrors) {
      toast.error('请填写必填字段')
      return
    }
    saving = true
    try {
      const body = {
        templateName: form.templateName,
        msgType: form.msgType,
        visibility: form.visibility,
        content: form.content,
        simulation: JSON.parse(form.simulation || '{}'),
      }
      if (isEdit) {
        await templates.update(template.templateId, body)
        toast.success('模板已更新')
      } else {
        await templates.create(body)
        toast.success('模板已创建')
      }
      onBack()
    } catch (e) {
      toast.error('保存失败: ' + e.message)
    } finally {
      saving = false
    }
  }

  function resetSimulation() {
    form.simulation = '{\n  "title": "Test",\n  "severity": "info"\n}'
    touched.simulation = false
    schedulePreview()
  }

  $effect(() => {
    const next = buildForm(template)
    form.templateName = next.templateName
    form.msgType = next.msgType
    form.visibility = next.visibility
    form.content = next.content
    form.simulation = next.simulation
    touched = {}
  })

  $effect(() => {
    if (form.content) schedulePreview()
  })

  const msgTypeOptions = [
    { value: 'markdown', label: 'Markdown' },
    { value: 'text', label: '纯文本 (text)' },
    { value: 'html', label: 'HTML' },
  ]

  const visibilityOptions = [
    { value: 'private', label: '私有' },
    { value: 'public', label: '公开' },
  ]
</script>

<div class="page-shell">
  <!-- Header -->
  <div class="page-header">
    <div class="flex items-center gap-3">
      <button class="btn btn-ghost p-2" onclick={onBack} title="返回列表" aria-label="返回列表">
        <ArrowLeft size={18} />
      </button>
      <div>
        <div class="breadcrumb">
          <button onclick={onBack}>消息模板</button>
          <span>/</span>
          <span>{isEdit ? '编辑' : '新建'}</span>
        </div>
        <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">{isEdit ? '编辑模板' : '新建模板'}</h1>
        {#if isEdit}
          <p class="text-sm text-[var(--color-text-secondary)] mt-0.5 font-mono">{template.templateId}</p>
        {/if}
      </div>
    </div>
    <div class="desktop-actions">
      <button class="btn btn-secondary" onclick={onBack}>取消</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        <Save size={16} />
        {saving ? '保存中' : '保存'}
      </button>
    </div>
  </div>

  <!-- Content -->
  <div class="editor-split">
    <!-- Left: Form -->
    <div class="editor-form-pane">
      <div class="max-w-2xl space-y-5">
        <FormField label="模板名称" forId="template-name" required error={errors.templateName}>
          <input
            id="template-name"
            type="text"
            class="input"
            bind:value={form.templateName}
            onblur={() => touched.templateName = true}
            placeholder="例如: generic-alert"
          />
        </FormField>

        <FormField label="消息类型" forId="template-msg-type" helper="选择消息渲染格式">
          <select id="template-msg-type" class="input" bind:value={form.msgType}>
            {#each msgTypeOptions as opt}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        </FormField>

        <FormField label="可见性" forId="template-visibility" helper="公开模板可被所有路由引用，私有模板仅限所有者使用">
          <select id="template-visibility" class="input" bind:value={form.visibility}>
            {#each visibilityOptions as opt}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        </FormField>

        <FormField
          label="模板内容"
          forId="template-content"
          required
          error={errors.content}
          helper={`支持 {{data.xxx}} 和 {{global.xxx}} 占位符，用于插入动态数据`}
        >
          <textarea
            id="template-content"
            class="input font-mono text-[13px] leading-[22px] resize-y"
            style="min-height: 160px; tab-size: 2;"
            bind:value={form.content}
            onblur={() => touched.content = true}
            oninput={schedulePreview}
            placeholder={'# {{data.title}}\n- severity: {{data.severity}}'}
          ></textarea>
        </FormField>

        <FormField
          label="模拟数据"
          forId="template-simulation"
          required
          error={errors.simulation}
          helper="JSON 格式的测试数据，用于实时预览模板渲染效果"
        >
          <div class="relative">
            <textarea
              id="template-simulation"
              class="input font-mono text-[13px] leading-[22px] resize-y"
              style="min-height: 120px;"
              bind:value={form.simulation}
              onblur={() => touched.simulation = true}
              oninput={schedulePreview}
            ></textarea>
            <button
              class="absolute top-2 right-2 text-xs text-[var(--color-accent)] hover:text-[var(--color-accent-hover)] bg-[var(--color-bg-primary)] px-2 py-1 rounded border border-[var(--color-border-subtle)] transition-colors"
              onclick={resetSimulation}
              type="button"
            >
              <RotateCcw size={12} class="inline mr-0.5" />重置
            </button>
          </div>
        </FormField>
      </div>
    </div>

    <!-- Right: Preview -->
    <div class="editor-preview-pane">
      <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border-subtle)] flex-shrink-0">
        <h3 class="text-sm font-medium text-[var(--color-text-primary)]">实时预览</h3>
        <span class="text-xs text-[var(--color-text-tertiary)]">自动更新</span>
      </div>

      <div class="flex-1 overflow-auto p-4 space-y-4">
        {#if previewLoading}
          <div class="flex items-center justify-center py-12">
            <div class="w-6 h-6 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
          </div>
        {:else if previewResult?.error}
          <div class="badge badge-error inline-flex">{previewResult.error}</div>
        {:else if previewResult}
          <div>
            <div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wide mb-2">渲染结果</div>
            <div class="bg-white rounded-md border border-[var(--color-border-subtle)] p-4 text-sm">
              {#if form.msgType === 'markdown'}
                <div class="prose prose-sm max-w-none">
                  {#each previewResult.split('\n') as line}
                    {#if line.startsWith('# ')}
                      <h1 class="text-lg font-bold mt-2 mb-1">{line.slice(2)}</h1>
                    {:else if line.startsWith('## ')}
                      <h2 class="text-base font-semibold mt-2 mb-1">{line.slice(3)}</h2>
                    {:else if line.startsWith('- ')}
                      <li class="ml-4">{line.slice(2)}</li>
                    {:else if line.trim() === ''}
                      <div class="h-2"></div>
                    {:else}
                      <p class="my-1">{line}</p>
                    {/if}
                  {/each}
                </div>
              {:else}
                <pre class="whitespace-pre-wrap font-mono text-xs">{previewResult}</pre>
              {/if}
            </div>
          </div>

          {@const envelopeJSON = JSON.stringify({
            msgType: form.msgType,
            content: previewResult,
            messageContent: '...',
            timestamp: Date.now(),
            requestId: 'req_xxx'
          }, null, 2)}
          <div>
            <div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wide mb-2">消息信封</div>
            <pre class="code-block">{envelopeJSON}</pre>
          </div>
        {:else}
          <div class="flex flex-col items-center justify-center py-12 text-center">
            <div class="w-10 h-10 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-3">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-text-tertiary)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
            </div>
            <p class="text-sm text-[var(--color-text-secondary)]">输入模板内容和模拟数据后<br/>将自动渲染预览</p>
          </div>
        {/if}
      </div>
    </div>
  </div>

  <div class="mobile-sticky-actions">
    <button class="btn btn-secondary" onclick={onBack}>取消</button>
    <button class="btn btn-primary" onclick={save} disabled={saving}>
      <Save size={16} />
      {saving ? '保存中' : '保存'}
    </button>
  </div>
</div>
