import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { ServiceIcon } from '../icons.jsx'
import { navigate } from '../App.jsx'

export default function Inventory() {
  const [services, setServices] = useState([])
  const [hasReview, setHasReview] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function load() {
    try { const s = await api.services(); setServices(s.services || []); setHasReview(s.has_review) }
    catch (e) { setErr(e.message) } finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  async function runAudit() { setBusy(true); try { await api.review(); await load() } catch (e) { setErr('Audit failed: ' + e.message) } finally { setBusy(false) } }

  return (
    <div>
      <div className="pagehead">
        <div><h1>Inventory</h1><p className="muted">Resources by service and region.</p></div>
        <button className="btn primary" onClick={runAudit} disabled={busy}>{busy ? 'Auditing…' : (hasReview ? 'Re-run full audit' : 'Run full audit')}</button>
      </div>
      {err && <div className="banner error">{err}</div>}
      {!hasReview && !loading && <div className="banner info">No audit yet — run a full audit to populate the inventory.</div>}
      {loading ? <div className="skeleton" style={{ height: 80 }} /> : (
        <div className="cards">
          {services.map(svc => (
            <div key={svc.key} className="card click" onClick={() => navigate('#/service/' + svc.key)}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <ServiceIcon svc={svc.key} />
                <div><div style={{ fontWeight: 700 }}>{svc.name}</div><div className="muted" style={{ fontSize: 11, textTransform: 'uppercase' }}>{svc.scope}</div></div>
              </div>
              <div className="toolbar" style={{ marginTop: 12, marginBottom: 0 }}>
                <span className="num"><b>{svc.resource_count}</b> <span className="muted">resources</span></span>
                <span className={'count ' + (svc.finding_count ? 'bad' : 'good')}>{svc.finding_count}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
