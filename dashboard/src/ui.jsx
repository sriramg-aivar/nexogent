import React from 'react'

export const SEV_ORDER = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFORMATIONAL']

export function sevClass(s) {
  return 'sev sev-' + (s || 'INFORMATIONAL').toLowerCase()
}

export function ratingClass(rating) {
  const r = (rating || '').toLowerCase()
  if (r.includes('excellent') || r.includes('good')) return 'rating good'
  if (r.includes('needs')) return 'rating warn'
  return 'rating bad'
}

const SERVICE_ICON = {
  ec2: '🖥️', s3: '🪣', iam: '🔑', lambda: 'λ', rds: '🗄️',
  dynamodb: '⚡', eks: '☸️', cloudtrail: '📜', vpc: '🌐',
}
export function serviceIcon(key) { return SERVICE_ICON[key] || '☁️' }

// Finding explanation drawer (shared across views).
export function Drawer({ finding, explanation, explaining, onClose }) {
  if (!finding) return null
  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <div className="drawer" onClick={(e) => e.stopPropagation()}>
        <button className="close" onClick={onClose}>✕</button>
        <span className={sevClass(finding.severity)}>{finding.severity}</span>
        <h2>{finding.title}</h2>
        <p className="muted">{finding.service} · {finding.region} · {finding.resource_id}</p>
        <div className="fmeta" style={{ marginBottom: 12 }}>
          <span className="src">{finding.source}</span>
          {finding.framework_control && <span className="ctrl">{finding.framework_control}</span>}
          <span className="risk-inline">risk {finding.risk_score}</span>
        </div>
        {explaining && <p>Generating explanation…</p>}
        {explanation && (
          <div className="explain">
            <h4>Problem</h4><p>{explanation.problem}</p>
            {explanation.why_dangerous && (<><h4>Why it's dangerous</h4><p>{explanation.why_dangerous}</p></>)}
            {explanation.impact && (<><h4>Impact</h4><p>{explanation.impact}</p></>)}
            <h4>How to fix</h4>
            <ol>{(explanation.how_to_fix || []).map((s, i) => <li key={i}>{s}</li>)}</ol>
            {explanation.aws_console_path && (<><h4>Console path</h4><p>{explanation.aws_console_path}</p></>)}
            <div className="provider">
              via <b>{explanation.provider}</b>{explanation.model ? ` (${explanation.model})` : ''}
              {explanation.note && <div className="note">{explanation.note}</div>}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export function FindingRow({ f, onClick }) {
  return (
    <div className="finding" onClick={onClick}>
      <div className="risk" title="risk score">{f.risk_score}</div>
      <div className="fbody">
        <div className="ftitle">{f.title}</div>
        <div className="fmeta">
          <span className={sevClass(f.severity)}>{f.severity}</span>
          <span className="src">{f.source}</span>
          {f.framework_control && <span className="ctrl">{f.framework_control}</span>}
          <span className="svc">{f.region}</span>
        </div>
      </div>
      <div className="chev">›</div>
    </div>
  )
}
