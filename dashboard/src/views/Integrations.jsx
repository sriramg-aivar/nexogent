import React, { useEffect, useState } from 'react'
import { api } from '../api.js'

export default function Integrations() {
  const [items, setItems] = useState([])
  const [err, setErr] = useState('')

  useEffect(() => {
    api.integrations()
      .then((d) => setItems(d.integrations || []))
      .catch((e) => setErr(e.message))
  }, [])

  return (
    <div>
      <div className="pagehead">
        <div>
          <h1>Enable these auditings</h1>
          <p className="muted">
            Threat-detection services. Disabled in v1 — available in a later version.
          </p>
        </div>
      </div>

      {err && <div className="banner error">{err}</div>}

      <div className="integrations">
        {items.map((it) => (
          <div key={it.key} className="integ">
            <div className="integ-body">
              <div className="integ-title">{it.name}</div>
              <div className="muted">{it.description}</div>
              {it.note && <div className="integ-note">{it.note}</div>}
            </div>
            <label className={'toggle disabled'} title="Disabled in v1">
              <input type="checkbox" checked={it.enabled} disabled readOnly />
              <span className="slider" />
            </label>
          </div>
        ))}
      </div>
    </div>
  )
}
