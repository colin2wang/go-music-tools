package main

import (
	"context"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"go-music-tools/internal/bindings"
	"go-music-tools/internal/logx"
)

// App 是 Wails 服务入口。嵌入 bindings.Service 后，其所有导出方法都会暴露给前端，
// 业务层无需 import Wails runtime。
type App struct {
	*bindings.Service

	svc *bindings.Service
	hub *logx.Hub
	// quitting 替代 v2 时期 a.ctx.Err() 的关闭保护：ServiceShutdown 无 ctx 可查。
	quitting atomic.Bool
}

func NewApp(svc *bindings.Service, hub *logx.Hub) *App {
	return &App{Service: svc, svc: svc, hub: hub}
}

// ServiceName 显式指定服务名，前端调用路径保持 "main.App.<Method>"，与 v2 语义一致。
func (a *App) ServiceName() string { return "App" }

func (a *App) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	app := application.Get()

	a.svc.Hooks.SetEmitter(func(event string, payload any) {
		// 应用关闭后不再向已销毁的 WebView 推送事件，避免 shutdown 阶段阻塞退出。
		if !a.quitting.Load() {
			app.Event.Emit(event, payload)
		}
	})
	a.svc.Hooks.SetDirectoryDialog(func(title string) (string, error) {
		return app.Dialog.OpenFile().SetTitle(title).
			CanChooseFiles(false).CanChooseDirectories(true).
			PromptForSingleSelection()
	})
	a.svc.Hooks.SetFileDialog(func(title string) (string, error) {
		return app.Dialog.OpenFile().SetTitle(title).
			CanChooseFiles(true).CanChooseDirectories(false).
			PromptForSingleSelection()
	})
	a.svc.Hooks.SetSaveDialog(func(title, defaultName string) (string, error) {
		return app.Dialog.SaveFile().SetMessage(title).
			SetFilename(defaultName).
			PromptForSingleSelection()
	})
	a.hub.Flush()

	app.Event.Emit("app:ready", map[string]any{
		"version":    "1.0.0",
		"configPath": a.svc.ConfigPath(),
		"logDir":     a.hub.LogDir(),
	})
	a.hub.Logger().Info("应用启动完成", "config", a.svc.ConfigPath(), "logs", a.hub.LogDir())
	return nil
}

func (a *App) ServiceShutdown() error {
	a.quitting.Store(true)
	a.svc.CancelTask()
	a.hub.Logger().Info("应用退出")
	a.hub.Flush()
	return nil
}
