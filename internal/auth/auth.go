// Package auth provides a minimal single-role (superadmin) auth gate for v1.
//
// v1 is intentionally simple: one shared superadmin password (the client).
// On login we issue an opaque bearer token (random, in-memory) that the SPA
// sends on every API call. This keeps the portal from being fully public.
// Later versions can swap this for Cognito/OIDC without changing the handlers.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

const Role = "superadmin"

// Gate holds the configured password and issued session tokens.
type Gate struct {
	password string
	enabled  bool

	mu     sync.RWMutex
	tokens map[string]time.Time // token -> expiry
	ttl    time.Duration
}

// New builds a Gate. If password is empty, auth is DISABLED (open) and a
// warning should be logged by the caller — useful for local dev only.
func New(password string) *Gate {
	return &Gate{
		password: password,
		enabled:  password != "",
		tokens:   map[string]time.Time{},
		ttl:      12 * time.Hour,
	}
}

func (g *Gate) Enabled() bool { return g.enabled }

// Login checks the password and, on success, returns a fresh session token.
func (g *Gate) Login(password string) (string, bool) {
	if !g.enabled {
		return "dev-open", true
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(g.password)) != 1 {
		return "", false
	}
	tok := randToken()
	g.mu.Lock()
	g.tokens[tok] = time.Now().Add(g.ttl)
	g.mu.Unlock()
	return tok, true
}

// valid reports whether a token is known and unexpired.
func (g *Gate) valid(tok string) bool {
	if !g.enabled {
		return true
	}
	g.mu.RLock()
	exp, ok := g.tokens[tok]
	g.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		g.mu.Lock()
		delete(g.tokens, tok)
		g.mu.Unlock()
		return false
	}
	return true
}

// Middleware protects handlers. Paths in the allow list bypass the check
// (health + login + the SPA itself, which the frontend serves).
func (g *Gate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.enabled || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		tok := bearer(r)
		if tok == "" || !g.valid(tok) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","role":"superadmin"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isPublicPath(p string) bool {
	switch p {
	case "/health", "/api/v1/health", "/api/v1/auth/login", "/api/v1/auth/status":
		return true
	}
	return false
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	// Allow a token query param as a fallback for simple clients.
	return r.URL.Query().Get("token")
}

func randToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
