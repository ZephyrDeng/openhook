<script>
  import { tokens } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import FormField from '../components/FormField.svelte'
  import Modal from '../components/Modal.svelte'
  import { Plus, KeyRound, Pencil, Trash2, Save, X, Copy } from 'lucide-svelte'

  let items = $state([])
  let loading = $state(true)
  let editingId = $state(null)
  let form = $state({ name: '', templateIds: [], isCoverAll: false, remark: '', expireAt: 0 })
  let showDeleteModal = $state(false)
  let deletingItem = $state(null)

  async function load() {
    loading = true
    try {
      const res = await tokens.list()
      items = res.data || []
    } catch (e) {
      toast.error('加载失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function startNew() {
    editingId = 'new'
    form = { name: '', templateIds: [], isCoverAll: false, remark: '', expireAt: 0 }
  }

  function startEdit(item) {
    editingId = item.token
    form = { name: item.name, templateIds: item.templateIds || [], isCoverAll: item.isCoverAll, remark: item.remark || '', expireAt: item.expireAt || 0 }
  }

  function cancelEdit() {
    editingId = null
  }

  async function save() {
    try {
      if (editingId === 'new') {
        await tokens.create(form)
        toast.success('令牌已创建')
      } else {
        await tokens.update(editingId, form)
        toast.success('令牌已更新')
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
      await tokens.delete(deletingItem.token)
      toast.success('已删除')
      load()
    } catch (e) {
      toast.error('删除失败: ' + e.message)
    } finally {
      showDeleteModal = false
      deletingItem = null
    }
  }

  function copyToken(t) {
    navigator.clipboard.writeText(t)
    toast.success('已复制到剪贴板')
  }

  $effect(() => { load() })

  const statusLabel = (s) => {
    if (s === 1) return '启用'
    if (s === 0) return '过期'
    return '已删除'
  }
</script>

<div class="page-shell">
  <div class="page-header mobile-stack">
    <div>
      <h1 class="page-title">令牌</h1>
      <p class="page-description">管理外部系统更新模板的访问令牌</p>
    </div>
    <button class="btn btn-primary" onclick={startNew}>
      <Plus size={16} />
      新建令牌
    </button>
  </div>

  <div class="page-content">
    {#if loading}
      <div class="flex items-center justify-center h-64"><div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div></div>
    {:else if items.length === 0 && !editingId}
      <div class="empty-state">
        <div class="empty-state-icon"><KeyRound size={24} /></div>
        <p class="empty-state-title">还没有令牌</p>
        <p class="empty-state-desc">创建令牌以允许外部系统更新模板</p>
        <button class="btn btn-primary" onclick={startNew}><Plus size={16} />新建令牌</button>
      </div>
    {:else}
      <div class="space-y-4">
        {#if editingId}
          <div class="card space-y-4 page-transition">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-[var(--color-text-primary)]">{editingId === 'new' ? '新建令牌' : '编辑令牌'}</h3>
              <button class="btn btn-ghost p-1" onclick={cancelEdit} aria-label="取消编辑"><X size={16} /></button>
            </div>
            <FormField label="名称" forId="token-name" required>
              <input id="token-name" type="text" class="input" bind:value={form.name} placeholder="例如: cicd-token" />
            </FormField>
            <FormField label="备注" forId="token-remark" helper="可选，用于标识令牌用途">
              <input id="token-remark" type="text" class="input" bind:value={form.remark} placeholder="例如: CI/CD 流水线使用" />
            </FormField>
            <div class="flex items-center gap-2">
              <input type="checkbox" id="cover-all" bind:checked={form.isCoverAll} class="w-4 h-4 accent-[var(--color-accent)]" />
              <label for="cover-all" class="text-sm text-[var(--color-text-primary)]">覆盖所有模板</label>
            </div>
            <div class="flex justify-end gap-2">
              <button class="btn btn-secondary" onclick={cancelEdit}>取消</button>
              <button class="btn btn-primary" onclick={save}><Save size={14} />保存</button>
            </div>
          </div>
        {/if}

        {#if items.length > 0}
          <div class="card desktop-table-card">
            <table class="w-full">
              <thead>
                <tr>
                  <th class="table-header">名称</th>
                  <th class="table-header">令牌</th>
                  <th class="table-header">覆盖范围</th>
                  <th class="table-header">状态</th>
                  <th class="table-header w-24">操作</th>
                </tr>
              </thead>
              <tbody>
                {#each items as item, i (item.token)}
                  <tr class="table-row" style="animation-delay: {i * 30}ms">
                    <td class="table-cell">
                      <div class="font-medium text-sm">{item.name}</div>
                      <div class="text-xs text-[var(--color-text-tertiary)]">{item.remark || '-'}</div>
                    </td>
                    <td class="table-cell">
                      <div class="flex items-center gap-2">
                        <code class="font-mono text-xs bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded">{item.token.slice(0, 16)}...</code>
                        <button class="text-[var(--color-text-tertiary)] hover:text-[var(--color-accent)] transition-colors" onclick={() => copyToken(item.token)} aria-label="复制令牌"><Copy size={14} /></button>
                      </div>
                    </td>
                    <td class="table-cell">
                      {#if item.isCoverAll}
                        <span class="badge badge-success">全部</span>
                      {:else}
                        <span class="text-xs text-[var(--color-text-secondary)]">{item.templateIds?.length || 0} 个模板</span>
                      {/if}
                    </td>
                    <td class="table-cell">
                      {#if item.status === 1}
                        <span class="badge badge-success">{statusLabel(item.status)}</span>
                      {:else}
                        <span class="badge badge-error">{statusLabel(item.status)}</span>
                      {/if}
                    </td>
                    <td class="table-cell">
                      <div class="flex items-center gap-1">
                        <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] transition-colors" onclick={() => startEdit(item)} aria-label="编辑令牌"><Pencil size={14} /></button>
                        <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)] transition-colors" onclick={() => confirmDelete(item)} aria-label="删除令牌"><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          <div class="mobile-card-list stagger-list">
            {#each items as item (item.token)}
              <article class="mobile-list-card">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <h2 class="text-sm font-semibold text-[var(--color-text-primary)] truncate">{item.name}</h2>
                    <div class="mt-1 text-xs text-[var(--color-text-tertiary)] truncate">{item.remark || '-'}</div>
                  </div>
                  {#if item.status === 1}
                    <span class="badge badge-success">{statusLabel(item.status)}</span>
                  {:else}
                    <span class="badge badge-error">{statusLabel(item.status)}</span>
                  {/if}
                </div>
                <div class="mobile-list-meta">
                  <div class="flex items-center gap-2">
                    <code class="font-mono bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded">{item.token.slice(0, 16)}...</code>
                    <button class="text-[var(--color-text-tertiary)] hover:text-[var(--color-accent)] transition-colors" onclick={() => copyToken(item.token)} aria-label="复制令牌">
                      <Copy size={14} />
                    </button>
                  </div>
                  {#if item.isCoverAll}
                    <span class="badge badge-success w-fit">全部模板</span>
                  {:else}
                    <span>{item.templateIds?.length || 0} 个模板</span>
                  {/if}
                </div>
                <div class="mobile-list-actions">
                  <button class="btn btn-secondary flex-1" onclick={() => startEdit(item)}><Pencil size={14} />编辑</button>
                  <button class="btn btn-danger flex-1" onclick={() => confirmDelete(item)}><Trash2 size={14} />删除</button>
                </div>
              </article>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<Modal show={showDeleteModal} title="确认删除" onclose={() => showDeleteModal = false}>
  <div class="text-sm text-[var(--color-text-secondary)]">确定要删除令牌 <strong class="text-[var(--color-text-primary)]">{deletingItem?.name}</strong> 吗？</div>
  <div class="flex justify-end gap-2 mt-5">
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>取消</button>
    <button class="btn btn-danger" onclick={doDelete}>删除</button>
  </div>
</Modal>
