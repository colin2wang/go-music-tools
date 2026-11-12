package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-music-tools/internal/bindings"
	"go-music-tools/internal/db"
	"go-music-tools/internal/logx"
)

// App 是 Wails 绑定的入口。嵌入 bindings.Service 后，其所有导出方法都会注册为
// window.go.main.App.<Method>，业务层无需 import Wails runtime。
type App struct {
	*bindings.Service

	svc      *bindings.Service
	hub      *logx.Hub
	database *db.DB
	ctx      context.Context
}

func NewApp(svc *bindings.Service, hub *logx.Hub, database *db.DB) *App {
	return &App{Service: svc, svc: svc, hub: hub, database: database}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	a.svc.SetEmitter(func(event string, payload any) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, event, payload)
		}
	})
	a.svc.SetDirectoryDialog(func(title string) (string, error) {
		return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	})
	a.svc.SetFileDialog(func(title string) (string, error) {
		return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	})
	a.svc.SetSaveDialog(func(title, defaultName string) (string, error) {
		return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           title,
			DefaultFilename: defaultName,
		})
	})
	a.hub.Flush()

	runtime.EventsEmit(ctx, "app:ready", map[string]any{
		"version":    "1.0.0",
		"configPath": a.svc.ConfigPath(),
		"logDir":     a.hub.LogDir(),
	})
	a.hub.Logger().Info("应用启动完成", "config", a.svc.ConfigPath(), "logs", a.hub.LogDir())
}

func (a *App) shutdown(_ context.Context) {
	a.svc.CancelTask()
	a.hub.Logger().Info("应用退出")
	a.hub.Flush()
}
