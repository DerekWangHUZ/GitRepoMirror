package main

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncAllHonorsConcurrencyLimit(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	app.data.Settings.Concurrency = 2
	for i := 0; i < 4; i++ {
		app.data.Repositories = append(app.data.Repositories, Repository{
			ID: string(rune('a' + i)), Mode: "mirror", Status: "healthy",
			Source: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/team/source.git", DisplayName: "source"},
			Target: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/team/target.git", DisplayName: "target"},
		})
	}
	app.quietHook = func(_ string, _ string, _ ...string) (string, error) { return "", nil }
	var active, maximum atomic.Int32
	app.commandHook = func(_ string, _ string, _ string, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			current := active.Add(1)
			for {
				seen := maximum.Load()
				if current <= seen || maximum.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(40 * time.Millisecond)
			active.Add(-1)
		}
		return nil
	}
	if err := app.SyncAll(); err != nil {
		t.Fatal(err)
	}
	if maximum.Load() != 2 {
		t.Fatalf("expected concurrency 2, got %d", maximum.Load())
	}
}

func TestSyncFailureIsPersisted(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	app.data.Repositories = []Repository{{
		ID: "failure", Mode: "mirror", Status: "healthy",
		Source: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/team/source.git"},
		Target: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/team/target.git"},
	}}
	app.quietHook = func(_ string, _ string, _ ...string) (string, error) { return "", nil }
	app.commandHook = func(_ string, _ string, _ string, _ ...string) error { return errors.New("network unavailable") }
	if err := app.SyncRepository("failure"); err == nil {
		t.Fatal("expected sync failure")
	}
	repository, _ := app.repositoryByID("failure")
	if repository.Status != "failed" || repository.LastError == "" {
		t.Fatalf("failure was not persisted: %#v", repository)
	}
}
