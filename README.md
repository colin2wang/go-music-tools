# Music Tools

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white) ![Wails](https://img.shields.io/badge/Wails-v2-red?logo=wails&logoColor=white) ![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?logo=typescript&logoColor=white) ![Tailwind_CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)

音乐下载与整理桌面工作台，全量重写自
`python_uv_cloud_music`。主界面常驻全局日志窗口，配置以 YAML 保存并可在「设置」页可视化修改。

## 功能

- 网易云单曲 / 专辑 / 歌单下载：ID 或分享链接、7 档音质、指定曲目编号、已存在跳过、实时进度
- 音乐搜索（网易云代理 `/Search`）与 OO Music 搜索、详情、音质对比、单曲与批量下载
- 自动整理：专辑/歌单目录、`album_info.txt` / `playlist_info.txt`、封面、LRC 歌词、ID3 / Vorbis / MP4 标签
- 存量音乐库工具：歌词修复、封面重下、艺术家拆分、元数据补全
- 备用解析通道：NextMusic V2（可切换、带失败重试）
- 全局日志窗口：实时日志、等级过滤、关键字搜索、清空、导出、打开日志目录
- 可视化配置：分组动态表单、字段校验、保存 / 重载 / 恢复默认、YAML 导入与导出、即时生效

## 环境要求

- Go 1.26+（依赖 `modernc.org/sqlite` 要求）
- Node.js 18+ / pnpm 9.x（仓库已提供 `frontend/pnpm-lock.yaml`）
- Windows：WebView2 Runtime（一般系统已内置）
- Wails CLI（可选，用于 `wails dev` / `wails build`；缺失时构建脚本自动回退 `go build`）

安装 CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
```

## 开发运行

```bash
pnpm --dir frontend install --frozen-lockfile
wails dev
```

`wails dev` 会同时启动 Vite 开发服务器与 Go 窗口，前端热更新可用。

## 构建

推荐在根目录运行一键构建脚本（脚本的注释与全部输出均为英文）：

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

脚本依次执行：预检 `pnpm` / `go` → `pnpm --dir frontend install --frozen-lockfile`
→ `pnpm --dir frontend run build`（并校验 `frontend/dist/index.html`）→ 桌面程序编译。
检测到 `wails` CLI 时执行 `wails build`；否则自动回退
`go build -ldflags="-H windowsgui" -o build/bin/go-music-tools.exe .`。
两条分支的产物都位于 `build/bin/go-music-tools.exe`，任一阶段失败都会打印英文错误并以非零退出码终止。

手动分步构建（使用 Wails CLI）：

```bash
pnpm --dir frontend run build
wails build
```

没有 Wails CLI 时也可以先构建前端再用 Go 直接编译（嵌入 `frontend/dist`）：

```bash
pnpm --dir frontend run build
go build -ldflags="-H windowsgui" -o go-music-tools.exe .
```

## 配置

首次启动会按内置默认值生成 `config.yaml`。字段分组：`api`、`download`、`filename`、
`metadata`、`network`、`logging`、`advanced`，全部字段都会真实生效（原 Python 版存在
若干死配置，新版已重新设计）。界面「设置」页由后端 `GetConfigSchema` 动态渲染，
新增字段无需改前端。

配置文件查找顺序：环境变量 `MUSIC_TOOLS_ROOT` 指定目录，否则当前工作目录。
数据缓存为同目录 `app.db`（SQLite，WAL 模式），日志在 `logs/`。

## 目录结构

```
main.go / app.go          Wails 入口与生命周期（绑定 main.App）
build.ps1                 根目录一键构建脚本（pnpm + Wails / go build）
internal/config           YAML 配置、schema 元数据、原子写
internal/logx             slog 扇出、轮转文件、环形缓冲、事件推送
internal/db               SQLite 偏好与缓存
internal/httpclient       UA / 重试 / 限流
internal/api              网易云、NextMusic V2、OO Music 客户端
internal/music            元数据、命名、编号解析、下载、目录组织
internal/tag              MP3 / FLAC / MP4 标签读写
internal/library          四条存量库修补流程
internal/oomusic          OO Music 业务服务
internal/task             任务取消与进度
internal/bindings         Wails 绑定服务（不依赖 Wails runtime）
frontend/src              Vue 3 + TypeScript 界面
docs/migration-notes.md   Python → Go 迁移映射与 API 契约
```

后端业务包不 import Wails runtime，事件发射器由 `app.go` 注入，因此 `go test ./...`
可独立运行。前端通过 `window.go.main.App.*` 调用绑定方法，通过 `window.runtime.EventsOn`
订阅 `log:batch`、`task:progress`、`task:item`、`task:done`、`config:changed` 事件；
其中 `task:item` 逐条推送曲目 / 目录结果，供进度表实时刷新。

## 免责声明

仅用于整理个人已获授权的音乐，请遵守音乐平台服务条款与著作权法律。
