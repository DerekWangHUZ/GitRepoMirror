package main

import (
	"context"
	"sort"

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
}

func NewApp() *App {
	app := &App{store: NewRepositoryStore(defaultData())}
	app.events = runtimeEventSink{context: func() context.Context { return app.ctx }}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	data, err := loadData()
	if err != nil {
		a.emitLog("", "error", err.Error())
		return
	}
	a.store.Replace(data)
}

func (a *App) shutdown(_ context.Context) {}

func (a *App) GetData() StoreData {
	result := a.store.Snapshot()
	sort.SliceStable(result.Repositories, func(i, j int) bool {
		return result.Repositories[i].CreatedAt.After(result.Repositories[j].CreatedAt)
	})
	return result
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
