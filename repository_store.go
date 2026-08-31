package main

import "sync"

// RepositoryStore owns the in-memory snapshot and serializes atomic file replacement.
type RepositoryStore struct {
	mu   sync.RWMutex
	data StoreData
}

func NewRepositoryStore(data StoreData) *RepositoryStore { return &RepositoryStore{data: data} }

func (store *RepositoryStore) Snapshot() StoreData {
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := store.data
	result.Repositories = append([]Repository(nil), store.data.Repositories...)
	return result
}
func (store *RepositoryStore) Replace(data StoreData) {
	store.mu.Lock()
	store.data = data
	store.mu.Unlock()
}
func (store *RepositoryStore) Settings() Settings {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.data.Settings
}
func (store *RepositoryStore) SaveSettings(settings Settings) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.data.Settings = settings
	return saveData(store.data)
}
func (store *RepositoryStore) Find(id string) (Repository, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, repository := range store.data.Repositories {
		if repository.ID == id {
			return repository, true
		}
	}
	return Repository{}, false
}
func (store *RepositoryStore) Upsert(repository Repository) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	for index := range store.data.Repositories {
		if store.data.Repositories[index].ID == repository.ID {
			store.data.Repositories[index] = repository
			return saveData(store.data)
		}
	}
	store.data.Repositories = append(store.data.Repositories, repository)
	return saveData(store.data)
}
func (store *RepositoryStore) Remove(id string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	filtered := make([]Repository, 0, len(store.data.Repositories)-1)
	for _, repository := range store.data.Repositories {
		if repository.ID != id {
			filtered = append(filtered, repository)
		}
	}
	store.data.Repositories = filtered
	return saveData(store.data)
}

func (a *App) repositoryByID(id string) (Repository, bool) { return a.store.Find(id) }
func (a *App) upsertRepository(repository Repository) error {
	err := a.store.Upsert(repository)
	if err == nil {
		a.emitChanged()
	}
	return err
}
