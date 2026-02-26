package storage

import (
	"context"
	"testing"
)

func TestNewLogStoreAndInsert(t *testing.T) {
	store, err := NewLogStore(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("NewLogStore error: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.InsertSend(context.Background(), "hello", "123@g.us"); err != nil {
		t.Fatalf("InsertSend error: %v", err)
	}

	if err := store.InsertUnauthorized(context.Background(), "127.0.0.1", "Unauthorized", map[string][]string{"Authorization": {"bad"}}); err != nil {
		t.Fatalf("InsertUnauthorized error: %v", err)
	}
}
