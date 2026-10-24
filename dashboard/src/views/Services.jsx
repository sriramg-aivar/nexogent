import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { serviceIcon } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function Services() {
  const [services, setServices] = useState([])
  const [hasReview, setHasReview] = useState(false)
  const [loading, setLoading] = useState(true)
  const [auditing, setAuditing] = useState(false)
  const [err, setErr] = useState('')

  async function load() {
    try {
      const s = await api.services()
      setServices(s.services || [])
      setHasReview(s.has_review)
    } catch (e) {
      setErr(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  async function runFullAudit() {
    setAuditing(true); setErr('')
    try {
      await api.review()
      await load()
    } catch (e) {
      setErr('Audit failed: ' + e.message)
    } finally {
      setAuditing(false)
    }
  }

  return (
    <div>
      <div className="pagehead">
        <div>
          <h1>AWS Services</h1>
          <p className="muted">Pick a service to see its resources and run audits.</p>
        </div>
        <button className="btn primary" onClick={runFullAudit} disabled={auditing}>
          {auditing ? 'Auditing all services…' : (hasReview ? 'Re-run full audit' : 'Run full audit')}
        </button>
      </div>

      {err && <div className="banner error">{err}</div>}
      {!hasReview && !loading && (
        <div className="banner info">No audit has run yet. Click <b>Run full audit</b> to discover
          resources across all regions and scan them (read-only).</div>
      )}

      {loading ? <p className="muted">Loading…</p> : (
        <div className="cards">
          {services.map((svc) => (
            <div key={svc.key} className="card" onClick={() => navigate('#/service/' + svc.key)}>
              <div className="card-ico">{serviceIcon(svc.key)}</div>
              <div className="card-body">
                <div className="card-title">{svc.name}</div>
                <div className="card-sub">{svc.scope}</div>
              </div>
              <div className="card-stats">
                <div><b>{svc.resource_count}</b><span>resources</span></div>
                <div className={svc.finding_count > 0 ? 'bad' : 'good'}>
                  <b>{svc.finding_count}</b><span>findings</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
