// Thin API client. Same-origin: Nginx proxies /api and /health to the backend.
const base = ''

async function j(method, path) {
  const res = await fetch(base + path, { method })
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}`)
  return res.json()
}

export const api = {
  health: () => j('GET', '/health'),
  runReview: () => j('POST', '/api/v1/review'),
  latest: () => j('GET', '/api/v1/review/latest'),
  resources: () => j('GET', '/api/v1/resources'),
  explain: (reviewId, findingId) =>
    j('POST', `/api/v1/explain?review=${encodeURIComponent(reviewId)}&finding=${encodeURIComponent(findingId)}`),
}
