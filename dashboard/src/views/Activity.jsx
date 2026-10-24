import React, { useEffect, useState } from 'react'
import { api, download } from '../api.js'

export default function Activity() {
  const [entries, setEntries] = useState(null)
  const [err, setErr] = useState('')
  useEffect(() => { api.activity().then(d => setEntries(d.entries || [])).catch(e => setErr(e.message)) }, [])

  if (err) return <div className="banner error">{err}</div>
  if (!entries) return <div className="skeleton" style={{ height: 60 }} />
  return (
    <div>
      <div className="pagehead">
        <h1>Activity</h1>
        <div className="toolbar" style={{ margin: 0 }}>
          <button className="btn" onClick={() => download('/api/v1/activity?format=csv', 'activity.csv')}>Export CSV</button>
          <button className="btn" onClick={() => download('/api/v1/activity?format=json', 'activity.json')}>Export JSON</button>
        </div>
      </div>
      {entries.length === 0 ? <div className="empty">No activity recorded yet.</div> : (
        <table className="data">
          <thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Target</th><th>Detail</th></tr></thead>
          <tbody>{entries.map((e, i) => (
            <tr key={i}><td className="mono">{new Date(e.at).toLocaleString()}</td><td>{e.actor}</td><td><span className="chip">{e.action}</span></td><td className="mono">{e.target}</td><td>{e.detail}</td></tr>
          ))}</tbody>
        </table>
      )}
    </div>
  )
}
