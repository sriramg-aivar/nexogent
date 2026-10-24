import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { Drawer, FindingRow, serviceIcon } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function ResourcePage({ resourceId }) {
  const [resource, setResource] = useState(null)
  const [findings, setFindings] = useState([])
  const [reviewId, setReviewId] = useState('')
  const [loading, setLoading] = useState(true)
  const [auditing, setAuditing] = useState(false)
  const [err, setErr] = useState('')
  const [scannedAt, setScannedAt] = useState('')

  // drawer state
  const [selected, setSelected] = useState(null)
  const [explanation, setExplanation] = useState(null)
  const [explaining, setExplaining] = useState(false)

  async function load() {
    setLoading(true); setErr('')
    try {
      const d = await api.resource(resourceId)
      setResource(d.resource)
      setFindings(d.findings || [])
      setReviewId(d.review_id || '')
    } catch (e) {
      setErr(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() /* eslint-disable-next-line */ }, [resourceId])

  async function auditResource() {
    setAuditing(true); setErr('')
    try {
      const svc = resource?.service || ''
      const d = await api.audit(svc, resourceId)
      setFindings(d.findings || [])
      setReviewId(d.review_id || '')
      setScannedAt(d.scanned_at || new Date().toISOString())
    } catch (e) {
      setErr('Audit failed: ' + e.message)
    } finally {
      setAuditing(false)
    }
  }

  async function openFinding(f) {
    setSelected(f); setExplanation(null); setExplaining(true)
    try {
      const ex = await api.explain(reviewId, f.id)
      setExplanation(ex)
    } catch (e) {
      setExplanation({ problem: f.title, why_dangerous: f.description, impact: '',
        how_to_fix: [f.recommendation], provider: 'offline', model: 'n/a',
        note: 'Explain failed: ' + e.message })
    } finally {
      setExplaining(false)
    }
  }

  const svc = resource?.service || resourceId.split(':')[2] || ''
  const sevClass = findings.length === 0 ? 'good' : 'bad'

  return (
    <div>
      <div className="crumbs">
        <a onClick={() => navigate('#/')}>Services</a> ›{' '}
        {svc && <><a onClick={() => navigate('#/service/' + svc)}>{svc.toUpperCase()}</a> › </>}
        <span>resource</span>
      </div>

      <div className="pagehead">
        <div>
          <h1>{serviceIcon(svc)} {resource?.name || resourceId}</h1>
          <p className="muted mono">{resourceId}</p>
          {resource && <p className="muted">{resource.type} · {resource.region} · account {resource.account_id}</p>}
        </div>
        <button className="btn primary" onClick={auditResource} disabled={auditing}>
          {auditing ? 'Auditing…' : 'Audit this resource'}
        </button>
      </div>

      {err && <div className="banner error">{err}</div>}

      <div className={'verdict ' + sevClass}>
        {findings.length === 0
          ? '✓ No findings on this resource. Looks fine.'
          : `⚠ ${findings.length} finding${findings.length === 1 ? '' : 's'} on this resource.`}
        {scannedAt && <span className="muted"> · scanned {new Date(scannedAt).toLocaleString()}</span>}
      </div>

      {loading ? <p className="muted">Loading…</p> : (
        <div className="findings">
          {findings.map((f) => <FindingRow key={f.id} f={f} onClick={() => openFinding(f)} />)}
        </div>
      )}

      <Drawer
        finding={selected}
        explanation={explanation}
        explaining={explaining}
        onClose={() => setSelected(null)}
      />
    </div>
  )
}
