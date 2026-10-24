import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { ScoreRing, TrendChart, ratingClass, Badge } from '../components.jsx'
import { navigate } from '../App.jsx'

export default function Overview({ meta }) {
  const [ov, setOv] = useState(null)
  const [err, setErr] = useState('')
  const [scanning, setScanning] = useState(false)
  const [deep, setDeep] = useState('')

  async function load() { try { setOv(await api.overview()) } catch (e) { setErr(e.message) } }
  useEffect(() => { load() }, [])

  async function quickAll() {
    setScanning(true); setErr('')
    try { await api.review(); await load() } catch (e) { setErr('Scan failed: ' + e.message) } finally { setScanning(false) }
  }
  async function deepScan() {
    setDeep('starting'); setErr('')
    try {
      const { run_id } = await api.deepScan()
      setDeep('running')
      const poll = setInterval(async () => {
        try {
          const st = await api.deepScanStatus(run_id)
          setDeep(st.state)
          if (st.state === 'completed' || st.state === 'failed') { clearInterval(poll); setTimeout(load, 2000) }
        } catch { clearInterval(poll); setDeep('') }
      }, 5000)
    } catch (e) { setErr('Deep scan: ' + e.message); setDeep('') }
  }

  if (err) return <div className="banner error">{err}</div>
  if (!ov) return <div className="skeleton" style={{ height: 120 }} />
  if (!ov.has_review) return (
    <div className="empty">
      <h2>No audit has run yet</h2>
      <p>Run a full read-only scan to discover resources across all regions and audit them.</p>
      <button className="btn primary" onClick={quickAll} disabled={scanning}>{scanning ? 'Scanning…' : 'Run full audit'}</button>
    </div>
  )

  const kubeAvail = meta?.kube_available
  return (
    <div>
      <div className="pagehead">
        <h1>Overview</h1>
        <div className="toolbar" style={{ margin: 0 }}>
          <button className="btn" onClick={quickAll} disabled={scanning}>{scanning ? 'Scanning…' : 'Quick audit all'}</button>
          {kubeAvail && <button className="btn primary" onClick={deepScan} disabled={!!deep && deep !== 'completed' && deep !== 'failed'}>
            {deep ? `Deep scan: ${deep}` : 'Deep scan'}</button>}
        </div>
      </div>

      <div className="tiles">
        <div className="card">
          <ScoreRing score={ov.score} rating={ov.rating} />
          <div className={'rating ' + ratingClass(ov.rating)}>{ov.rating}</div>
          {ov.previous_score != null && <p className="muted" style={{ marginTop: 8 }}>Previous: {ov.previous_score}</p>}
        </div>
        <div className="card">
          <h3>Score trend</h3>
          <TrendChart points={(ov.trend || []).slice().reverse()} />
          <h3 style={{ marginTop: 16 }}>What changed since last scan</h3>
          <div className="tilegrid">
            <div className="card" onClick={() => navigate('#/findings')} style={{ cursor: 'pointer' }}><b className="num" style={{ fontSize: 22 }}>{ov.changes?.new_count || 0}</b><div className="muted">New</div></div>
            <div className="card"><b className="num" style={{ fontSize: 22 }}>{ov.changes?.fixed_count || 0}</b><div className="muted">Fixed</div></div>
            <div className="card"><b className="num" style={{ fontSize: 22 }}>{ov.changes?.regressed_count || 0}</b><div className="muted">Regressed</div></div>
          </div>
        </div>
      </div>

      <h2 style={{ marginTop: 24 }}>Fix these first</h2>
      {(ov.top_fixes || []).length === 0 ? <p className="muted">Nothing open — looks clean.</p> :
        (ov.top_fixes || []).map((t, i) => (
          <div key={i} className="finding" onClick={() => navigate('#/findings')}>
            <div className="risk">{Math.round(t.finding.risk_score)}</div>
            <div className="fbody"><div className="ftitle">{t.finding.title}</div>
              <div className="fmeta"><Badge sev={t.finding.severity} /><span className="chip">{t.finding.region}</span>
                {t.fix && <span className="chip">effort {t.fix.effort}</span>}</div></div>
          </div>
        ))}

      <div className="tilegrid" style={{ marginTop: 20 }}>
        <div className="card click" onClick={() => navigate('#/paths')}><b className="num" style={{ fontSize: 22 }}>{ov.paths || 0}</b><div className="muted">Attack paths</div></div>
        <div className="card click" onClick={() => navigate('#/risks')}><b className="num" style={{ fontSize: 22 }}>{ov.accepted || 0}</b><div className="muted">Accepted risks</div></div>
        <div className="card"><b className="num" style={{ fontSize: 22 }}>{ov.duplicates_merged || 0}</b><div className="muted">Duplicates merged</div></div>
      </div>

      <h3 style={{ marginTop: 20 }}>Sources</h3>
      <div className="fmeta">
        {Object.entries(ov.sources_status || {}).map(([src, st]) => (
          <span key={src} className="chip">{src}: {st.ran ? `ran (${st.count})` : 'not run'}</span>
        ))}
      </div>
    </div>
  )
}
