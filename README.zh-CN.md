# Music Tools

[English](README.md) | 简体中文

![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white) ![Wails](https://img.shields.io/badge/Wails-v3-red?logo=wails&logoColor=white) ![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?logo=typescript&logoColor=white) ![Tailwind_CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)

音乐下载与整理桌面工作台。所有下载均通过 API 完成（无需第三方桌面客户端）。主界面常驻全局日志窗口与任务面板，配置以
YAML 保存并可在「设置」页可视化修改。

## 功能

### 下载

- 单曲 / 专辑 / 歌单下载：资源 ID 或分享链接、7 档音质（`standard` → `jymaster`）、指定曲目编号、已存在跳过、实时进度
- OO Music 搜索、歌曲详情、音质对比、单曲与批量下载
- 备用解析通道：NextMusic V2（可切换、带失败重试）
- 记住上次下载偏好（类型 / 音质 / 曲目编号），重启后自动恢复

### 音乐库整理

- 自动整理：专辑/歌单目录、`album_info.txt` / `playlist_info.txt`、封面、LRC 歌词、ID3 / Vorbis / MP4 标签
- 存量音乐库工具：歌词修复、封面重下、艺术家拆分、元数据补全——对所选目录的一级子目录批量执行

### 任务队列

- 两条互不阻塞的 FIFO 通道（`oomusic:*` 与 `download:*` / `library:*`）：通道内严格串行、通道间可并行；提交立即返回 `task_id`
- 任务面板：实时进度、逐条子项结果、移出排队任务、全部取消或按 ID 取消、仅重试失败子项；进程内注册表保留最近 50 个任务

### 日志、缓存与设置

- 全局日志窗口：实时日志、等级过滤、关键字搜索、清空、导出、打开日志目录；可运行时切换日志级别，不写入 `config.yml`
- 缓存页面：按区查看与清空缓存（OO 搜索/详情、偏好设置、下载历史），支持格式化 JSON 详情
- 可视化配置：分组动态表单由后端 Schema 渲染、字段校验、保存 / 重载 / 恢复默认、YAML 导入与导出、即时生效

## 环境要求

- Go 1.26+
- Node.js 18+ / pnpm 9.x（仓库已提供 `frontend/pnpm-lock.yaml`）
- Windows：WebView2 Runtime（一般系统已内置）
- Wails v3 CLI（可选，用于 `wails3 dev` / `wails3 task build`；缺失时构建脚本自动回退 `go build -tags production,desktop`）

安装 CLI：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
```

## 开发

```bash
pnpm --dir frontend install --frozen-lockfile
wails3 dev
```

`wails3 dev` 同时启动 Vite 开发服务器与 Go 窗口，前端热更新。

## 构建

在仓库根目录运行一键构建脚本（脚本注释与输出均为英文）：

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

脚本按顺序执行：预检 `pnpm` / `go` → `pnpm --dir frontend install --frozen-lockfile` →
`pnpm --dir frontend run build`（并校验 `frontend/dist/index.html`）→ 桌面二进制编译。检测到
`wails3` CLI 时执行 `wails3 task build`；否则回退
`go build -tags production,desktop -ldflags="-H windowsgui" -o build/bin/go-music-tools.exe .`。
两条路径产物均为 `build/bin/go-music-tools.exe`，任一步骤失败会以英文报错并不为零退出。

使用 Wails v3 CLI 手动构建（构建编排位于 `Taskfile.yml`）：

```bash
pnpm --dir frontend run build
wails3 task build
```

无 Wails CLI 时，先构建前端再用 Go 直接编译（内嵌 `frontend/dist`）：

```bash
pnpm --dir frontend run build
go build -tags production,desktop -ldflags="-H windowsgui" -o go-music-tools.exe .
```

## 配置

首次启动会按内置默认值生成 `config.yml`。字段分组：`api`、`download`、`filename`、`metadata`、
`network`、`logging`、`advanced`——每个字段均真实生效（原 Python 版存在无效选项，已全部重新设计）。
设置页由后端 `GetConfigSchema` 动态渲染表单，新增字段无需改动前端。

配置查找顺序：优先 `MUSIC_TOOLS_ROOT` 环境变量指定的目录，否则为当前工作目录。数据缓存保存在进程内存
（go-cache）；上次下载偏好与元数据限流时间戳持久化到同目录的 `state.yml`；日志写入 `logs/`。

## 项目结构

```
main.go / app.go          Wails 入口与生命周期（绑定 main.App）
build.ps1                 根目录一键构建脚本（pnpm + Wails / go build）
internal/config           YAML 配置、Schema 元数据、原子写
internal/logx             slog 分发、轮转文件、环形缓冲、事件推送
internal/store            go-cache 进程内偏好与缓存
internal/httpclient       UA / 重试 / 限流 / TLS 指纹
internal/api              网易云、NextMusic V2、OO Music 客户端
internal/music            元数据、命名、编号解析、下载、整理
internal/tag              MP3 / FLAC / MP4 标签读写
internal/library          存量音乐库修补流程
internal/oomusic          OO Music 业务服务
internal/task             任务队列管理（双通道）、进度、重试、事件
internal/bindings         Wails 绑定服务（不依赖 Wails runtime）
frontend/src              Vue 3 + TypeScript 界面（视图：下载、OO 音乐、音乐库
                          工具、缓存、设置；中英文 i18n）
docs/migration-notes.md   Python → Go 迁移映射与 API 契约
```

后端业务包不 import Wails runtime；事件发射器由 `app.go` 注入，因此 `go test ./...` 可独立运行。
前端通过 `@wailsio/runtime` 的 `Call.ByName("main.App.*")` 调用绑定，经 `Events.On` 订阅
`log:batch`、`task:progress`、`task:item`、`task:done`、`task:queued`、`config:changed` 事件；
任务面板由 `task:queued` 推送的注册表快照驱动，并通过 `GetTaskBoard` 全量刷新。

## 免责声明

仅供整理个人已授权的音乐使用。请遵守各音乐平台服务条款与著作权法。
