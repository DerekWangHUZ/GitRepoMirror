package main

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

type recordingEventSink struct {
	mu       sync.Mutex
	logs     []LogEvent
	progress []ProgressEvent
	changed  int
}

func (sink *recordingEventSink) Log(event LogEvent) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.logs = append(sink.logs, event)
}

func (sink *recordingEventSink) Progress(event ProgressEvent) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.progress = append(sink.progress, event)
}

func (sink *recordingEventSink) Changed() {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.changed++
}

func TestSyncEngineEmitsStableProgressContract(t *testing.T) {
	app := NewApp()
	events := &recordingEventSink{}
	app.events = events
	app.quietHook = func(_ string, _ string, _ ...string) (string, error) { return "", nil }
	app.commandHook = func(_ string, _ string, _ string, _ ...string) error { return nil }
	repo := Repository{
		ID: "progress", Mode: "mirror",
		Source: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/source.git"},
		Target: RemoteSpec{Platform: PlatformGeneric, CloneURL: "https://code.example/target.git"},
	}
	if err := app.performSync(repo, false, EnvironmentStatus{}); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, event := range events.progress {
		got = append(got, event.Label+":"+event.Status)
	}
	want := []string{"拉取源仓库:running", "拉取源仓库:done", "推送目标仓库:running", "推送目标仓库:done", "清理临时文件:done"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected progress contract: %#v", got)
	}
}

func TestSyncEngineMarksCloneFailure(t *testing.T) {
	app := NewApp()
	events := &recordingEventSink{}
	app.events = events
	app.quietHook = func(_ string, _ string, _ ...string) (string, error) { return "", nil }
	app.commandHook = func(_ string, _ string, _ string, _ ...string) error { return errors.New("offline") }
	err := app.performSync(Repository{ID: "failure", Mode: "mirror", Source: RemoteSpec{Platform: PlatformGeneric}}, false, EnvironmentStatus{})
	if err == nil {
		t.Fatal("expected clone failure")
	}
	got := events.progress
	if len(got) != 2 || got[0].Status != "running" || got[1].Status != "failed" {
		t.Fatalf("unexpected failure progress: %#v", got)
	}
}

func TestSaveSettingsValidationAndNormalization(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	app := NewApp()
	if err := app.SaveSettings(Settings{Proxy: "file:///tmp/proxy", Concurrency: 2}); err == nil {
		t.Fatal("expected unsafe proxy to be rejected")
	}
	if err := app.SaveSettings(Settings{Proxy: " http://127.0.0.1:7890 ", Prefix: " mirror- ", Concurrency: 3}); err != nil {
		t.Fatal(err)
	}
	data := app.GetData()
	if data.Settings.Proxy != "http://127.0.0.1:7890" || data.Settings.Prefix != "mirror-" || data.Settings.Concurrency != 3 {
		t.Fatalf("settings were not normalized: %#v", data.Settings)
	}
}

func TestRemoveRepositoryProtectsGitOnlyRemote(t *testing.T) {
	app := NewApp()
	data := app.store.Snapshot()
	data.Repositories = []Repository{{ID: "generic", ManagementMode: ManagementGitOnly}}
	app.store.Replace(data)
	if err := app.RemoveRepository("generic", true); err == nil {
		t.Fatal("expected remote deletion to be rejected")
	}
}
