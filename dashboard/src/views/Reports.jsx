import React, { useEffect, useState } from 'react'
import { api, download } from '../api.js'

export default function Reports() {
  const [list, setList] = useState(null)
  const [err, setErr] = useState('')
  const [sel, setSel] = useState(null)
  const [scope, setScope] = useState('account')

  async function load() { try { const d = await api.reportsList(); setList(d.reports || []) } catch (e) { setErr(e.message) } }
  useEffect(() => { load() }, [])

  async function create() {
    setErr('')
    try { const m = await api.createReport(scope); await load(); open(m.id) } catch (e) { setErr(e.message) }
  }
  async function open(id) { try { setSel(await api.report(id)) } catch (e) { setErr(e.message) } }
  async function act(id, action) {
    const note = action === 'reject' ? prompt('Reason for rejection:') || '' : ''
    try { await api.reportTransition(id, action, note); await load(); await open(id) } catch (e) { setErr(e.message) }
  }

  if (err) return <div className="banner error">{err}</div>
  if (!list) return <div className="skeleton" style={{ height: 60 }} />
  return (
    <div>
      <div className="pagehead">
        <h1>Reports</h1>
        <div className="toolbar" style={{ margin: 0 }}>
          <select value={scope} onChange={e => setScope(e.target.value)}><option value="account">Whole account</option></select>
          <button className="btn primary" onClick={create}>New report</button>
        </div>
      </div>
      {list.length === 0 ? <div className="empty">No reports yet.</div> : (
        <table className="data">
          <thead><tr><th>ID</th><th>Scope</th><th>Version</th><th>Status</th><th></th></tr></thead>
          <tbody>{list.map(m => (
            <tr key={m.id} className="click" onClick={() => open(m.id)}>
              <td className="mono">{m.id.slice(0, 8)}</td><td>{m.scope}</td><td className="num">{m.version}</td>
              <td><span className={'badge st-' + (m.status === 'approved' ? 'fixed' : m.status === 'rejected' ? 'regressed' : 'open')}>{m.status}</span></td>
              <td className="arrow">›</td>
            </tr>
          ))}</tbody>
        </table>
      )}

      {sel && (
        <div className="modal-backdrop" onClick={() => setSel(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ width: 520 }}>
            <button className="close" onClick={() => setSel(null)}>✕</button>
            <h2>Report {sel.meta.id.slice(0, 8)}</h2>
            <p className="muted">{sel.meta.scope} · v{sel.meta.version} · <b>{sel.meta.status}</b></p>
            <p>{sel.preview}</p>
            <div className="toolbar">
              {sel.meta.status === 'draft' && <button className="btn" onClick={() => act(sel.meta.id, 'submit')}>Submit for review</button>}
              {sel.meta.status === 'in_review' && <>
                <button className="btn primary" onClick={() => act(sel.meta.id, 'approve')}>Approve</button>
                <button className="btn" onClick={() => act(sel.meta.id, 'reject')}>Reject</button>
              </>}
            </div>
            <h3>Download</h3>
            <div className="toolbar">
              <button className="btn" onClick={() => download(`/api/v1/reports/${sel.meta.id}/download?format=docx`, 'report.docx')}>Word</button>
              <button className="btn" onClick={() => download(`/api/v1/reports/${sel.meta.id}/download?format=md`, 'report.md')}>Markdown</button>
              <button className="btn" onClick={() => download(`/api/v1/reports/${sel.meta.id}/download?format=html`, 'report.html')}>HTML</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
