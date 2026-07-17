package main

import (
	"context"
)

// App is the Wails application context. Bound methods exposed to the
// frontend live on this struct.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup saves the runtime context so bound methods can call Wails runtime
// functions.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
