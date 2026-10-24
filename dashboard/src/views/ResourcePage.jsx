import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { ServiceIcon } from '../icons.jsx'
import { FindingRow, FindingDrawer } from '../components.jsx'
import { navigate } from '../App.jsx'

export default function ResourcePage({ resourceId }) {
  const [resource, setResource] = useState(null)
  const [findings, setFindings] = useState([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [sel, setSel] = useState('')

  async function load() {
    setErr('')
    try { const d = await api.resource(resourceId); setResource(d.resource); setFindings(d.findings || []) }
    catch (e) { setErr(e.message) }
  }
  useEffect(() => { load() /* eslint-disable-next-line */ }, [resourceId])
  async function audit() { setBusy(true); try { const d = await api.audit(resource?.service || '', resourceId); setFindings(d.findings || []) } catch (e) { setErr('Audit failed: ' + e.message) } finally { setBusy(false) } }

  const svc = resource?.service || ''
  return (
    <div>
      <div className="crumbs">
        <a onClick={() => navigate('#/inventory')}>Inventory</a> › {svc && <><a onClick={() => navigate('#/service/' + svc)}>{svc.toUpperCase()}</a> › </>}resource
      </div>
      <div className="pagehead">
        <div>
          <h1 style={{ display: 'flex', alignItems: 'center', gap: 8 }}><ServiceIcon svc={svc} /> {resource?.name || resourceId}</h1>
          <p className="muted mono">{resourceId}</p>
          {resource && <p className="muted">{resource.type} · {resource.region}</p>}
        </div>
        <button className="btn primary" onClick={audit} disabled={busy}>{busy ? 'Auditing…' : 'Quick audit this resource'}</button>
      </div>
      {err && <div className="banner error">{err}</div>}
      <div className={'banner ' + (findings.length ? 'warn' : 'info')}>
        {findings.length ? `⚠ ${findings.length} finding${findings.length === 1 ? '' : 's'} on this resource` : '✓ No findings on this resource'}
      </div>
      {findings.map(f => <FindingRow key={f.id} f={f} onClick={() => setSel(f.id)} />)}
      {sel && <FindingDrawer id={sel} onClose={() => setSel('')} onChanged={load} />}
    </div>
  )
}
