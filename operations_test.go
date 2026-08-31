package main

import (
	"context"
	"testing"
)

func TestOperationRegistryPreventsBatchOverlapAndCancelsTasks(t *testing.T) {
	registry := newOperationRegistry()
	ctx, finish, err := registry.begin(context.Background(), "repo", false)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	if _, err := registry.beginBatch(); err == nil {
		t.Fatal("expected batch sync to be rejected while a repository is active")
	}
	registry.cancelAll()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected active operation context to be cancelled")
	}
}

func TestOperationRegistryRejectsIndividualSyncDuringBatch(t *testing.T) {
	registry := newOperationRegistry()
	finish, err := registry.beginBatch()
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	if _, _, err := registry.begin(context.Background(), "repo", false); err == nil {
		t.Fatal("expected individual sync to be rejected during batch sync")
	}
}
