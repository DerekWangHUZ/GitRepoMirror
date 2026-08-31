package main

import (
	"context"
	"fmt"
	"sync"
)

// operationRegistry prevents overlapping writes to the same mirror and owns
// the cancellation functions for commands started by the application.
type operationRegistry struct {
	mu     sync.Mutex
	batch  bool
	active map[string]operation
}

type operation struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func newOperationRegistry() *operationRegistry {
	return &operationRegistry{active: make(map[string]operation)}
}

func (registry *operationRegistry) begin(base context.Context, id string, batchMember bool) (context.Context, func(), error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.batch && !batchMember {
		return nil, nil, fmt.Errorf("批量同步正在进行，请稍后重试")
	}
	if _, exists := registry.active[id]; exists {
		return nil, nil, fmt.Errorf("仓库正在同步，请稍后重试")
	}
	ctx, cancel := context.WithCancel(base)
	registry.active[id] = operation{ctx: ctx, cancel: cancel}
	return ctx, func() { registry.finish(id) }, nil
}

func (registry *operationRegistry) finish(id string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if current, exists := registry.active[id]; exists {
		current.cancel()
		delete(registry.active, id)
	}
}

func (registry *operationRegistry) beginBatch() (func(), error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.batch || len(registry.active) > 0 {
		return nil, fmt.Errorf("已有同步任务正在进行，请稍后重试")
	}
	registry.batch = true
	return func() {
		registry.mu.Lock()
		registry.batch = false
		registry.mu.Unlock()
	}, nil
}

func (registry *operationRegistry) context(id string, fallback context.Context) context.Context {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if current, exists := registry.active[id]; exists {
		return current.ctx
	}
	return fallback
}

func (registry *operationRegistry) cancelAll() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for id, current := range registry.active {
		current.cancel()
		delete(registry.active, id)
	}
	registry.batch = false
}
