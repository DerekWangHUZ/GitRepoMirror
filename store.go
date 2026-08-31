package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func defaultData() StoreData {
	return StoreData{Settings: Settings{Concurrency: 2}}
}

func dataFilePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "GitRepoMirror", "data.json"), nil
}

func loadData() (StoreData, error) {
	path, err := dataFilePath()
	if err != nil {
		return StoreData{}, err
	}
	payload, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultData(), nil
	}
	if err != nil {
		return StoreData{}, err
	}
	data := defaultData()
	if err := json.Unmarshal(payload, &data); err != nil {
		return StoreData{}, fmt.Errorf("配置文件格式无效: %w", err)
	}
	if data.Settings.Concurrency < 1 || data.Settings.Concurrency > 4 {
		data.Settings.Concurrency = 2
	}
	return data, nil
}

func saveData(data StoreData) error {
	path, err := dataFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "data-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
