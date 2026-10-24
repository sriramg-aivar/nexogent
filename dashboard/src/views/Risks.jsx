import React, { useEffect, useState } from 'react'
import { api } from '../api.js'

export default function Risks() {
  const [risks, setRisks] = useState(null)
  const [err, setErr] = useState('')
  async function load() { try { const d = await api.risks(); setRisks(d.risks || []) } catch (e) { setErr(e.message) } }
  useEffect(() => { load() }, [])
  async function revoke(fp) { try { await api.revokeRisk(fp); load() } catch (e) { setErr(e.message) } }

  function daysLeft(iso) { return Math.round((new Date(iso) - Date.now()) / 86400000) }

  if (err) return <div className="banner error">{err}</div>
  if (!risks) return <div className="skeleton" style={{ height: 60 }} />
  return (
    <div>
      <div className="pagehead"><h1>Accepted risks ({risks.length})</h1></div>
      {risks.length === 0 ? <div className="empty">No accepted risks. Accept one from a finding's drawer.</div> : (
        <table className="data">
          <thead><tr><th>Reason</th><th>Owner</th><th>Expires</th><th></th></tr></thead>
          <tbody>
            {risks.map(r => {
              const d = daysLeft(r.expires_at)
              return (
                <tr key={r.fingerprint}>
                  <td>{r.reason}</td><td>{r.owner}</td>
                  <td className={d < 0 ? 'count bad' : ''}>{new Date(r.expires_at).toLocaleDateString()} {d >= 0 ? `(in ${d}d)` : '(expired)'}</td>
                  <td><button className="btn ghost" onClick={() => revoke(r.fingerprint)}>Revoke</button></td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}
