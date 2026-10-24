package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryKVRoundTrip(t *testing.T) {
	kv := NewMemory().KV()
	ctx := context.Background()
	type row struct {
		Name string `json:"name"`
	}
	if err := kv.PutItem(ctx, "settings", "current", row{"x"}); err != nil {
		t.Fatal(err)
	}
	var got row
	ok, err := kv.GetItem(ctx, "settings", "current", &got)
	if err != nil || !ok || got.Name != "x" {
		t.Fatalf("get: ok=%v err=%v got=%+v", ok, err, got)
	}
}

func TestMemoryKVQueryNewestFirst(t *testing.T) {
	kv := NewMemory().KV()
	ctx := context.Background()
	for _, sk := range []string{"2024-01-01", "2024-02-01", "2024-03-01"} {
		_ = kv.PutItem(ctx, "history", sk, map[string]string{"sk": sk})
	}
	var rows []map[string]string
	if err := kv.Query(ctx, "history", "", 0, true, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0]["sk"] != "2024-03-01" {
		t.Fatalf("newest-first query wrong: %+v", rows)
	}
}

func TestMemoryLease(t *testing.T) {
	kv := NewMemory().KV()
	ctx := context.Background()
	ok, _ := kv.AcquireLease(ctx, "scheduler", "host-a", time.Minute)
	if !ok {
		t.Fatal("first acquire should succeed")
	}
	ok2, _ := kv.AcquireLease(ctx, "scheduler", "host-b", time.Minute)
	if ok2 {
		t.Fatal("second owner should be blocked while lease held")
	}
	ok3, _ := kv.AcquireLease(ctx, "scheduler", "host-a", time.Minute)
	if !ok3 {
		t.Fatal("same owner should renew")
	}
}
