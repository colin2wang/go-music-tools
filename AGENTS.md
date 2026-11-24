# AGENTS.md — 项目代理指令

## 技术栈

Go 1.26 + Wails v3 桌面应用（`application.New` + 服务注册），前端 Vue 3 + TypeScript + Vite（`frontend/`）。

## 命令

```bash
# 前端依赖（仓库用 pnpm，wails.json 与 Taskfile 均已配置 pnpm 命令）
pnpm --dir frontend install --frozen-lockfile

# 开发（同时启动 Vite 与 Go 窗口，前端热更新）
wails3 dev

# 测试（后端包不 import Wails runtime，可直接跑）
go test ./...
go vet ./...

# 前端构建含 vue-tsc --noEmit 类型检查
pnpm --dir frontend run build

# 生成 Go 服务的前端 TS 绑定（输出 frontend/bindings/）
wails3 generate bindings

# 一键构建（产物 build/bin/music-tools.exe；无 wails3 CLI 时自动回退 go build）
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

注意：不要主动编译、构建或运行测试来验证改动，用户会自己编译测试；除非用户明确要求。

## 架构边界

- 后端业务包（`internal/*`）禁止 import Wails runtime；事件发射器由 `app.go` 注入。新增绑定服务放 `internal/bindings` 并在 `app.go` 注册。
- 前端通过 `@wailsio/runtime` 的 `Call.ByName("main.App.<Method>")` 调用绑定（统一封装在 `frontend/src/api/bindings.ts`）、`Events.On` 订阅事件（`log:batch`、`task:progress`、`task:item`、`task:done`、`task:queued`、`config:changed`，封装在 `frontend/src/api/events.ts`；v3 回调收 `event.data` 而非 `data[0]`）。改动绑定方法签名须同步前端调用。
- 任务队列：`internal/task.Manager` 维护两条互不阻塞的 FIFO 通道，`oomusic:*` 走 OO Music 通道，其余（`download:*`、`library:*`）走音乐工作台通道；通道内严格串行、通道间可并行，提交立即返回 `task_id`（不阻塞、不报错）。事件 `task:progress` / `task:item` / `task:done` / `task:queued` 推送增量，其中 `task:queued` 载荷含通道状态与任务注册表快照。
- 任务状态：注册表以 `task_id` 追踪每个独立任务的 `queued / running / done / failed / cancelled / removed` 状态、起止时间、耗时、进度与子项，进程内最多保留最近 50 个（运行中/排队中不淘汰），重启清空；不写 `state.yml`、不使用数据库。
- 任务绑定：`GetTaskBoard`（全量快照）、`GetTaskQueue` / `GetTaskStatus`（兼容派生视图）、`CancelTask`（取消全部运行中）、`CancelTaskByID`、`RetryFailedItems`（只重试失败子项，需提交时用 `task.WithRetry` 注入捕获原始请求的工厂；`library:*` 不支持）、`RemoveQueuedTask`、`ClearTaskQueue`。提交统一走私有 `start(kind, total, fn, ...StartOption)`；不要提供接收 `task.TaskFunc` 的导出方法，否则 `wails3 generate bindings` 会报 function-types 警告。
- 任务 UI：前端 `stores/task.ts` 用 `entries` 注册表 + 派生 getter 驱动底部停靠窗口「任务」Tab 的分段列表（`GlobalLogPanel` → `TaskProgressCard` → `TaskCard`）与顶栏汇总（`AppTopBar` + 纯 SVG `TaskMiniDonut`，无图表库依赖）；各视图只按通道/任务类型取数，不再单独渲染全局进度。
- 配置：新增字段在 `internal/config/schema.go` 加表单元数据即可，设置页由 `GetConfigSchema` 动态渲染，前端无需改动；保存走原子写。
- 本地运行产物（均为生成物，已 ignore）：`config.yml`、`state.yml`、`logs/`、`frontend/bindings/`（`wails3 generate bindings` 生成的 TS 绑定，改动后端绑定后重新生成，勿手改；本项目前端走 `Call.ByName` 动态调用，绑定产物仅作渐进迁移备用）。
- 构建体系：`wails3 build`/`wails3 task build` 内部走 Task（根 `Taskfile.yml` + `build/Taskfile.yml` + `build/windows/Taskfile.yml`，应用元数据在 `build/config.yml`）。v3 绑定生成为纯静态分析（不执行本程序），无 v2 时代构建期产出 `config.yml`/`logs/` 的问题；手动 `go build` 回退须加 `-tags production,desktop`。
- 偏好/缓存/下载历史存于 `internal/store`（go-cache，进程内）；其中上次下载偏好与限流时间戳持久化到 `state.yml`，缓存与下载历史重启即清空。任务面板注册表是另一份进程内状态，由 `internal/task.Manager` 独立维护，两者都由 `onFinish` 驱动，勿混用。不使用数据库，勿重新引入 SQLite。

## 参考文档

- `docs/migration-notes.md`：Python→Go 迁移映射与 API 契约，涉及 API 行为差异时查阅。
- `README.md`：完整构建与配置说明。

## 维护规则

项目结构、构建/测试命令、架构边界、开发约定或本文件记录的其他事实发生变化时，必须在同一次改动中同步更新本文件：用当前事实替换过期表述，不逐次追加变更历史。本文件只保留当前规则、命令、边界与非显然注意事项；详细事故复盘、性能对比、实验过程和已完成迁移记录放到有链接的文档（如 `docs/`），仅在任务相关时读取。
