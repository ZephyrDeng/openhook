<script>
  import { dedupRules } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import Modal from '../components/Modal.svelte'
  import { Plus, CopyX, Pencil, Trash2, Save, X } from 'lucide-svelte'

  let items = $state([])
  let loading = $state(true)
  let editingId = $state(null)
  let form = $state({ name: '', status: true, domain: [], platform: '', payload: {} })
  let showDeleteModal = $state(false)
  let deletingItem = $state(null)

  async function load() {
    loading = true
    try {
      const res = await dedupRules.list()
      items = res.data || []
    } catch (e) {
      toast.error('加载失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function startNew() {
    editingId = 'new'
    form = { name: '', status: true, domain: [], platform: '', payload: {} }
  }

  function startEdit(item) {
    editingId = item.id
    form = { name: item.name, status: item.status, domain: item.domain || [], platform: item.platform || '', payload: item.payload || {} }
  }

  function cancelEdit() {
    editingId = null
  }

  async function save() {
    try {
      const body = { ...form, payload: JSON.stringify(form.payload) }
      if (editingId === 'new') {
        await dedupRules.create(body)
        toast.success('已创建')
      } else {
        await dedupRules.update(editingId, body)
        toast.success('已更新')
      }
      editingId = null
      load()
    } catch (e) {
      toast.error('保存失败: ' + e.message)
    }
  }

  async function doDelete() {
    try {
      await dedupRules.delete(deletingItem.id)
      toast.success('已删除')
      load()
    } catch (e) {
      toast.error('删除失败')
    } finally {
      showDeleteModal = false
      deletingItem = null
    }
  }

  $effect(() => { load() })
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-subtle)]">
    <div>
      <h1 class="text-xl font-semibold text-[var(--color-text-primary)]">去重规则</h1>
      <p class="text-sm text-[var(--color-text-secondary)] mt-0.5">配置消息去重策略</p>
    </div>
    <button class="btn btn-primary" onclick={startNew}><Plus size={16} />新建</button>
  </div>

  <div class="flex-1 overflow-auto p-6">
    {#if loading}
      <div class="flex items-center justify-center h-64"><div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div></div>
    {:else if items.length === 0 && !editingId}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="w-12 h-12 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-4"><CopyX size={24} class="text-[var(--color-text-tertiary)]" /></div>
        <p class="text-sm font-medium text-[var(--color-text-primary)]">还没有去重规则</p>
        <button class="btn btn-primary mt-4" onclick={startNew}><Plus size={16} />新建</button>
      </div>
    {:else}
      <div class="space-y-4">
        {#if editingId}
          <div class="card space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-medium">{editingId === 'new' ? '新建去重规则' : '编辑去重规则'}</h3>
              <button class="btn btn-ghost p-1" onclick={cancelEdit}><X size={16} /></button>
            </div>
            <input type="text" class="input" bind:value={form.name} placeholder="名称" />
            <div class="flex items-center gap-2">
              <input type="checkbox" id="d-status" bind:checked={form.status} class="w-4 h-4 accent-[var(--color-accent)]" />
              <label for="d-status" class="text-sm">启用</label>
            </div>
            <div class="flex justify-end gap-2">
              <button class="btn btn-secondary" onclick={cancelEdit}>取消</button>
              <button class="btn btn-primary" onclick={save}><Save size={14} />保存</button>
            </div>
          </div>
        {/if}

        {#if items.length > 0}
          <div class="card overflow-hidden">
            <table class="w-full">
              <thead><tr><th class="table-header">名称</th><th class="table-header">状态</th><th class="table-header w-24">操作</th></tr></thead>
              <tbody>
                {#each items as item}
                  <tr class="table-row">
                    <td class="table-cell font-medium">{item.name}</td>
                    <td class="table-cell">{#if item.status}<span class="badge badge-success">启用</span>{:else}<span class="badge badge-error">禁用</span>{/if}</td>
                    <td class="table-cell">
                      <div class="flex items-center gap-1">
                        <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]" onclick={() => startEdit(item)}><Pencil size={14} /></button>
                        <button class="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:text-[var(--color-error)]" onclick={() => { deletingItem = item; showDeleteModal = true; }}><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<Modal show={showDeleteModal} title="确认删除" onclose={() => showDeleteModal = false}>
  <div class="text-sm text-[var(--color-text-secondary)]">确定要删除规则 <strong class="text-[var(--color-text-primary)]">{deletingItem?.name}</strong> 吗？</div>
  <div class="flex justify-end gap-2 mt-5">
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>取消</button>
    <button class="btn btn-danger" onclick={doDelete}>删除</button>
  </div>
</Modal>
