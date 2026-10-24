// API client for the Nexogent portal. Same-origin: Nginx proxies /api + /health
// to the backend. Carries the superadmin bearer token on every call.

const TOKEN_KEY = 'nexogent_token'

export function getToken() { return localStorage.getItem(TOKEN_KEY) || '' }
export function setToken(t) { localStorage.setItem(TOKEN_KEY, t) }
export function clearToken() { localStorage.removeItem(TOKEN_KEY) }

async function req(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  const tok = getToken()
  if (tok) headers['Authorization'] = 'Bearer ' + tok
  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) {
    clearToken()
    const err = new Error('unauthorized')
    err.unauthorized = true
    throw err
  }
  if (!res.ok) {
    let msg = `${method} ${path} -> ${res.status}`
    try { const j = await res.json(); if (j.error) msg = j.error } catch {}
    throw new Error(msg)
  }
  return res.json()
}

export const api = {
  authStatus: () => req('GET', '/api/v1/auth/status'),
  login: (password) => req('POST', '/api/v1/auth/login', { password }),

  health: () => req('GET', '/health'),
  services: () => req('GET', '/api/v1/services'),
  regions: () => req('GET', '/api/v1/regions'),
  resources: (service, region) => {
    const q = new URLSearchParams()
    if (service) q.set('service', service)
    if (region) q.set('region', region)
    return req('GET', '/api/v1/resources?' + q.toString())
  },
  resource: (id) => req('GET', '/api/v1/resource?id=' + encodeURIComponent(id)),
  review: () => req('POST', '/api/v1/review'),
  latest: () => req('GET', '/api/v1/review/latest'),
  reviews: () => req('GET', '/api/v1/reviews'),
  audit: (service, resource) => {
    const q = new URLSearchParams()
    if (service) q.set('service', service)
    if (resource) q.set('resource', resource)
    return req('POST', '/api/v1/audit?' + q.toString())
  },
  explain: (reviewId, findingId) =>
    req('POST', `/api/v1/explain?review=${encodeURIComponent(reviewId)}&finding=${encodeURIComponent(findingId)}`),
  integrations: () => req('GET', '/api/v1/integrations'),
}
