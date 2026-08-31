package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
		backup, backupErr := backupDamagedData(path)
		if backupErr != nil {
			return StoreData{}, fmt.Errorf("配置文件格式无效，且备份失败: %v: %w", backupErr, err)
		}
		return StoreData{}, fmt.Errorf("配置文件格式无效，已备份到 %s: %w", backup, err)
	}
	if data.Settings.Concurrency < 1 || data.Settings.Concurrency > 4 {
		data.Settings.Concurrency = 2
	}
	return data, nil
}

func backupDamagedData(path string) (string, error) {
	stamp := time.Now().Format("20060102-150405.000000000")
	backup := filepath.Join(filepath.Dir(path), "data.corrupt-"+stamp+".json")
	if err := os.Rename(path, backup); err != nil {
		return "", err
	}
	return backup, nil
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
