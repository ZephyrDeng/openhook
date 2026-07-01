<script>
  import { onMount } from 'svelte'
  import { Braces, FileText, Route, Send, Workflow } from 'lucide-svelte'
  import { publicWebhookUrl as buildPublicWebhookUrl, webhookDeliveryPath } from '../stores/api.js'

  const templateFields = [
    { name: '模板名称', value: '例如 generic-alert，用于列表和路由选择' },
    { name: '消息类型', value: 'Markdown、纯文本、HTML，决定渲染内容格式' },
    { name: '模板内容', value: '使用占位符把 payload 字段插入消息' },
    { name: '模拟数据', value: 'JSON 示例数据，编辑时实时预览渲染效果' },
    { name: '可见性', value: '私有模板归当前用户使用，公开模板可被其他用户复用' },
  ]

  const routeFields = [
    { name: '消息模板', value: '选择当前路由要使用的 templateId' },
    { name: '目标 Webhook 地址', value: '填写下游服务地址，例如企业微信、Telegram、QQ 网关或任意 Webhook URL' },
    { name: '请求头', value: '按目标服务要求补充 Content-Type、鉴权头等字段' },
    { name: '投递模式', value: 'envelope 发送标准信封，raw 发送渲染后的原始内容' },
    { name: '中间件', value: '管理员可添加 JavaScript 逻辑做过滤、改写、拒绝投递' },
  ]

  const routePayload = `{
  "title": "Checkout down",
  "severity": "critical",
  "service": "checkout"
}`

  const envelopePayload = `{
  "msgType": "markdown",
  "content": "# Checkout down",
  "messageContent": {
    "title": "Checkout down",
    "severity": "critical"
  },
  "requestId": "req_xxx"
}`

  let publicOrigin = $state('https://your-openhook.example')

  onMount(() => {
    publicOrigin = window.location.origin
  })

  const productionWebhookUrl = $derived(buildPublicWebhookUrl(publicOrigin))
  const webhookPath = $derived(webhookDeliveryPath())

  const curlExample = $derived(`curl -X POST \\
  '${productionWebhookUrl}' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "title": "Checkout down",
    "severity": "critical",
    "service": "checkout"
  }'`)
</script>

<div class="page-shell">
  <div class="page-header mobile-stack">
    <div>
      <h1 class="page-title">使用指南</h1>
      <p class="page-description">从消息模板到路由投递的最短操作路径</p>
    </div>
  </div>

  <div class="page-content guide-page">
    <section class="guide-hero" aria-label="快速上手">
      <div class="guide-hero-title">
        <div class="guide-hero-icon">
          <Workflow size={20} />
        </div>
        <div>
          <p class="guide-kicker">快速上手</p>
          <h2>三步完成一次 Webhook 转发</h2>
        </div>
      </div>
      <div class="guide-flow-grid">
        <article>
          <span>01</span>
          <strong>创建消息模板</strong>
          <p>定义接收 payload 后要渲染出的消息内容。</p>
        </article>
        <article>
          <span>02</span>
          <strong>创建路由</strong>
          <p>选择模板，绑定目标 Webhook 地址和投递模式。</p>
        </article>
        <article>
          <span>03</span>
          <strong>调用投递接口</strong>
          <p>外部系统向 routeId 投递 JSON，OpenHook 负责渲染和转发。</p>
        </article>
      </div>
    </section>

    <div class="guide-two-column">
      <section class="guide-section">
        <div class="guide-section-heading">
          <FileText size={18} />
          <h2>消息模板使用说明</h2>
        </div>
        <p class="guide-copy">模板内容支持 {'{{data.title}}'}、{'{{data.severity}}'} 这类 payload 占位符，也支持 {'{{global.routeId}}'} 读取路由和请求上下文。</p>
        <div class="guide-field-list">
          {#each templateFields as field}
            <div class="guide-field-row">
              <span>{field.name}</span>
              <p>{field.value}</p>
            </div>
          {/each}
        </div>
        <div class="guide-example">
          <div class="guide-example-title">
            <Braces size={14} />
            <span>Markdown 模板示例</span>
          </div>
          <pre class="code-block">{`# {{data.title}}
- severity: {{data.severity}}
- service: {{data.service}}
- route: {{global.routeId}}`}</pre>
        </div>
      </section>

      <section class="guide-section">
        <div class="guide-section-heading">
          <Route size={18} />
          <h2>路由使用说明</h2>
        </div>
        <p class="guide-copy">路由把模板、目标地址、请求头和中间件组合成稳定的投递入口。生产集成优先使用 routeId 投递。</p>
        <div class="guide-field-list">
          {#each routeFields as field}
            <div class="guide-field-row">
              <span>{field.name}</span>
              <p>{field.value}</p>
            </div>
          {/each}
          <div class="guide-field-row">
            <span>对外 Webhook 地址</span>
            <p>创建路由后，把 {productionWebhookUrl} 提供给上游系统调用。</p>
          </div>
          <div class="guide-field-row">
            <span>目标 Webhook 地址</span>
            <p>这是 OpenHook 转发到下游服务的地址，填写企业微信群机器人、Telegram 网关、QQ 网关或业务系统 Webhook。</p>
          </div>
        </div>
        <div class="guide-mode-grid">
          <div>
            <strong>envelope</strong>
            <p>发送标准消息信封，适合内部统一接收方。</p>
          </div>
          <div>
            <strong>raw</strong>
            <p>直接发送模板渲染结果，适合企业微信、Telegram、QQ 等 provider。</p>
          </div>
        </div>
      </section>
    </div>

    <section class="guide-section">
      <div class="guide-section-heading">
        <Send size={18} />
        <h2>投递调用</h2>
      </div>
      <p class="guide-copy">创建路由后，外部系统调用 POST {webhookPath}，请求体会作为 data 进入模板渲染。生产环境的完整地址是 {productionWebhookUrl}。</p>
      <div class="guide-field-list">
        <div class="guide-field-row">
          <span>请求方法</span>
          <p>POST</p>
        </div>
        <div class="guide-field-row">
          <span>请求 Header</span>
          <p>Content-Type: application/json</p>
        </div>
        <div class="guide-field-row">
          <span>routeId 来源</span>
          <p>保存路由后，在路由列表或路由编辑页复制当前路由的对外 Webhook 地址。</p>
        </div>
        <div class="guide-field-row">
          <span>字段映射</span>
          <p>请求体里的 title、severity、service 会在模板中对应 {'{{data.title}}'}、{'{{data.severity}}'}、{'{{data.service}}'}。</p>
        </div>
      </div>
      <div class="guide-code-grid">
        <div class="guide-example">
          <div class="guide-example-title">
            <span>curl 调用示例</span>
          </div>
          <pre class="code-block">{curlExample}</pre>
        </div>
        <div class="guide-example">
          <div class="guide-example-title">
            <span>请求体</span>
          </div>
          <pre class="code-block">{routePayload}</pre>
        </div>
        <div class="guide-example">
          <div class="guide-example-title">
            <span>envelope 输出结构</span>
          </div>
          <pre class="code-block">{envelopePayload}</pre>
        </div>
      </div>
    </section>
  </div>
</div>
