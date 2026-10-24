import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { Badge, StatusBadge, SourceChips, FindingDrawer } from '../components.jsx'

const SEVS = ['', 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFORMATIONAL']
const STATUSES = ['', 'open', 'regressed', 'accepted', 'fixed']

export default function Findings() {
  const [data, setData] = useState(null)
  const [err, setErr] = useState('')
  const [f, setF] = useState({ status: '', severity: '', service: '', source: '', q: '', sort: 'risk' })
  const [sel, setSel] = useState('')

  async function load() {
    setErr('')
    try {
      const params = {}
      Object.entries(f).forEach(([k, v]) => { if (v) params[k] = v })
      setData(await api.findings(params))
    } catch (e) { setErr(e.message) }
  }
  useEffect(() => { load() }, [f])

  const items = data?.items || []
  return (
    <div>
      <div className="pagehead"><h1>Findings {data ? `(${data.total})` : ''}</h1></div>
      <div className="toolbar">
        <select value={f.status} onChange={e => setF({ ...f, status: e.target.value })}>{STATUSES.map(s => <option key={s} value={s}>{s || 'all statuses'}</option>)}</select>
        <select value={f.severity} onChange={e => setF({ ...f, severity: e.target.value })}>{SEVS.map(s => <option key={s} value={s}>{s || 'all severities'}</option>)}</select>
        <input type="text" placeholder="service" value={f.service} onChange={e => setF({ ...f, service: e.target.value })} style={{ width: 110 }} />
        <input type="text" placeholder="source" value={f.source} onChange={e => setF({ ...f, source: e.target.value })} style={{ width: 110 }} />
        <input type="text" placeholder="search…" value={f.q} onChange={e => setF({ ...f, q: e.target.value })} />
        <select value={f.sort} onChange={e => setF({ ...f, sort: e.target.value })}><option value="risk">by risk</option><option value="age">by age</option><option value="seen">by last seen</option></select>
      </div>
      {err && <div className="banner error">{err}</div>}
      {!data ? <div className="skeleton" style={{ height: 60 }} /> : items.length === 0 ? <div className="empty">No findings match.</div> : (
        <table className="data">
          <thead><tr><th>Risk</th><th>Severity</th><th>Status</th><th>Finding</th><th>Service</th><th>Region</th><th>Sources</th></tr></thead>
          <tbody>
            {items.map(it => (
              <tr key={it.id} className="click" onClick={() => setSel(it.id)}>
                <td className="num">{Math.round(it.risk_score)}</td>
                <td><Badge sev={it.severity} /></td>
                <td><StatusBadge status={it.status} /></td>
                <td>{it.title}{it.on_path?.length ? <span className="chip" style={{ marginLeft: 6 }}>on path</span> : null}</td>
                <td>{it.service}</td><td>{it.region}</td>
                <td><SourceChips finding={it} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {sel && <FindingDrawer id={sel} onClose={() => setSel('')} onChanged={load} />}
    </div>
  )
}
