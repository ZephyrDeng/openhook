<script>
  import { templates } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import Modal from '../components/Modal.svelte'
  import { Plus, Search, Pencil, Trash2, FileText } from 'lucide-svelte'

  let { onEdit, onNew } = $props()

  let items = $state([])
  let loading = $state(true)
  let searchQuery = $state('')
  let deletingItem = $state(null)
  let showDeleteModal = $state(false)

  async function load() {
    loading = true
    try {
      const res = await templates.list(searchQuery)
      items = res.data || []
    } catch (e) {
      toast.error('加载模板失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function confirmDelete(item) {
    deletingItem = item
    showDeleteModal = true
  }

  async function doDelete() {
    if (!deletingItem) return
    try {
      await templates.delete(deletingItem.templateId)
      toast.success('模板已删除')
      load()
    } catch (e) {
      toast.error('删除失败: ' + e.message)
    } finally {
      showDeleteModal = false
      deletingItem = null
    }
  }

  $effect(() => {
    load()
  })
</script>

<div class="flex flex-col h-full">
  <!-- Header -->
  <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-subtle)]">
    <div>
      <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">消息模板</h1>
      <p class="text-sm text-[var(--color-text-secondary)] mt-0.5">管理 Webhook 消息模板和预览渲染效果</p>
    </div>
    <button class="btn btn-primary" onclick={onNew}>
      <Plus size={16} />
      新建模板
    </button>
  </div>

  <!-- Search -->
  <div class="px-6 py-3 border-b border-[var(--color-border-subtle)]">
    <div class="relative max-w-md">
      <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-tertiary)]" />
      <input
        type="text"
        class="input pl-9"
        placeholder="搜索模板..."
        bind:value={searchQuery}
        oninput={() => load()}
      />
    </div>
  </div>

  <!-- Content -->
  <div class="flex-1 overflow-auto p-6">
    {#if loading}
      <div class="flex items-center justify-center h-64">
        <div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
      </div>
    {:else if items.length === 0}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="w-12 h-12 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-4">
          <FileText size={24} class="text-[var(--color-text-tertiary)]" />
        </div>
        <p class="text-sm font-medium text-[var(--color-text-primary)]">还没有消息模板</p>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">创建一个模板来定义你的 Webhook 消息格式</p>
        <button class="btn btn-primary mt-4" onclick={onNew}>
          <Plus size={16} />
          新建模板
        </button>
      </div>
    {:else}
      <div class="card overflow-hidden">
        <table class="w-full">
          <thead>
            <tr>
              <th class="table-header">模板名称</th>
              <th class="table-header">类型</th>
              <th class="table-header">内容预览</th>
              <th class="table-header w-24">操作</th>
            </tr>
          </thead>
          <tbody>
            {#each items as item (item.templateId)}
              <tr class="table-row cursor-pointer" onclick={() => onEdit(item)}>
                <td class="table-cell">
                  <div class="font-medium text-[var(--color-text-primary)]">{item.templateName}</div>
                  <div class="text-xs text-[var(--color-text-tertiary)] mt-0.5 font-mono">{item.templateId}</div>
                </td>
                <td class="table-cell">
                  <span class="badge badge-success">{item.msgType}</span>
                </td>
                <td class="table-cell">
                  <div class="max-w-xs truncate text-[var(--color-text-secondary)]">{item.content}</div>
                </td>
                <td class="table-cell">
                  <div class="flex items-center gap-1">
                    <button
                      class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors"
                      onclick={(e) => { e.stopPropagation(); onEdit(item) }}
                      title="编辑"
                    >
                      <Pencil size={14} />
                    </button>
                    <button
                      class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)] transition-colors"
                      onclick={(e) => { e.stopPropagation(); confirmDelete(item) }}
                      title="删除"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<Modal show={showDeleteModal} title="确认删除" onclose={() => showDeleteModal = false}>
  <div class="text-sm text-[var(--color-text-secondary)]">
    确定要删除模板 <strong class="text-[var(--color-text-primary)]">{deletingItem?.templateName}</strong> 吗？此操作不可撤销。
  </div>
  <div class="flex justify-end gap-2 mt-5">
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>取消</button>
    <button class="btn btn-danger" onclick={doDelete}>删除</button>
  </div>
</Modal>
