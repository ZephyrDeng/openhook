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

  const modeLabel = (m) => {
    if (m === 'envelope') return '包装消息'
    if (m === 'raw') return '原始内容'
    return m
  }
</script>

<div class="page-shell">
  <div class="page-header mobile-stack">
    <div>
      <h1 class="page-title">路由</h1>
      <p class="page-description">配置模板与目标 Webhook 地址的映射关系</p>
    </div>
    <button class="btn btn-primary" onclick={onNew}>
      <Plus size={16} />
      新建路由
    </button>
  </div>

  <div class="page-content">
    {#if loading}
      <div class="flex items-center justify-center h-64">
        <div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
      </div>
    {:else if items.length === 0}
      <div class="empty-state">
        <div class="empty-state-icon">
          <Route size={24} />
        </div>
        <p class="empty-state-title">还没有路由</p>
        <p class="empty-state-desc">创建一条路由将模板绑定到目标 Webhook 地址</p>
        <button class="btn btn-primary" onclick={onNew}>
          <Plus size={16} />
          新建路由
        </button>
      </div>
    {:else}
      <div class="card desktop-table-card">
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
            {#each items as item, i (item.routeId)}
              <tr class="table-row cursor-pointer" onclick={() => onEdit(item)} style="animation-delay: {i * 30}ms">
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
                  <span class="badge badge-success">{modeLabel(item.mode)}</span>
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
                    <button
                      class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors"
                      onclick={(e) => { e.stopPropagation(); onEdit(item) }}
                      title="编辑"
                      aria-label="编辑路由"
                    >
                      <Pencil size={14} />
                    </button>
                    <button
                      class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)] transition-colors"
                      onclick={(e) => { e.stopPropagation(); confirmDelete(item) }}
                      title="删除"
                      aria-label="删除路由"
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
      <div class="mobile-card-list stagger-list">
        {#each items as item (item.routeId)}
          <article class="mobile-list-card">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-sm font-semibold text-[var(--color-text-primary)] truncate">{item.name}</h2>
                <div class="mt-1 text-[11px] text-[var(--color-text-tertiary)] font-mono truncate">{item.routeId}</div>
              </div>
              {#if item.enabled}
                <span class="badge badge-success">启用</span>
              {:else}
                <span class="badge badge-error">禁用</span>
              {/if}
            </div>
            <div class="mobile-list-meta">
              <div class="flex items-center gap-2">
                <span class="text-[var(--color-text-tertiary)]">模式</span>
                <span class="badge badge-success">{modeLabel(item.mode)}</span>
              </div>
              <div class="font-mono truncate">{item.templateId}</div>
              <div class="grid gap-1">
                {#each item.targetUrls.slice(0, 2) as url}
                  <div class="break-all">{url}</div>
                {/each}
                {#if item.targetUrls.length > 2}
                  <div class="text-[var(--color-text-tertiary)]">+{item.targetUrls.length - 2} 更多</div>
                {/if}
              </div>
            </div>
            <div class="mobile-list-actions">
              <button class="btn btn-secondary flex-1" onclick={(e) => { e.stopPropagation(); onEdit(item) }}>
                <Pencil size={14} />
                编辑
              </button>
              <button class="btn btn-danger flex-1" onclick={(e) => { e.stopPropagation(); confirmDelete(item) }}>
                <Trash2 size={14} />
                删除
              </button>
            </div>
          </article>
        {/each}
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
