<script>
  import { templates } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import { ArrowLeft, Play, Save, RotateCcw } from 'lucide-svelte'

  let { template = null, onBack } = $props()

  let isEdit = $derived(!!template)

  function buildForm(t) {
    return {
      templateName: t?.templateName || '',
      msgType: t?.msgType || 'markdown',
      content: t?.content || '# {{data.title}}\n- severity: {{data.severity}}\n- service: {{data.service}}',
      simulation: t?.simulation ? JSON.stringify(t.simulation, null, 2) : '{\n  "title": "Test Alert",\n  "severity": "critical",\n  "service": "checkout"\n}',
    }
  }

  let form = $state(buildForm(null))

  let previewResult = $state(null)
  let previewLoading = $state(false)
  let saving = $state(false)

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

  // Debounced preview
  let previewTimer = null
  function schedulePreview() {
    if (previewTimer) clearTimeout(previewTimer)
    previewTimer = setTimeout(() => {
      if (form.content && form.simulation) doPreview()
    }, 400)
  }

  async function save() {
    saving = true
    try {
      const body = {
        templateName: form.templateName,
        msgType: form.msgType,
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

  // Auto-preview on mount
  $effect(() => {
    const next = buildForm(template)
    form.templateName = next.templateName
    form.msgType = next.msgType
    form.content = next.content
    form.simulation = next.simulation
  })

  $effect(() => {
    if (form.content) schedulePreview()
  })
</script>

<div class="flex flex-col h-full">
  <!-- Header -->
  <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-subtle)]">
    <div class="flex items-center gap-3">
      <button class="btn btn-ghost p-2" onclick={onBack} title="返回">
        <ArrowLeft size={18} />
      </button>
      <div>
        <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">{isEdit ? '编辑模板' : '新建模板'}</h1>
        <p class="text-sm text-[var(--color-text-secondary)] mt-0.5">{isEdit ? template.templateId : '创建一个新的消息模板'}</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button class="btn btn-secondary" onclick={onBack}>取消</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        <Save size={16} />
        {saving ? '保存中...' : '保存'}
      </button>
    </div>
  </div>

  <!-- Content -->
  <div class="flex-1 flex overflow-hidden">
    <!-- Left: Form -->
    <div class="flex-1 overflow-auto p-6 border-r border-[var(--color-border-subtle)]">
      <div class="max-w-2xl space-y-5">
        <!-- Name -->
        <div>
          <label for="template-name" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">模板名称</label>
          <input id="template-name" type="text" class="input" bind:value={form.templateName} placeholder="例如: generic-alert" />
        </div>

        <!-- MsgType -->
        <div>
          <label for="template-msg-type" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">消息类型</label>
          <select id="template-msg-type" class="input" bind:value={form.msgType}>
            <option value="markdown">markdown</option>
            <option value="text">text</option>
            <option value="html">html</option>
          </select>
        </div>

        <!-- Content -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label for="template-content" class="text-sm font-medium text-[var(--color-text-primary)]">模板内容</label>
            <span class="text-xs text-[var(--color-text-tertiary)]">支持 {'{{'}data.xxx{'}}'} 和 {'{{'}global.xxx{'}}'} 占位符</span>
          </div>
          <textarea
            id="template-content"
            class="input font-mono text-[13px] leading-[22px] resize-y"
            style="min-height: 160px; tab-size: 2;"
            bind:value={form.content}
            oninput={schedulePreview}
            placeholder={'# {{data.title}}\n- severity: {{data.severity}}'}
          ></textarea>
        </div>

        <!-- Simulation Data -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label for="template-simulation" class="text-sm font-medium text-[var(--color-text-primary)]">模拟数据 (JSON)</label>
            <button class="text-xs text-[var(--color-accent)] hover:underline" onclick={() => { form.simulation = '{\n  "title": "Test",\n  "severity": "info"\n}'; schedulePreview() }}>
              <RotateCcw size={12} class="inline mr-0.5" />重置
            </button>
          </div>
          <textarea
            id="template-simulation"
            class="input font-mono text-[13px] leading-[22px] resize-y"
            style="min-height: 120px;"
            bind:value={form.simulation}
            oninput={schedulePreview}
          ></textarea>
        </div>
      </div>
    </div>

    <!-- Right: Preview -->
    <div class="w-[420px] flex-shrink-0 flex flex-col bg-[var(--color-bg-secondary)]">
      <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border-subtle)]">
        <h3 class="text-sm font-medium text-[var(--color-text-primary)]">实时预览</h3>
        <button class="btn btn-ghost text-xs px-2 py-1" onclick={doPreview}>
          <Play size={12} />
          刷新
        </button>
      </div>

      <div class="flex-1 overflow-auto p-4 space-y-4">
        {#if previewLoading}
          <div class="flex items-center justify-center py-12">
            <div class="w-6 h-6 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
          </div>
        {:else if previewResult?.error}
          <div class="badge badge-error inline-flex">{previewResult.error}</div>
        {:else if previewResult}
          <!-- Rendered Content -->
          <div>
            <div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase mb-2">渲染结果</div>
            <div class="bg-white rounded-md border border-[var(--color-border-subtle)] p-4 text-sm">
              {#if form.msgType === 'markdown'}
                <div class="prose prose-sm max-w-none">
                  <!-- Simple markdown-like rendering -->
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
            <pre class="code-block">{envelopeJSON}</pre>
        {:else}
          <div class="flex flex-col items-center justify-center py-12 text-center">
            <Play size={24} class="text-[var(--color-text-tertiary)] mb-2" />
            <p class="text-sm text-[var(--color-text-secondary)]">输入模板内容和模拟数据后预览</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
