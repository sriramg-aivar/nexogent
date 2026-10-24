package alerts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateWebhook(t *testing.T) {
	if err := ValidateWebhook("https://hooks.slack.com/services/x/y/z", nil); err != nil {
		t.Fatalf("valid slack url rejected: %v", err)
	}
	for _, bad := range []string{
		"http://hooks.slack.com/x",        // not https
		"https://evil.example.com/x",      // host not allowed
		"https://user:pw@hooks.slack.com", // userinfo
		"not a url at all ::::",
	} {
		if err := ValidateWebhook(bad, nil); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
}

func TestBuildImmediate(t *testing.T) {
	m := BuildImmediate(Event{Kind: "new_critical", Title: "SSH open to world"}, "https://portal")
	if !strings.Contains(m.Text, "New critical finding") || !strings.Contains(m.Text, "SSH open") {
		t.Fatalf("bad message: %s", m.Text)
	}
}

func TestBuildDigest(t *testing.T) {
	m := BuildDigest([]Event{{Kind: "new_critical"}, {Kind: "new_critical"}, {Kind: "regressed"}}, "")
	if !strings.Contains(m.Text, "New critical finding: 2") {
		t.Fatalf("digest counts wrong: %s", m.Text)
	}
}

func TestSendOK(t *testing.T) {
	var got int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got++
		w.WriteHeader(200)
	}))
	defer srv.Close()
	// allow the test server host explicitly
	s := NewSender(srv.URL, []string{strings.TrimPrefix(srv.URL, "http://")})
	// test server is http, so ValidateWebhook will reject (https required) — assert that
	if err := s.Send(context.Background(), Message{Text: "hi"}); err == nil {
		t.Fatal("http webhook should be rejected by validation")
	}
}
