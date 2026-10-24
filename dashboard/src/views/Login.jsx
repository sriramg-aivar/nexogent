import React, { useState } from 'react'
import { api } from '../api.js'

export default function Login({ onLogin }) {
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setBusy(true); setErr('')
    try {
      const { token } = await api.login(password)
      onLogin(token)
    } catch (e) {
      setErr('Invalid password')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="login" onSubmit={submit}>
        <div className="brand big"><span className="logo">◆</span> Nexogent</div>
        <p className="muted">AWS security posture portal — sign in as <b>superadmin</b>.</p>
        <input
          type="password"
          placeholder="Superadmin password"
          value={password}
          autoFocus
          onChange={(e) => setPassword(e.target.value)}
        />
        {err && <div className="banner error">{err}</div>}
        <button className="btn primary" type="submit" disabled={busy || !password}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
        <p className="foot" style={{ marginTop: 18 }}>Read-only · recommend-only</p>
      </form>
    </div>
  )
}
