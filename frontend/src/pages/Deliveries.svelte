<script>
  import { deliveries } from '../stores/api.js'
  import { toast } from '../stores/toast.js'
  import Modal from '../components/Modal.svelte'
  import { Truck, ChevronDown, ChevronUp } from 'lucide-svelte'

  let items = $state([])
  let loading = $state(true)
  let expandedId = $state(null)
  let showDetailModal = $state(false)
  let detailItem = $state(null)

  async function load() {
    loading = true
    try {
      const res = await deliveries.list(50, 0)
      items = res.data || []
    } catch (e) {
      toast.error('加载失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function showDetail(item) {
    detailItem = item
    showDetailModal = true
  }

  function formatTime(ts) {
    if (!ts) return '-'
    return new Date(ts).toLocaleString('zh-CN')
  }

  $effect(() => { load() })
</script>

<div class="page-shell">
  <div class="page-header">
    <div>
      <h1 class="page-title">投递日志</h1>
      <p class="page-description">查看所有 Webhook 投递记录和响应详情</p>
    </div>
  </div>

  <div class="page-content">
    {#if loading}
      <div class="flex items-center justify-center h-64"><div class="w-8 h-8 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div></div>
    {:else if items.length === 0}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="w-12 h-12 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center mb-4"><Truck size={24} class="text-[var(--color-text-tertiary)]" /></div>
        <p class="text-sm font-medium text-[var(--color-text-primary)]">暂无投递记录</p>
        <p class="text-sm text-[var(--color-text-secondary)] mt-1">通过路由投递消息后将在这里显示日志</p>
      </div>
    {:else}
      <div class="card desktop-table-card">
        <table class="w-full">
          <thead>
            <tr>
              <th class="table-header w-10"></th>
              <th class="table-header">请求 ID</th>
              <th class="table-header">目标地址</th>
              <th class="table-header">状态码</th>
              <th class="table-header">结果</th>
              <th class="table-header">时间</th>
            </tr>
          </thead>
          <tbody>
            {#each items as item (item.id)}
              <tr class="table-row cursor-pointer" onclick={() => showDetail(item)}>
                <td class="table-cell w-10">
                  {#if item.success}
                    <span class="inline-block w-2 h-2 rounded-full bg-[var(--color-success)]"></span>
                  {:else}
                    <span class="inline-block w-2 h-2 rounded-full bg-[var(--color-error)]"></span>
                  {/if}
                </td>
                <td class="table-cell font-mono text-xs">{item.requestId}</td>
                <td class="table-cell text-xs text-[var(--color-text-secondary)]">{item.targetUrl}</td>
                <td class="table-cell">
                  <span class="font-mono text-xs {item.statusCode >= 200 && item.statusCode < 300 ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'}">{item.statusCode || '-'}</span>
                </td>
                <td class="table-cell">
                  {#if item.success}
                    <span class="badge badge-success">成功</span>
                  {:else}
                    <span class="badge badge-error">{item.message || '失败'}</span>
                  {/if}
                </td>
                <td class="table-cell text-xs text-[var(--color-text-tertiary)]">{formatTime(item.createAt)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="mobile-card-list">
        {#each items as item (item.id)}
          <button type="button" class="mobile-list-card text-left w-full" onclick={() => showDetail(item)}>
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="font-mono text-xs text-[var(--color-text-primary)] truncate">{item.requestId}</div>
                <div class="mt-1 text-xs text-[var(--color-text-tertiary)]">{formatTime(item.createAt)}</div>
              </div>
              {#if item.success}
                <span class="badge badge-success">成功</span>
              {:else}
                <span class="badge badge-error">{item.message || '失败'}</span>
              {/if}
            </div>
            <div class="mobile-list-meta">
              <div class="flex items-center gap-2">
                <span class="text-[var(--color-text-tertiary)]">状态码</span>
                <span class="font-mono {item.statusCode >= 200 && item.statusCode < 300 ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'}">{item.statusCode || '-'}</span>
              </div>
              <div class="break-all">{item.targetUrl}</div>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>

<Modal show={showDetailModal} title="投递详情" onclose={() => showDetailModal = false}>
  {#if detailItem}
    <div class="space-y-4 max-h-[70vh] overflow-auto">
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <span class="text-[var(--color-text-tertiary)]">请求 ID</span>
          <div class="font-mono text-xs mt-0.5">{detailItem.requestId}</div>
        </div>
        <div>
          <span class="text-[var(--color-text-tertiary)]">时间</span>
          <div class="mt-0.5">{formatTime(detailItem.createAt)}</div>
        </div>
        <div>
          <span class="text-[var(--color-text-tertiary)]">状态</span>
          <div class="mt-0.5">
            {#if detailItem.success}
              <span class="badge badge-success">成功</span>
            {:else}
              <span class="badge badge-error">失败</span>
            {/if}
          </div>
        </div>
        <div>
          <span class="text-[var(--color-text-tertiary)]">状态码</span>
          <div class="font-mono text-xs mt-0.5">{detailItem.statusCode || '-'}</div>
        </div>
      </div>

      {#if detailItem.routeId}
        <div>
          <span class="text-[var(--color-text-tertiary)] text-sm">路由</span>
          <div class="font-mono text-xs mt-0.5">{detailItem.routeId}</div>
        </div>
      {/if}
      {#if detailItem.templateId}
        <div>
          <span class="text-[var(--color-text-tertiary)] text-sm">模板</span>
          <div class="font-mono text-xs mt-0.5">{detailItem.templateId}</div>
        </div>
      {/if}

      <div>
        <span class="text-[var(--color-text-tertiary)] text-sm">目标地址</span>
        <div class="font-mono text-xs mt-0.5 break-all">{detailItem.targetUrl}</div>
      </div>

      {#if detailItem.requestBody}
        <div>
          <span class="text-[var(--color-text-tertiary)] text-sm">请求体</span>
          <pre class="code-block mt-1 text-xs">{JSON.stringify(detailItem.requestBody, null, 2)}</pre>
        </div>
      {/if}

      {#if detailItem.responseBody}
        <div>
          <span class="text-[var(--color-text-tertiary)] text-sm">响应体</span>
          <pre class="code-block mt-1 text-xs">{typeof detailItem.responseBody === 'string' ? detailItem.responseBody : JSON.stringify(detailItem.responseBody, null, 2)}</pre>
        </div>
      {/if}
    </div>
  {/if}
</Modal>
