package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"go-music-tools/internal/bindings"
	"go-music-tools/internal/config"
	"go-music-tools/internal/logx"
	storepkg "go-music-tools/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
		os.Exit(1)
	}
}

func run() error {
	root := appRoot()
	migrateLegacyYAML(root)
	store := config.NewStore(filepath.Join(root, "config.yml"))
	if err := store.Load(); err != nil {
		return fmt.Errorf("加载配置: %w", err)
	}

	hub, err := logx.NewHub(store.Get().Logging, root)
	if err != nil {
		return fmt.Errorf("初始化日志: %w", err)
	}
	defer hub.Close()

	svc, err := bindings.NewService(bindings.Deps{
		Root:   root,
		Config: store,
		Logs:   hub,
		Cache:  storepkg.New(filepath.Join(root, "state.yml")),
		Logger: hub.Logger(),
	})
	if err != nil {
		return fmt.Errorf("初始化服务: %w", err)
	}
	defer svc.Close()

	app := NewApp(svc, hub)

	return wails.Run(&options.App{
		Title:     "Music Tools · 音乐工作台",
		Width:     1440,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 16, B: 32, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
}

func appRoot() string {
	if v := os.Getenv("MUSIC_TOOLS_ROOT"); v != "" {
		return v
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// migrateLegacyYAML 把旧的 config.yaml / state.yaml 重命名为 .yml，
// 保留已有配置与偏好；目标已存在时保留新文件，旧文件原样留在原地。
func migrateLegacyYAML(root string) {
	for _, pair := range [][2]string{
		{"config.yaml", "config.yml"},
		{"state.yaml", "state.yml"},
	} {
		oldPath := filepath.Join(root, pair[0])
		newPath := filepath.Join(root, pair[1])
		if _, err := os.Stat(oldPath); err != nil {
			continue
		}
		if _, err := os.Stat(newPath); err == nil {
			continue
		}
		_ = os.Rename(oldPath, newPath)
	}
}
