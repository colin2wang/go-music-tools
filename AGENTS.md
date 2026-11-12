# AGENTS.md — 项目代理指令

## 技术栈

Go 1.26 + Wails v2 桌面应用（`main.App` 绑定），前端 Vue 3 + TypeScript + Vite（`frontend/`）。

## 命令

```bash
# 前端依赖（仓库用 pnpm；wails.json 里写的 npm install 不生效/勿改依赖此假设）
pnpm --dir frontend install --frozen-lockfile

# 开发（同时启动 Vite 与 Go 窗口，前端热更新）
wails dev

# 测试（后端包不 import Wails runtime，可直接跑）
go test ./...
go vet ./...

# 前端构建含 vue-tsc --noEmit 类型检查
pnpm --dir frontend run build

# 一键构建（产物 build/bin/go-music-tools.exe；无 wails CLI 时自动回退 go build）
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

## 架构边界

- 后端业务包（`internal/*`）禁止 import Wails runtime；事件发射器由 `app.go` 注入。新增绑定服务放 `internal/bindings` 并在 `app.go` 注册。
- 前端通过 `window.go.main.App.*` 调用绑定、`window.runtime.EventsOn` 订阅事件（`log:batch`、`task:progress`、`task:item`、`task:done`、`config:changed`）。改动绑定方法签名须同步前端调用。
- 配置：新增字段在 `internal/config/schema.go` 加表单元数据即可，设置页由 `GetConfigSchema` 动态渲染，前端无需改动；保存走原子写。
- 本地运行产物（均为生成物，已 ignore）：`config.yaml`、`state.yaml`、`logs/`、`frontend/wailsjs/`（Wails 生成的 TS 绑定，改动后端绑定后重新生成，勿手改）。
- 偏好/缓存/任务历史存于 `internal/store`（go-cache，进程内）；其中上次下载偏好与限流时间戳持久化到 `state.yaml`，缓存与任务历史重启即清空。不使用数据库，勿重新引入 SQLite。

## 参考文档

- `docs/migration-notes.md`：Python→Go 迁移映射与 API 契约，涉及 API 行为差异时查阅。
- `README.md`：完整构建与配置说明。

## 维护规则

项目结构、构建/测试命令、架构边界、开发约定或本文件记录的其他事实发生变化时，必须在同一次改动中同步更新本文件：用当前事实替换过期表述，不逐次追加变更历史。本文件只保留当前规则、命令、边界与非显然注意事项；详细事故复盘、性能对比、实验过程和已完成迁移记录放到有链接的文档（如 `docs/`），仅在任务相关时读取。
