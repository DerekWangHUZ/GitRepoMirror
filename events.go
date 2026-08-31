package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type EventSink interface {
	Log(LogEvent)
	Progress(ProgressEvent)
	Changed()
}

type runtimeEventSink struct{ context func() context.Context }

func (s runtimeEventSink) Log(event LogEvent) {
	if ctx := s.context(); ctx != nil {
		runtime.EventsEmit(ctx, "log", event)
	}
}
func (s runtimeEventSink) Progress(event ProgressEvent) {
	if ctx := s.context(); ctx != nil {
		runtime.EventsEmit(ctx, "progress", event)
	}
}
func (s runtimeEventSink) Changed() {
	if ctx := s.context(); ctx != nil {
		runtime.EventsEmit(ctx, "repositories-changed")
	}
}

func (a *App) emitLog(id, level, message string) {
	if a.events != nil {
		a.events.Log(LogEvent{RepositoryID: id, Level: level, Message: message, Time: time.Now().Format("15:04:05")})
	}
}
func (a *App) emitProgress(id string, step int, label, status string) {
	if a.events != nil {
		a.events.Progress(ProgressEvent{RepositoryID: id, Step: step, Label: label, Status: status})
	}
}
func (a *App) emitChanged() {
	if a.events != nil {
		a.events.Changed()
	}
}

func newID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes)
}
