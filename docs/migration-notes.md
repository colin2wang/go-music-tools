# Python → Go 迁移说明

源项目：`C:\Workspaces\JetBrains\PyCharm\python_uv_cloud_music`（只读参考，未修改）
目标项目：`go-music-tools`（Go + Wails v2 + Vue 3 + TypeScript）

## 1. 模块映射表

| Python 模块 | Go 包 / 文件 | 说明 |
|---|---|---|
| `src/util/util_config.py` | `internal/config` | 结构体重设计，YAML 为唯一真实来源，附带 schema 元数据 |
| `src/util/util_logging.py` | `internal/logx` | `slog` 多 Handler 扇出 + 轮转文件 + 环形缓冲 + Wails 事件 |
| `src/util/util_database.py` | `internal/store` | 进程内 go-cache 存储（无数据库）；原版 `music_url` 只写内存，本版同为内存但统一由 store 管理 |
| `src/util/util_commons.py` | `internal/music/filename.go`、`indexes.go`、`internal/httpclient/retry.go` | 清洗、编号解析、限流 |
| `src/util/util_music_api.py` | `internal/api/netease.go` | `/Song_V1`、`/Album`、`/Playlist`；`/Search` 原版未接入，本版作为搜索页启用 |
| `src/tool/tool_next_music_v2.py` | `internal/api/nextmusic.go` | 明文 JSON + Chrome TLS 指纹降级 |
| `src/util/util_oo_music_api.py` | `internal/api/oomusic.go` | 搜索/详情/音质解析 |
| `src/scripts/oo_music_download.py` | `internal/oomusic/service.go` + `OOMusicView.vue` | 菜单交互改 GUI |
| `src/util/util_music_metadata.py` | `internal/music/metadata.go` | 限流、重试、音质校验、通道分流 |
| `src/util/util_music_downloader.py` | `internal/music/{downloader,organize,filename}.go` | 流式下载、目录组织、说明文件 |
| `src/util/util_audio_tag.py` | `internal/tag` | 统一 `TagHandler` 抽象 |
| `src/util/util_album.py` | `internal/library/albuminfo.go` | `album_info.txt` 解析与音频枚举 |
| `src/process/process_from_folders.py` | `internal/library/folders.go` | 批量调度 |
| `src/process/process_album_lyrics_fix.py` | `internal/library/fix_lyrics.go` | `CMUSIC_ID` 幂等 |
| `src/process/process_album_cover_redownload.py` | `internal/library/fix_cover.go` | 分辨率阈值 400×400 |
| `src/process/process_album_artist_fix.py` | `internal/library/fix_artists.go` | `/`、`&`、`、` 拆分 |
| `src/process/process_album_metadata_fix.py` | `internal/library/fix_metadata.go` | 仅修 title/artist/album 缺失 |
| `interactive_process.py` | `internal/bindings/download.go` + `DownloadView.vue` | 交互式菜单改 GUI |
| `src/process/process_cloud_music.py` | 不迁移 | 用户选择仅桌面 GUI |

## 2. API 字段契约（以 `docs/api/` 为准）

### `POST {base_url}/Song_V1`
form：`url`(歌曲 ID)、`level`、`type=json`。
响应 `data` 共 10 字段：`al_name`、`ar_name`、`id`(字符串)、`level`、`lyric`、`name`、`pic`、`size`(字符串，如 `"23.82MB"`)、`tlyric`、`url`。**没有 `duration`**。

### `GET {base_url}/Album?id=`
`data.album`：`artist`、`coverImgUrl`、`description`、`id`、`name`、`publishTime`(毫秒)、`songs[]`；
`songs[]`：`album`、`artists`(字符串)、`id`、`name`、`picUrl`；`data.status` 为整数 `200`。

### `GET {base_url}/Playlist?id=`
`data.playlist`：`coverImgUrl`、`creator`、`description`、`id`、`name`、`trackCount`、`tracks[]`；
`tracks[]` 同 `songs[]`；`data.status` 是字符串 `"success"`（与 Album 不同）。
`creator` 文档样例为字符串，代码兼容 `{nickname: ...}` 对象，Go 侧用 `json.RawMessage` 双解码。

### NextMusic V2（`https://nextmusic.toubiec.cn`）
三接口 `/api/getSongInfo`、`/api/getSongLyric`、`/api/getSongUrl`，payload `{id, timestamp(ms), ip, level?}`，
顶层 `code == 200 && data != nil`。拼装兼容 `data` 时：`size` 恒为空、`lyric` 可能缺 key、`duration` 来自 info。

### OO Music（`https://tang.api.s01s.cn/music_open_api.php`）
搜索 `?msg={keyword}&type=json` → `[{song_title, singer_name, song_mid, pay}]`；
详情 `?msg={singer_name}&type=json&mid={song_mid}`。
原始 key：标题优先 `song_title` 否则 `song_name`；专辑优先 `album_name` 否则 `album_title`；
封面 `album_pic`；LRC `song_lyric`；纯文本 `lyric`；`qrc`；`singer_pic`；`song_h5_url`；`song_play_time`/`duration`；`vip`/`language`/`genre`/`public_time`。
带后缀字段（后缀 ∈ `sq|hq|standard|fq`）各 5 类：`song_play_url_{s}`、`song_filename_{s}`、`kbps_{s}`、`song_size_{s}`、`song_size_{s}_str`。
音质映射 `SQ→sq`、`HQ→hq`、`standard→standard`、`low→fq`，回退顺序 `SQ, HQ, standard, low`。
扩展名由 `song_filename_{s}` 前缀推断：`F000→.flac`，`C600/C400/C200→.m4a`。

## 3. 修补流程关键判据

| 流程 | 判据 |
|---|---|
| 歌词修复 | 专辑内**所有**音频都含 `CMUSIC_ID` 则整体跳过；单文件按该 tag 存在性跳过；MP3 歌词按 5000 字符分块写多个 `COMM`；结束把 `.lrc` 移入 `.delete/` |
| 封面重下 | `width >= 400 && height >= 400` 跳过；否则删除后重下，封面 URL 去掉 `?` 后的参数 |
| 艺术家拆分 | 仅当 artist 含 `/` 触发；归一化 `/`、`&`、`、` → `, `；用 `index_format` 重建 `{index} - {artists} - {title}` |
| 元数据补全 | `needsFix = !(hasTitle && hasArtist && hasAlbum)`，只看 3 字段；年份取 `publish_date[:4]`；替换封面先删旧 `APIC` / `clear_pictures()` |

## 4. 明确不迁移

- `src/tool/tool_next_music_v1.py`（AES-256-GCM，零业务引用）与 `docs/tasks/20260513/` 实验脚本；加密文档保留参考。
- `src/tool/tool_next_music_album.py`（`/api/getAlbum`，未接入）。
- 死配置：`filename.pattern`、`network.api_delay_min`、`logging.*`（原版从未读取）。
- `src/model/basic/config.py`、`src/model/extra/song_metadata.py`（未实例化）。
- 原版 `tqdm`、`msvcrt` 逐字符 ESC 监听（改为 GUI 取消按钮 + context）。

## 5. 主动修正的源项目缺陷

1. `music_url` 缓存与原版同为进程内存储，Go 版统一收敛到 `internal/store`（go-cache）。
2. OO Music 选低音质时扩展名/日志音质与播放地址不匹配，Go 版按实际选中的音质推断扩展名。
3. 日志原来每启动一个文件且无轮转，Go 版用 `lumberjack` 按大小/天数轮转。
4. `config.yml` 中的死配置在新 `config.yml` 中重新设计，全部字段都真实生效。
5. 音频下载改为流式写 `.part` 后原子 rename，不留半文件。

## 6. 配置差异

新版配置文件为 `config.yml`（历史版本曾用 `config.yaml`/`config.yml` 命名，现统一 `.yml` 扩展名）；不使用数据库，偏好/缓存/任务历史存于进程内（`internal/store`，go-cache），其中上次下载偏好与限流时间戳持久化到 `state.yml`。
