package session

import (
	"testing"
	"time"
)

func TestIssueVerifyRoundTrip(t *testing.T) {
	m := New("a-very-long-session-secret-of-32b+", time.Hour)
	tok := m.Issue("superadmin")
	p, err := m.Verify(tok)
	if err != nil || p.Sub != "superadmin" {
		t.Fatalf("round trip failed: %v %+v", err, p)
	}
}

func TestTamperedFails(t *testing.T) {
	m := New("secret-secret-secret-secret-secret", time.Hour)
	tok := m.Issue("superadmin")
	if _, err := m.Verify(tok + "x"); err == nil {
		t.Fatal("tampered signature should fail")
	}
}

func TestExpired(t *testing.T) {
	m := New("secret-secret-secret-secret-secret", -time.Second) // already expired ttl
	tok := m.Issue("superadmin")
	if _, err := m.Verify(tok); err == nil {
		t.Fatal("expired token should fail")
	}
}

func TestAcrossReplicas(t *testing.T) {
	// two managers with the same secret must accept each other's tokens
	a := New("shared-secret-shared-secret-shared", time.Hour)
	b := New("shared-secret-shared-secret-shared", time.Hour)
	tok := a.Issue("superadmin")
	if _, err := b.Verify(tok); err != nil {
		t.Fatalf("second replica rejected token: %v", err)
	}
}

func TestThrottle(t *testing.T) {
	th := NewThrottle()
	now := time.Now()
	ip := "1.2.3.4"
	for i := 0; i < 5; i++ {
		if ok, _ := th.Allow(ip, now); !ok {
			t.Fatalf("attempt %d should be allowed", i)
		}
		th.Record(ip, now)
	}
	if ok, retry := th.Allow(ip, now); ok || retry <= 0 {
		t.Fatal("6th attempt should be throttled with retry-after")
	}
}
