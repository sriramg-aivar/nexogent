import React, { useEffect, useState } from 'react'
import { api } from '../api.js'

const NATIVE = [
  ['guardduty', 'Amazon GuardDuty', 'Continuous threat detection (active threats).'],
  ['securityhub', 'AWS Security Hub', 'Aggregated findings and compliance standards.'],
  ['inspector', 'Amazon Inspector', 'Automated vulnerability (CVE) scanning.'],
]

export default function Settings() {
  const [s, setS] = useState(null)
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')

  useEffect(() => { api.settings().then(setS).catch(e => setErr(e.message)) }, [])

  function set(path, val) {
    setS(prev => {
      const n = structuredClone(prev)
      let o = n; const parts = path.split('.')
      for (let i = 0; i < parts.length - 1; i++) o = o[parts[i]]
      o[parts[parts.length - 1]] = val
      return n
    })
  }

  async function save() {
    setErr(''); setMsg('')
    try { await api.saveSettings(s); setMsg('Saved.') } catch (e) { setErr(e.message) }
  }

  if (err && !s) return <div className="banner error">{err}</div>
  if (!s) return <div className="skeleton" style={{ height: 80 }} />
  const w = s.rubric.weights
  const sum = (w.severity + w.exposure + w.reachability + w.criticality).toFixed(2)

  return (
    <div>
      <div className="pagehead"><h1>Settings</h1><button className="btn primary" onClick={save}>Save</button></div>
      {err && <div className="banner error">{err}</div>}
      {msg && <div className="banner info">{msg}</div>}

      <div className="card"><h3>Branding</h3>
        <div className="toolbar">
          <input type="text" placeholder="MSP name" value={s.branding.msp_name || ''} onChange={e => set('branding.msp_name', e.target.value)} />
          <input type="text" placeholder="Client name" value={s.branding.client_name || ''} onChange={e => set('branding.client_name', e.target.value)} />
        </div>
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Sources</h3>
        {Object.keys(s.sources || {}).map(k => (
          <label key={k} style={{ display: 'flex', gap: 8, alignItems: 'center', padding: '4px 0' }}>
            <span className={'toggle' + (s.sources[k] ? ' on' : '')} onClick={() => set('sources.' + k, !s.sources[k])}><input type="checkbox" checked={!!s.sources[k]} readOnly /><span className="slider" /></span>
            {k}
          </label>
        ))}
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Schedule</h3>
        <select value={s.scan_interval} onChange={e => set('scan_interval', e.target.value)}>
          {['6h', '12h', '24h', 'off'].map(v => <option key={v} value={v}>{v}</option>)}
        </select>
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Scoring rubric</h3>
        {['severity', 'exposure', 'reachability', 'criticality'].map(k => (
          <div key={k} className="toolbar">
            <label style={{ width: 110 }}>{k}</label>
            <input type="range" min="0" max="1" step="0.05" value={w[k]} onChange={e => set('rubric.weights.' + k, parseFloat(e.target.value))} />
            <span className="num">{w[k].toFixed(2)}</span>
          </div>
        ))}
        <p className={'muted' + (Math.abs(sum - 1) > 0.001 ? ' count bad' : '')}>sum = {sum} (must be 1.00)</p>
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Attack paths</h3>
        {Object.keys(s.paths || {}).map(k => (
          <label key={k} style={{ display: 'inline-flex', gap: 6, alignItems: 'center', marginRight: 14 }}>
            <span className={'toggle' + (s.paths[k] ? ' on' : '')} onClick={() => set('paths.' + k, !s.paths[k])}><input type="checkbox" checked={!!s.paths[k]} readOnly /><span className="slider" /></span>{k}
          </label>
        ))}
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Reviewer & alerts</h3>
        <div className="toolbar">
          <input type="text" placeholder="Reviewer name" value={s.reviewer || ''} onChange={e => set('reviewer', e.target.value)} />
          <select value={s.alerts?.mode || 'immediate'} onChange={e => set('alerts.mode', e.target.value)}><option value="immediate">immediate</option><option value="digest">digest</option></select>
          <span className="chip">{s.alerts?.webhook_set ? 'webhook set' : 'no webhook'}</span>
        </div>
      </div>

      <div className="card" style={{ marginTop: 12 }}><h3>Enable these auditings</h3>
        <p className="muted">Paid AWS services, disabled in this free build.</p>
        {NATIVE.map(([k, name, desc]) => (
          <div key={k} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderTop: '1px solid var(--line)' }}>
            <div><b>{name}</b><div className="muted">{desc}</div></div>
            <span className="toggle disabled" title="Disabled in the free build"><input type="checkbox" disabled /><span className="slider" /></span>
          </div>
        ))}
      </div>
    </div>
  )
}
