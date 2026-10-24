import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { serviceIcon } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function ServicePage({ serviceKey }) {
  const [data, setData] = useState(null)
  const [region, setRegion] = useState('')
  const [loading, setLoading] = useState(true)
  const [auditing, setAuditing] = useState(false)
  const [err, setErr] = useState('')

  async function load(r = region) {
    setLoading(true); setErr('')
    try {
      const d = await api.resources(serviceKey, r)
      setData(d)
    } catch (e) {
      setErr(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load('') /* eslint-disable-next-line */ }, [serviceKey])

  function onRegion(e) {
    const r = e.target.value
    setRegion(r)
    load(r)
  }

  async function auditService() {
    setAuditing(true); setErr('')
    try {
      await api.audit(serviceKey, '')
      await load()
    } catch (e) {
      setErr('Audit failed: ' + e.message)
    } finally {
      setAuditing(false)
    }
  }

  const resources = data?.resources || []
  const regions = data?.regions || []

  return (
    <div>
      <div className="crumbs">
        <a onClick={() => navigate('#/')}>Services</a> › <span>{serviceKey.toUpperCase()}</span>
      </div>

      <div className="pagehead">
        <div>
          <h1>{serviceIcon(serviceKey)} {serviceKey.toUpperCase()} resources</h1>
          <p className="muted">Click a resource to see its security audit.</p>
        </div>
        <button className="btn primary" onClick={auditService} disabled={auditing}>
          {auditing ? 'Auditing…' : 'Audit this service'}
        </button>
      </div>

      <div className="toolbar">
        <label>Region</label>
        <select value={region} onChange={onRegion}>
          <option value="">All regions</option>
          {regions.map((r) => <option key={r} value={r}>{r}</option>)}
        </select>
        <span className="muted">{resources.length} resource{resources.length === 1 ? '' : 's'}</span>
      </div>

      {err && <div className="banner error">{err}</div>}

      {loading ? <p className="muted">Loading…</p> : (
        resources.length === 0
          ? <div className="banner info">No {serviceKey.toUpperCase()} resources{region ? ' in ' + region : ''}.
              {' '}Try <b>Audit this service</b> or run a full audit first.</div>
          : (
            <table className="restable">
              <thead>
                <tr><th>Name</th><th>Type</th><th>Region</th><th>Findings</th><th></th></tr>
              </thead>
              <tbody>
                {resources.map((r) => (
                  <tr key={r.id} onClick={() => navigate('#/resource/' + encodeURIComponent(r.id))}>
                    <td className="mono">{r.name || r.id}</td>
                    <td>{r.type}</td>
                    <td><span className="pill">{r.region}</span></td>
                    <td>
                      {r.finding_count > 0
                        ? <span className="count bad">{r.finding_count}</span>
                        : <span className="count good">0</span>}
                    </td>
                    <td className="chev">›</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )
      )}
    </div>
  )
}
