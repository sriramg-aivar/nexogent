// Package session issues and verifies stateless, signed session tokens, and
// throttles logins (F14). Tokens survive restarts and work across replicas.
package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Payload is the token body.
type Payload struct {
	Sub string `json:"sub"`
	IAT int64  `json:"iat"`
	Exp int64  `json:"exp"`
	JTI string `json:"jti"`
}

// Manager signs and verifies tokens with an HMAC secret.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// New builds a Manager. The secret should be >= 32 bytes; a short secret is
// still accepted but the caller should warn.
func New(secret string, ttl time.Duration) *Manager {
	if ttl == 0 {
		ttl = 8 * time.Hour
	}
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// RandomSecret returns a fresh 48-byte hex secret for when none is configured.
func RandomSecret() string {
	b := make([]byte, 48)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Issue creates a signed token for the subject.
func (m *Manager) Issue(sub string) string {
	now := time.Now()
	p := Payload{Sub: sub, IAT: now.Unix(), Exp: now.Add(m.ttl).Unix(), JTI: RandomSecret()[:16]}
	raw, _ := json.Marshal(p)
	b64 := base64.RawURLEncoding.EncodeToString(raw)
	return b64 + "." + m.sign(b64)
}

// Verify checks a token's signature and expiry, returning the payload.
func (m *Manager) Verify(token string) (Payload, error) {
	var p Payload
	dot := -1
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			dot = i
			break
		}
	}
	if dot <= 0 {
		return p, errors.New("malformed token")
	}
	b64, sig := token[:dot], token[dot+1:]
	if !hmac.Equal([]byte(sig), []byte(m.sign(b64))) {
		return p, errors.New("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return p, errors.New("bad payload")
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, errors.New("bad payload")
	}
	if time.Now().Unix() > p.Exp {
		return p, errors.New("expired")
	}
	return p, nil
}

func (m *Manager) sign(b64 string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(b64))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// Throttle limits login attempts per client IP and globally.
type Throttle struct {
	mu       sync.Mutex
	perIP    map[string][]time.Time
	global   []time.Time
	window   time.Duration
	maxIP    int
	maxTotal int
}

func NewThrottle() *Throttle {
	return &Throttle{perIP: map[string][]time.Time{}, window: 10 * time.Minute, maxIP: 5, maxTotal: 50}
}

// Allow reports whether a login attempt from ip is permitted right now, and the
// retry-after duration when it is not.
func (t *Throttle) Allow(ip string, now time.Time) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	cut := now.Add(-t.window)
	t.perIP[ip] = after(t.perIP[ip], cut)
	t.global = after(t.global, cut)
	if len(t.perIP[ip]) >= t.maxIP {
		return false, t.window - now.Sub(t.perIP[ip][0])
	}
	if len(t.global) >= t.maxTotal {
		return false, t.window - now.Sub(t.global[0])
	}
	return true, 0
}

// Record registers a failed attempt (only failures count toward the limit).
func (t *Throttle) Record(ip string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.perIP[ip] = append(t.perIP[ip], now)
	t.global = append(t.global, now)
}

func after(ts []time.Time, cut time.Time) []time.Time {
	var out []time.Time
	for _, x := range ts {
		if x.After(cut) {
			out = append(out, x)
		}
	}
	return out
}

var _ = fmt.Sprint
