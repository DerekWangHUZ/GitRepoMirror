package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTripAndReplacement(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	data := defaultData()
	data.Settings.Proxy = "http://127.0.0.1:7890"
	data.Repositories = []Repository{{ID: "one", Source: RemoteSpec{DisplayName: "source/repository"}}}
	if err := saveData(data); err != nil {
		t.Fatal(err)
	}
	data.Settings.Concurrency = 4
	if err := saveData(data); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadData()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Settings.Proxy != data.Settings.Proxy || loaded.Settings.Concurrency != 4 || len(loaded.Repositories) != 1 {
		t.Fatalf("round trip mismatch: %#v", loaded)
	}
	path, err := dataFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "GitRepoMirror") {
		t.Fatalf("unexpected data path: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsDamagedConfiguration(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	path, err := dataFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadData(); err == nil {
		t.Fatal("expected damaged configuration error")
	}
}
