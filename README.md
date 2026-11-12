# Music Tools

English| [简体中文](README.zh-CN.md)

![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white) ![Wails](https://img.shields.io/badge/Wails-v2-red?logo=wails&logoColor=white) ![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?logo=typescript&logoColor=white) ![Tailwind_CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)

A desktop workbench for music downloading and library organization, fully rewritten from
`python_uv_cloud_music`. The main window keeps a global log panel always available, and the
configuration is stored in YAML and editable visually on the Settings page.

## Features

- NetEase single / album / playlist download: ID or share link, 7 quality levels, selected track
  indexes, skip existing files, live progress
- Music search (NetEase proxy `/Search`) plus OO Music search, details, quality comparison, and
  single / batch download
- Automatic organization: album / playlist directories, `album_info.txt` / `playlist_info.txt`,
  cover art, LRC lyrics, ID3 / Vorbis / MP4 tags
- Existing-library tools: lyrics repair, cover re-download, artist splitting, metadata completion
- Fallback resolution channel: NextMusic V2 (switchable, with retry on failure)
- Global log panel: live logs, level filter, keyword search, clear, export, open log directory
- Visual configuration: grouped dynamic forms, field validation, save / reload / reset, YAML
  import & export, effective immediately

## Requirements

- Go 1.26+
- Node.js 18+ / pnpm 9.x (`frontend/pnpm-lock.yaml` is provided in the repository)
- Windows: WebView2 Runtime (usually preinstalled)
- Wails CLI (optional, for `wails dev` / `wails build`; the build script falls back to plain
  `go build` when it is missing)

Install the CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
```

## Development

```bash
pnpm --dir frontend install --frozen-lockfile
wails dev
```

`wails dev` starts the Vite dev server and the Go window together, with frontend hot reload.

## Build

Run the one-click build script from the repository root (script comments and output are English):

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

The script runs in order: precheck `pnpm` / `go` → `pnpm --dir frontend install --frozen-lockfile`
→ `pnpm --dir frontend run build` (and verifies `frontend/dist/index.html`) → desktop binary
compilation. When the `wails` CLI is detected it runs `wails build`; otherwise it falls back to
`go build -ldflags="-H windowsgui" -o build/bin/go-music-tools.exe .`.
Both branches produce `build/bin/go-music-tools.exe`, and any failed step prints an English
error and exits non-zero.

Manual build with the Wails CLI:

```bash
pnpm --dir frontend run build
wails build
```

Without the Wails CLI, build the frontend first and compile with Go directly (embeds
`frontend/dist`):

```bash
pnpm --dir frontend run build
go build -ldflags="-H windowsgui" -o go-music-tools.exe .
```

## Configuration

On first launch `config.yaml` is generated from built-in defaults. Field groups: `api`,
`download`, `filename`, `metadata`, `network`, `logging`, `advanced` — every field is actually
effective (the original Python version had dead options; they have been redesigned). The Settings
page renders the form dynamically from the backend `GetConfigSchema`, so adding a field requires
no frontend change.

Config lookup order: directory specified by the `MUSIC_TOOLS_ROOT` environment variable, otherwise
the current working directory. Data caches live in process memory (go-cache); the last download
preferences and the metadata rate-limit timestamp are persisted to `state.yaml` in the same
directory; logs go to `logs/`.

## Project Layout

```
main.go / app.go          Wails entry and lifecycle (binds main.App)
build.ps1                 One-click root build script (pnpm + Wails / go build)
internal/config           YAML config, schema metadata, atomic writes
internal/logx             slog fan-out, rotating files, ring buffer, event push
internal/store            go-cache in-memory preferences and caches
internal/httpclient       UA / retry / rate limiting
internal/api              NetEase, NextMusic V2, OO Music clients
internal/music            Metadata, naming, index parsing, download, organizing
internal/tag              MP3 / FLAC / MP4 tag read/write
internal/library          Four existing-library repair flows
internal/oomusic          OO Music business service
internal/task             Task cancellation and progress
internal/bindings         Wails binding services (no Wails runtime dependency)
frontend/src              Vue 3 + TypeScript UI
docs/migration-notes.md   Python → Go migration mapping and API contracts
```

Backend business packages do not import the Wails runtime; the event emitter is injected by
`app.go`, so `go test ./...` runs standalone. The frontend calls bindings via
`window.go.main.App.*` and subscribes to `log:batch`, `task:progress`, `task:item`,
`task:done`, `config:changed` events via `window.runtime.EventsOn`; `task:item` pushes
track / directory results one by one for live progress tables.

## Disclaimer

For organizing personally licensed music only. Please respect the music platforms' terms of
service and copyright law.
