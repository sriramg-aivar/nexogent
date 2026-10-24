import React, { useEffect, useState } from 'react'
import { api } from '../api.js'
import { Badge, PathDiagram } from '../components.jsx'

export default function Paths() {
  const [paths, setPaths] = useState(null)
  const [err, setErr] = useState('')
  useEffect(() => { api.paths().then(d => setPaths(d.paths || [])).catch(e => setErr(e.message)) }, [])

  if (err) return <div className="banner error">{err}</div>
  if (!paths) return <div className="skeleton" style={{ height: 80 }} />
  return (
    <div>
      <div className="pagehead"><h1>Attack paths ({paths.length})</h1></div>
      {paths.length === 0 ? <div className="empty">No attack paths found. Run a full audit if you haven't yet.</div> :
        paths.map(p => (
          <div className="path" key={p.id}>
            <div><Badge sev={p.severity} /> <b>{p.kind} — {p.title}</b> <span className="chip">risk {Math.round(p.score)}</span></div>
            <PathDiagram path={p} />
            <ul>{(p.evidence || []).map((e, i) => <li key={i}>{e}</li>)}</ul>
            <p><b>Break the chain:</b> {p.break_hint}</p>
          </div>
        ))}
    </div>
  )
}
