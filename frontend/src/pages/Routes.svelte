<script>
  import { routes } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import Modal from '../components/Modal.svelte'
  import { Plus, Route, Pencil, Trash2 } from 'lucide-svelte'

  let { onEdit, onNew } = $props()

  let items = $state([])
  let loading = $state(true)
  let deletingItem = $state(null)
  let showDeleteModal = $state(false)

  async function load() {
    loading = true
    try {
      const res = await routes.list()
      items = res.data || []
    } catch (e) {
      toast.error('加载路由失败: ' + e.message)
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
      await routes.delete(deletingItem.routeId)
      toast.success('路由已删除')
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
  <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-subtle)]">
    <div>
      <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">路由</h1>
      <p class="text-sm text-[var(--color-text-secondary)] mt-0.5">配置模板与目标 Webhook 地址的映射关系</p>
    </div>
    <button class="btn btn-primary" onclick={onNew}>
      <Plus size={16} />
      新建路由
    </button>
  </div>

  <div class="flex-1 overflow-auto p-6">
    {#if loading}
      <div class="flex items-center justify-center h-64">
        <div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
      </div>
    {:else if items.length === 0}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="w-12 h-12 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-4">
          <Route size={24} class="text-[var(--color-text-tertiary)]" />
        </div>
        <p class="text-sm font-medium text-[var(--color-text-primary)]">还没有路由</p>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">创建一条路由将模板绑定到目标 Webhook 地址</p>
        <button class="btn btn-primary mt-4" onclick={onNew}>
          <Plus size={16} />
          新建路由
        </button>
      </div>
    {:else}
      <div class="card overflow-hidden">
        <table class="w-full">
          <thead>
            <tr>
              <th class="table-header">路由名称</th>
              <th class="table-header">模板</th>
              <th class="table-header">目标地址</th>
              <th class="table-header">模式</th>
              <th class="table-header">状态</th>
              <th class="table-header w-24">操作</th>
            </tr>
          </thead>
          <tbody>
            {#each items as item (item.routeId)}
              <tr class="table-row cursor-pointer" onclick={() => onEdit(item)}>
                <td class="table-cell">
                  <div class="font-medium text-[var(--color-text-primary)]">{item.name}</div>
                  <div class="text-xs text-[var(--color-text-tertiary)] mt-0.5 font-mono">{item.routeId}</div>
                </td>
                <td class="table-cell">
                  <div class="text-sm text-[var(--color-text-secondary)] font-mono">{item.templateId}</div>
                </td>
                <td class="table-cell">
                  <div class="flex flex-col gap-0.5">
                    {#each item.targetUrls.slice(0, 2) as url}
                      <div class="text-xs text-[var(--color-text-secondary)] truncate max-w-[200px]">{url}</div>
                    {/each}
                    {#if item.targetUrls.length > 2}
                      <div class="text-xs text-[var(--color-text-tertiary)]">+{item.targetUrls.length - 2} 更多</div>
                    {/if}
                  </div>
                </td>
                <td class="table-cell">
                  <span class="badge badge-success">{item.mode}</span>
                </td>
                <td class="table-cell">
                  {#if item.enabled}
                    <span class="badge badge-success">● 启用</span>
                  {:else}
                    <span class="badge badge-error">● 禁用</span>
                  {/if}
                </td>
                <td class="table-cell">
                  <div class="flex items-center gap-1">
                    <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors" onclick={(e) => { e.stopPropagation(); onEdit(item) }}>
                      <Pencil size={14} />
                    </button>
                    <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)] transition-colors" onclick={(e) => { e.stopPropagation(); confirmDelete(item) }}>
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
    确定要删除路由 <strong class="text-[var(--color-text-primary)]">{deletingItem?.name}</strong> 吗？此操作不可撤销。
  </div>
  <div class="flex justify-end gap-2 mt-5">
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>取消</button>
    <button class="btn btn-danger" onclick={doDelete}>删除</button>
  </div>
</Modal>
