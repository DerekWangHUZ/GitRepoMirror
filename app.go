package main

import (
	"context"
	"sort"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the stable Wails-facing facade. Business operations live in focused
// service modules; these methods remain here to preserve the generated API.
type App struct {
	ctx         context.Context
	store       *RepositoryStore
	quietHook   func(dir, name string, args ...string) (string, error)
	commandHook func(id, dir, name string, args ...string) error
	events      EventSink
	operations  *operationRegistry
	pendingMu   sync.Mutex
	pendingLogs []LogEvent
}

func NewApp() *App {
	app := &App{store: NewRepositoryStore(defaultData()), operations: newOperationRegistry()}
	app.events = runtimeEventSink{context: func() context.Context { return app.ctx }}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	data, err := loadData()
	if err != nil {
		a.queueStartupLog(err.Error())
		return
	}
	a.store.Replace(data)
}

func (a *App) shutdown(_ context.Context) { a.operations.cancelAll() }

func (a *App) GetData() StoreData {
	result := a.store.Snapshot()
	sort.SliceStable(result.Repositories, func(i, j int) bool {
		return result.Repositories[i].CreatedAt.After(result.Repositories[j].CreatedAt)
	})
	a.flushStartupLogs()
	return result
}

func (a *App) queueStartupLog(message string) {
	a.pendingMu.Lock()
	a.pendingLogs = append(a.pendingLogs, LogEvent{Level: "error", Message: message})
	a.pendingMu.Unlock()
}

func (a *App) flushStartupLogs() {
	a.pendingMu.Lock()
	logs := a.pendingLogs
	a.pendingLogs = nil
	a.pendingMu.Unlock()
	for _, event := range logs {
		a.emitLog(event.RepositoryID, event.Level, event.Message)
	}
}

func (a *App) Minimise() { runtime.WindowMinimise(a.ctx) }

func (a *App) ToggleMaximise() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

func (a *App) Close() { runtime.Quit(a.ctx) }
