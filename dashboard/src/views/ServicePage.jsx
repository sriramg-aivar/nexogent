import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { ServiceIcon } from '../icons.jsx'
import { navigate } from '../App.jsx'

export default function ServicePage({ serviceKey }) {
  const [data, setData] = useState(null)
  const [region, setRegion] = useState('')
  const [rtype, setRtype] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function load(r = region, t = rtype) {
    setErr('')
    try { setData(await api.resources(serviceKey, r, t)) } catch (e) { setErr(e.message) }
  }
  useEffect(() => { setRegion(''); setRtype(''); load('', '') /* eslint-disable-next-line */ }, [serviceKey])
  async function audit() { setBusy(true); try { await api.audit(serviceKey, ''); await load() } catch (e) { setErr('Audit failed: ' + e.message) } finally { setBusy(false) } }

  const resources = data?.resources || []
  return (
    <div>
      <div className="crumbs"><a onClick={() => navigate('#/inventory')}>Inventory</a> › {serviceKey.toUpperCase()}</div>
      <div className="pagehead">
        <h1 style={{ display: 'flex', alignItems: 'center', gap: 8 }}><ServiceIcon svc={serviceKey} /> {serviceKey.toUpperCase()}</h1>
        <button className="btn primary" onClick={audit} disabled={busy}>{busy ? 'Auditing…' : 'Audit this service'}</button>
      </div>
      <div className="toolbar">
        <label>Region</label>
        <select value={region} onChange={e => { setRegion(e.target.value); load(e.target.value, rtype) }}>
          <option value="">all</option>{(data?.regions || []).map(r => <option key={r} value={r}>{r}</option>)}
        </select>
        <label>Type</label>
        <select value={rtype} onChange={e => { setRtype(e.target.value); load(region, e.target.value) }}>
          <option value="">all</option>{(data?.types || []).map(t => <option key={t} value={t}>{t}</option>)}
        </select>
        <span className="muted">{resources.length} resource{resources.length === 1 ? '' : 's'}</span>
      </div>
      {err && <div className="banner error">{err}</div>}
      {resources.length === 0 ? <div className="banner info">No resources{region ? ' in ' + region : ''}.</div> : (
        <table className="data">
          <thead><tr><th>Name</th><th>Type</th><th>Region</th><th>Findings</th><th></th></tr></thead>
          <tbody>
            {resources.map(r => (
              <tr key={r.id} className="click" onClick={() => navigate('#/resource/' + encodeURIComponent(r.id))}>
                <td className="mono">{r.name || r.id}</td><td>{r.type}</td>
                <td><span className="chip">{r.region}</span></td>
                <td><span className={'count ' + (r.finding_count ? 'bad' : 'good')}>{r.finding_count}</span></td>
                <td className="arrow">›</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
