import React, { useEffect, useMemo, useState } from 'react'
import { api } from './api.js'

const SEV_ORDER = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFORMATIONAL']

function sevClass(s) {
  return 'sev sev-' + (s || 'INFORMATIONAL').toLowerCase()
}

function ratingClass(rating) {
  const r = (rating || '').toLowerCase()
  if (r.includes('excellent')) return 'rating good'
  if (r.includes('good')) return 'rating good'
  if (r.includes('needs')) return 'rating warn'
  return 'rating bad'
}

export default function App() {
  const [review, setReview] = useState(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')
  const [service, setService] = useState('all')
  const [selected, setSelected] = useState(null) // finding for the drawer
  const [explanation, setExplanation] = useState(null)
  const [explaining, setExplaining] = useState(false)

  async function load() {
    setErr('')
    try {
      const latest = await api.latest().catch(() => null)
      if (latest && latest.review_id) setReview(latest)
    } catch (e) {
      /* no review yet */
    }
  }

  useEffect(() => { load() }, [])

  async function runReview() {
    setLoading(true); setErr('')
    try {
      const r = await api.runReview()
      setReview(r)
      setService('all')
    } catch (e) {
      setErr('Review failed: ' + e.message)
    } finally {
      setLoading(false)
    }
  }

  async function openFinding(f) {
    setSelected(f); setExplanation(null); setExplaining(true)
    try {
      const ex = await api.explain(review.review_id, f.id)
      setExplanation(ex)
    } catch (e) {
      setExplanation({ problem: f.title, why_dangerous: f.description, impact: '',
        how_to_fix: [f.recommendation], provider: 'offline', model: 'n/a',
        note: 'Explain request failed: ' + e.message })
    } finally {
      setExplaining(false)
    }
  }

  const findings = review?.findings || []
  const resources = review?.resources || []

  const services = useMemo(() => {
    const set = new Set(findings.map(f => f.service))
    resources.forEach(r => set.add(r.service))
    return ['all', ...Array.from(set).sort()]
  }, [findings, resources])

  const shown = useMemo(() => {
    const list = service === 'all' ? findings : findings.filter(f => f.service === service)
    return [...list].sort((a, b) => b.risk_score - a.risk_score)
  }, [findings, service])

  const resourcesByService = useMemo(() => {
    const g = {}
    resources.forEach(r => { (g[r.service] ||= []).push(r) })
    return g
  }, [resources])

  const counts = review?.summary?.by_severity || {}

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="logo">◆</span> Nexogent
          <span className="tag">AWS Security Posture · read-only</span>
        </div>
        <button className="btn primary" onClick={runReview} disabled={loading}>
          {loading ? 'Scanning…' : 'Run review'}
        </button>
      </header>

      {err && <div className="banner error">{err}</div>}

      {!review && (
        <div className="empty">
          <h2>No review yet</h2>
          <p>Run a read-only posture review of this AWS account.</p>
          <button className="btn primary" onClick={runReview} disabled={loading}>
            {loading ? 'Scanning…' : 'Run review'}
          </button>
        </div>
      )}

      {review && (
        <>
          <section className="posture">
            <div className="score-card">
              <div className="score">{review.posture_score}<span>/100</span></div>
              <div className={ratingClass(review.rating)}>{review.rating}</div>
            </div>
            <div className="meta">
              <div><label>Account</label><span>{review.account_id}</span></div>
              <div><label>Region</label><span>{(review.regions || []).join(', ')}</span></div>
              <div><label>Findings</label><span>{review.summary?.total ?? 0}</span></div>
              <div><label>Resources</label><span>{resources.length}</span></div>
              <div><label>Sources</label><span>{(review.sources_run || []).join(', ')}</span></div>
            </div>
            <div className="sev-counts">
              {SEV_ORDER.map(s => (
                <div key={s} className={sevClass(s)}>
                  <b>{counts[s] || 0}</b><span>{s}</span>
                </div>
              ))}
            </div>
          </section>

          <div className="layout">
            <aside className="sidebar">
              <h3>Services</h3>
              <ul>
                {services.map(s => (
                  <li key={s}
                      className={s === service ? 'active' : ''}
                      onClick={() => setService(s)}>
                    <span>{s === 'all' ? 'All services' : s.toUpperCase()}</span>
                    <small>{s === 'all'
                      ? findings.length
                      : findings.filter(f => f.service === s).length}</small>
                  </li>
                ))}
              </ul>
              {service !== 'all' && (
                <div className="res-list">
                  <h4>Resources</h4>
                  {(resourcesByService[service] || []).map(r => (
                    <div key={r.id} className="res" title={r.id}>
                      <b>{r.type}</b> {r.name}
                    </div>
                  ))}
                  {!(resourcesByService[service] || []).length && <p className="muted">none discovered</p>}
                </div>
              )}
            </aside>

            <main className="findings">
              <h3>{service === 'all' ? 'All findings' : service.toUpperCase() + ' findings'} ({shown.length})</h3>
              {!shown.length && <p className="muted">No findings here. Looks clean.</p>}
              {shown.map(f => (
                <div key={f.id} className="finding" onClick={() => openFinding(f)}>
                  <div className="risk" title="risk score">{f.risk_score}</div>
                  <div className="fbody">
                    <div className="ftitle">{f.title}</div>
                    <div className="fmeta">
                      <span className={sevClass(f.severity)}>{f.severity}</span>
                      <span className="src">{f.source}</span>
                      {f.framework_control && <span className="ctrl">{f.framework_control}</span>}
                      <span className="svc">{f.service}</span>
                    </div>
                  </div>
                  <div className="chev">›</div>
                </div>
              ))}
            </main>
          </div>
        </>
      )}

      {selected && (
        <div className="drawer-backdrop" onClick={() => setSelected(null)}>
          <div className="drawer" onClick={e => e.stopPropagation()}>
            <button className="close" onClick={() => setSelected(null)}>✕</button>
            <span className={sevClass(selected.severity)}>{selected.severity}</span>
            <h2>{selected.title}</h2>
            <p className="muted">{selected.service} · {selected.resource_id}</p>

            {explaining && <p>Generating explanation…</p>}
            {explanation && (
              <div className="explain">
                <h4>Problem</h4><p>{explanation.problem}</p>
                {explanation.why_dangerous && (<><h4>Why it's dangerous</h4><p>{explanation.why_dangerous}</p></>)}
                {explanation.impact && (<><h4>Impact</h4><p>{explanation.impact}</p></>)}
                <h4>How to fix</h4>
                <ol>{(explanation.how_to_fix || []).map((s, i) => <li key={i}>{s}</li>)}</ol>
                {explanation.aws_console_path && (<><h4>Console path</h4><p>{explanation.aws_console_path}</p></>)}
                <div className="provider">
                  via <b>{explanation.provider}</b>{explanation.model ? ` (${explanation.model})` : ''}
                  {explanation.note && <div className="note">{explanation.note}</div>}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <footer className="foot">
        Read-only · recommend-only · never changes your account · open-source Prowler + built-in checks
      </footer>
    </div>
  )
}
