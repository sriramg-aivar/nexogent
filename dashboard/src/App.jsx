import React, { useEffect, useState } from 'react'
import { api, getToken, setToken, clearToken } from './api.js'
import Login from './views/Login.jsx'
import Services from './views/Services.jsx'
import ServicePage from './views/ServicePage.jsx'
import ResourcePage from './views/ResourcePage.jsx'
import Integrations from './views/Integrations.jsx'

// Minimal hash router: #/ , #/service/<key> , #/resource/<id> , #/integrations
function parseHash() {
  const h = (window.location.hash || '#/').replace(/^#/, '')
  const parts = h.split('/').filter(Boolean) // ['service','ec2']
  if (parts[0] === 'service') return { view: 'service', key: decodeURIComponent(parts[1] || '') }
  if (parts[0] === 'resource') return { view: 'resource', id: decodeURIComponent(parts.slice(1).join('/')) }
  if (parts[0] === 'integrations') return { view: 'integrations' }
  return { view: 'services' }
}

export function navigate(hash) { window.location.hash = hash }

export default function App() {
  const [authRequired, setAuthRequired] = useState(true)
  const [authed, setAuthed] = useState(!!getToken())
  const [route, setRoute] = useState(parseHash())
  const [booting, setBooting] = useState(true)

  useEffect(() => {
    const onHash = () => setRoute(parseHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  useEffect(() => {
    api.authStatus()
      .then((s) => {
        setAuthRequired(s.auth_required)
        if (!s.auth_required) setAuthed(true)
      })
      .catch(() => {})
      .finally(() => setBooting(false))
  }, [])

  function onLogin(token) { setToken(token); setAuthed(true) }
  function onLogout() { clearToken(); setAuthed(false); navigate('#/') }

  if (booting) return <div className="boot">Loading Nexogent…</div>
  if (authRequired && !authed) return <Login onLogin={onLogin} />

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand" onClick={() => navigate('#/')} style={{ cursor: 'pointer' }}>
          <span className="logo">◆</span> Nexogent
          <span className="tag">superadmin · read-only</span>
        </div>
        <nav className="topnav">
          <a onClick={() => navigate('#/')}>Services</a>
          <a onClick={() => navigate('#/integrations')}>Integrations</a>
          <button className="btn ghost" onClick={onLogout}>Sign out</button>
        </nav>
      </header>

      {route.view === 'services' && <Services />}
      {route.view === 'service' && <ServicePage serviceKey={route.key} />}
      {route.view === 'resource' && <ResourcePage resourceId={route.id} />}
      {route.view === 'integrations' && <Integrations />}

      <footer className="foot">
        Read-only · recommend-only · never changes your account · Prowler + ScoutSuite + cloudsplaining + built-in checks
      </footer>
    </div>
  )
}
