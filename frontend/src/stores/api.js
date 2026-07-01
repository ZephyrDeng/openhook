function apiBase() {
  return localStorage.getItem('openhook-api-base') || ''
}

function adminToken() {
  return localStorage.getItem('openhook-token') || ''
}

function apiEndpointTemplate() {
  return import.meta.env.VITE_OPENHOOK_API_ENDPOINT || ''
}

function webhookEndpointTemplate() {
  return import.meta.env.VITE_OPENHOOK_WEBHOOK_ENDPOINT || ''
}

function endpointFromTemplate(template, path) {
  if (!template) {
    return path
  }
  return template
    .replaceAll('{path}', encodeURIComponent(path))
    .replaceAll('{rawPath}', path)
}

function apiUrl(path) {
  const base = apiBase()
  if (base) {
    return `${base}${path}`
  }
  return endpointFromTemplate(apiEndpointTemplate(), path)
}

export function webhookDeliveryPath(routeId = '{routeId}') {
  const path = `/webhook/routes/${routeId}`
  return endpointFromTemplate(webhookEndpointTemplate(), path)
}

export function publicWebhookUrl(origin, routeId = '{routeId}') {
  return `${origin}${webhookDeliveryPath(routeId)}`
}

async function api(path, opts = {}) {
  const token = adminToken()
  const url = apiUrl(path)
  const headers = {
    ...(token ? { 'X-OpenHook-Admin-Token': token } : {}),
    ...opts.headers,
  }
  if (opts.body !== undefined && !hasHeader(headers, 'content-type')) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(url, {
    ...opts,
    headers,
    credentials: 'same-origin',
  })

  const data = await res.json().catch(() => null)

  if (!res.ok || (data && data.code && data.code >= 400)) {
    const msg = data?.message || res.statusText
    throw new Error(msg)
  }

  return data
}

function hasHeader(headers, name) {
  return Object.keys(headers).some((key) => key.toLowerCase() === name)
}

export const auth = {
  me: () => api('/api/auth/me'),
  logout: () => api('/api/auth/logout', { method: 'POST' }),
}

export const meta = {
  get: () => api('/api/meta'),
}

export const templates = {
  list: (search = '') => api(`/api/templates${search ? `?search=${encodeURIComponent(search)}` : ''}`),
  get: (id) => api(`/api/templates/${id}`),
  create: (body) => api('/api/templates', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/templates/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/templates/${id}`, { method: 'DELETE' }),
  render: (id, body) => api(`/api/templates/${id}/render`, { method: 'POST', body: JSON.stringify(body) }),
  preview: (body) => api('/api/templates/preview', { method: 'POST', body: JSON.stringify(body) }),
}

export const providers = {
  list: () => api('/api/providers'),
  get: (id) => api(`/api/providers/${id}`),
  createTemplate: (id, body = {}) => api(`/api/providers/${id}/templates`, { method: 'POST', body: JSON.stringify(body) }),
}

export const routes = {
  list: () => api('/api/routes'),
  get: (id) => api(`/api/routes/${id}`),
  create: (body) => api('/api/routes', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/routes/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/routes/${id}`, { method: 'DELETE' }),
  deliver: (id, body) => api(`/api/routes/${id}/deliver`, { method: 'POST', body: JSON.stringify(body) }),
}

export const middlewares = {
  list: () => api('/api/middlewares'),
  get: (id) => api(`/api/middlewares/${id}`),
  create: (body) => api('/api/middlewares', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/middlewares/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/middlewares/${id}`, { method: 'DELETE' }),
}

export const tokens = {
  list: () => api('/api/tokens'),
  get: (id) => api(`/api/tokens/${id}`),
  create: (body) => api('/api/tokens/create', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/tokens/${id}`, { method: 'POST', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/tokens/${id}`, { method: 'DELETE' }),
}

export const deliveries = {
  list: (limit = 50, offset = 0) => api(`/api/deliveries?limit=${limit}&offset=${offset}`),
}

export const filters = {
  list: () => api('/api/filters'),
  create: (body) => api('/api/filters', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/filters/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/filters/${id}`, { method: 'DELETE' }),
}

export const dedupRules = {
  list: () => api('/api/dedup-rule'),
  create: (body) => api('/api/dedup-rule', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => api(`/api/dedup-rule/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => api(`/api/dedup-rule/${id}`, { method: 'DELETE' }),
}
