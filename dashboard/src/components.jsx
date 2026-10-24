import React, { useEffect, useState } from 'react'
import { api } from './api.js'

export function ratingClass(r) {
  r = (r || '').toLowerCase()
  if (r.includes('excellent') || r.includes('good')) return 'good'
  if (r.includes('needs')) return 'warn'
  return 'bad'
}

export function Badge({ sev }) { return <span className={'badge sev-' + sev}>{sev}</span> }
export function StatusBadge({ status }) { return <span className={'badge st-' + status}>{status}</span> }

export function SourceChips({ finding }) {
  const srcs = (finding.sources && finding.sources.length) ? finding.sources.map(s => s.source) : [finding.source]
  const uniq = [...new Set(srcs)]
  return <>{uniq.map(s => <span key={s} className="chip">{s}</span>)}
    {finding.framework_control && <span className="chip">{finding.framework_control}</span>}</>
}

// Animated SVG score ring.
export function ScoreRing({ score, rating }) {
  const r = 54, c = 2 * Math.PI * r
  const pct = Math.max(0, Math.min(100, score)) / 100
  const color = ratingClass(rating) === 'good' ? 'var(--good)' : ratingClass(rating) === 'warn' ? 'var(--warn)' : 'var(--bad)'
  return (
    <svg className="ring" viewBox="0 0 140 140" role="img" aria-label={`Score ${score} of 100`}>
      <circle cx="70" cy="70" r={r} fill="none" stroke="var(--line)" strokeWidth="12" />
      <circle className="val" cx="70" cy="70" r={r} fill="none" stroke={color} strokeWidth="12"
        strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - pct)} transform="rotate(-90 70 70)" />
      <text x="70" y="66" textAnchor="middle" fontSize="30" fontWeight="800">{score}</text>
      <text x="70" y="88" textAnchor="middle" fontSize="12" fill="var(--muted)">/ 100</text>
    </svg>
  )
}

// Simple SVG trend chart.
export function TrendChart({ points }) {
  if (!points || points.length < 2) return <p className="muted">Not enough history yet.</p>
  const w = 320, h = 80, pad = 6
  const xs = points.map((_, i) => pad + (i * (w - 2 * pad)) / (points.length - 1))
  const ys = points.map(p => h - pad - ((p.score || 0) / 100) * (h - 2 * pad))
  const d = xs.map((x, i) => `${i ? 'L' : 'M'}${x.toFixed(1)},${ys[i].toFixed(1)}`).join(' ')
  return (
    <svg width={w} height={h} role="img" aria-label="Score trend">
      <path d={d} fill="none" stroke="var(--accent)" strokeWidth="2" />
      {xs.map((x, i) => <circle key={i} cx={x} cy={ys[i]} r="2.5" fill="var(--accent)" />)}
    </svg>
  )
}

export function PathDiagram({ path }) {
  return (
    <div className="steps">
      {path.steps.map((s, i) => (
        <React.Fragment key={i}>
          <div className={'step' + (i === path.break_at ? ' brk' : '')}>
            <div><b>{s.label}</b></div>
            {s.detail && <div className="muted" style={{ fontSize: 12 }}>{s.detail}</div>}
          </div>
          {i < path.steps.length - 1 && <span className="arrow">→</span>}
        </React.Fragment>
      ))}
    </div>
  )
}

// Finding detail drawer with fix tabs, explanation, plan, accept-risk.
export function FindingDrawer({ id, onClose, onChanged }) {
  const [data, setData] = useState(null)
  const [tab, setTab] = useState('tf')
  const [err, setErr] = useState('')
  const [owner, setOwner] = useState(''); const [effort, setEffort] = useState(''); const [due, setDue] = useState('')
  const [accept, setAccept] = useState(false)
  const [reason, setReason] = useState(''); const [rOwner, setROwner] = useState(''); const [expiry, setExpiry] = useState('')

  useEffect(() => {
    api.finding(id).then(d => {
      setData(d)
      if (d.finding?.plan) { setOwner(d.finding.plan.owner || ''); setEffort(d.finding.plan.effort || ''); setDue(d.finding.plan.due_date || '') }
    }).catch(e => setErr(e.message))
  }, [id])

  async function savePlan() {
    try { await api.savePlan(id, owner, effort, due || undefined); onChanged && onChanged() } catch (e) { setErr(e.message) }
  }
  async function submitAccept() {
    try {
      await api.acceptRisk(data.finding.fingerprint, reason, rOwner, new Date(expiry).toISOString())
      setAccept(false); onChanged && onChanged(); onClose()
    } catch (e) { setErr(e.message) }
  }

  const f = data?.finding
  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <div className="drawer" onClick={e => e.stopPropagation()} role="dialog" aria-label="Finding detail">
        <button className="close" onClick={onClose} aria-label="Close">✕</button>
        {err && <div className="banner error">{err}</div>}
        {!f ? <div className="skeleton" /> : <>
          <Badge sev={f.severity} /> <StatusBadge status={f.status} />
          <h2 style={{ marginTop: 8 }}>{f.title}</h2>
          <p className="muted mono">{f.service} · {f.region} · {f.resource_id}</p>
          <div className="fmeta"><SourceChips finding={f} /><span className="chip">risk {Math.round(f.risk_score)}</span><span className="chip">age {f.age_days}d</span></div>

          {data.explanation && <div className="explain">
            <h4>Why it matters</h4><p>{data.explanation.why_dangerous || data.explanation.problem}</p>
            {data.explanation.impact && <p>{data.explanation.impact}</p>}
          </div>}

          {data.fix && <>
            <h4 style={{ marginTop: 16 }}>Fix — {data.fix.title} <span className="chip">{data.fix.label}</span></h4>
            <div className="tabs">
              <button className={tab === 'tf' ? 'active' : ''} onClick={() => setTab('tf')}>Terraform</button>
              <button className={tab === 'cli' ? 'active' : ''} onClick={() => setTab('cli')}>CLI</button>
            </div>
            <pre className="code">{tab === 'tf' ? data.fix.terraform : data.fix.cli}</pre>
            <p className="muted"><b>What could break:</b> {data.fix.what_could_break} <span className="chip">effort {data.fix.effort}</span></p>
          </>}

          <h4 style={{ marginTop: 16 }}>Remediation plan</h4>
          <div className="toolbar">
            <input type="text" placeholder="Owner" value={owner} onChange={e => setOwner(e.target.value)} />
            <select value={effort} onChange={e => setEffort(e.target.value)}><option value="">effort</option><option>S</option><option>M</option><option>L</option></select>
            <input type="date" value={due} onChange={e => setDue(e.target.value)} />
            <button className="btn" onClick={savePlan}>Save</button>
          </div>

          {!accept ? <button className="btn ghost" onClick={() => setAccept(true)}>Accept this risk…</button> : <div className="card" style={{ marginTop: 8 }}>
            <textarea placeholder="Reason (min 10 chars)" value={reason} onChange={e => setReason(e.target.value)} style={{ width: '100%' }} />
            <div className="toolbar">
              <input type="text" placeholder="Owner" value={rOwner} onChange={e => setROwner(e.target.value)} />
              <input type="date" value={expiry} onChange={e => setExpiry(e.target.value)} />
            </div>
            <button className="btn primary" onClick={submitAccept} disabled={!reason || !rOwner || !expiry}>Accept risk</button>
            <button className="btn ghost" onClick={() => setAccept(false)}>Cancel</button>
          </div>}
        </>}
      </div>
    </div>
  )
}

export function FindingRow({ f, onClick }) {
  return (
    <div className="finding" onClick={onClick} tabIndex={0} role="button"
      onKeyDown={e => { if (e.key === 'Enter') onClick() }}>
      <div className="risk">{Math.round(f.risk_score)}</div>
      <div className="fbody">
        <div className="ftitle">{f.title}</div>
        <div className="fmeta"><Badge sev={f.severity} /><StatusBadge status={f.status} /><SourceChips finding={f} /><span className="chip">{f.region}</span></div>
      </div>
      <span className="arrow">›</span>
    </div>
  )
}
