<script>
  import { middlewares } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import Modal from '../components/Modal.svelte'
  import { Plus, Blocks, Pencil, Trash2, Save, X } from 'lucide-svelte'

  let items = $state([])
  let loading = $state(true)
  let editingId = $state(null)
  let form = $state({ name: '', code: '', enabled: true })
  let showDeleteModal = $state(false)
  let deletingItem = $state(null)

  async function load() {
    loading = true
    try {
      const res = await middlewares.list()
      items = res.data || []
    } catch (e) {
      toast.error('加载失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function startNew() {
    editingId = 'new'
    form = { name: '', code: 'if (ctx.severity === "debug") {\n  return { reject: true, message: "debug ignored" };\n}\nheaders["X-Source"] = "openhook";\nreturn true;', enabled: true }
  }

  function startEdit(item) {
    editingId = item.middlewareId
    form = { name: item.name, code: item.code, enabled: item.enabled }
  }

  function cancelEdit() {
    editingId = null
  }

  async function save() {
    try {
      if (editingId === 'new') {
        await middlewares.create(form)
        toast.success('中间件已创建')
      } else {
        await middlewares.update(editingId, form)
        toast.success('中间件已更新')
      }
      editingId = null
      load()
    } catch (e) {
      toast.error('保存失败: ' + e.message)
    }
  }

  function confirmDelete(item) {
    deletingItem = item
    showDeleteModal = true
  }

  async function doDelete() {
    try {
      await middlewares.delete(deletingItem.middlewareId)
      toast.success('已删除')
      load()
    } catch (e) {
      toast.error('删除失败: ' + e.message)
    } finally {
      showDeleteModal = false
      deletingItem = null
    }
  }

  $effect(() => { load() })
</script>

<div class="page-shell">
  <div class="page-header mobile-stack">
    <div>
      <h1 class="page-title">中间件</h1>
      <p class="page-description">JavaScript 中间件，在投递前处理请求数据</p>
    </div>
    <button class="btn btn-primary" onclick={startNew}>
      <Plus size={16} />
      新建中间件
    </button>
  </div>

  <div class="page-content">
    {#if loading}
      <div class="flex items-center justify-center h-64"><div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div></div>
    {:else if items.length === 0 && !editingId}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="w-12 h-12 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-4"><Blocks size={24} class="text-[var(--color-text-tertiary)]" /></div>
        <p class="text-sm font-medium text-[var(--color-text-primary)]">还没有中间件</p>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">创建 JavaScript 中间件来处理和过滤 webhook 数据</p>
        <button class="btn btn-primary mt-4" onclick={startNew}><Plus size={16} />新建中间件</button>
      </div>
    {:else}
      <div class="space-y-4">
        {#if editingId}
          <div class="card space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-medium text-[var(--color-text-primary)]">{editingId === 'new' ? '新建中间件' : '编辑中间件'}</h3>
              <button class="btn btn-ghost p-1" onclick={cancelEdit}><X size={16} /></button>
            </div>
            <div>
              <label for="middleware-name" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">名称</label>
              <input id="middleware-name" type="text" class="input" bind:value={form.name} placeholder="例如: drop-test" />
            </div>
            <div>
              <label for="middleware-code" class="block text-sm font-medium text-[var(--color-text-primary)] mb-1.5">代码 (JavaScript)</label>
              <textarea id="middleware-code" class="input font-mono text-xs" style="min-height: 200px;" bind:value={form.code}></textarea>
              <p class="text-xs text-[var(--color-text-tertiary)] mt-1">可用变量: ctx, global, headers。return true 继续，return false 或 {'{reject: true, message: "..."}'} 拒绝</p>
            </div>
            <div class="flex items-center gap-2">
              <input type="checkbox" id="mw-enabled" bind:checked={form.enabled} class="w-4 h-4 accent-[var(--color-accent)]" />
              <label for="mw-enabled" class="text-sm text-[var(--color-text-primary)]">启用</label>
            </div>
            <div class="flex justify-end gap-2">
              <button class="btn btn-secondary" onclick={cancelEdit}>取消</button>
              <button class="btn btn-primary" onclick={save}><Save size={14} />保存</button>
            </div>
          </div>
        {/if}

        {#each items as item (item.middlewareId)}
          <div class="card card-hover">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <span class="font-medium text-sm text-[var(--color-text-primary)]">{item.name}</span>
                {#if item.enabled}
                  <span class="badge badge-success">启用</span>
                {:else}
                  <span class="badge badge-error">禁用</span>
                {/if}
              </div>
              <div class="flex items-center gap-1">
                <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]" onclick={() => startEdit(item)}><Pencil size={14} /></button>
                <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)]" onclick={() => confirmDelete(item)}><Trash2 size={14} /></button>
              </div>
            </div>
            <pre class="code-block mt-3 text-xs">{item.code}</pre>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<Modal show={showDeleteModal} title="确认删除" onclose={() => showDeleteModal = false}>
  <div class="text-sm text-[var(--color-text-secondary)]">确定要删除中间件 <strong class="text-[var(--color-text-primary)]">{deletingItem?.name}</strong> 吗？</div>
  <div class="flex justify-end gap-2 mt-5">
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>取消</button>
    <button class="btn btn-danger" onclick={doDelete}>删除</button>
  </div>
</Modal>
