package config

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

// Config 是配置的唯一真实来源：yaml 标签决定文件结构，json 标签决定绑定层 DTO，
// schema.go 依据同一结构体生成前端表单元数据。
type Config struct {
	API      APIConfig      `yaml:"api" json:"api"`
	Download DownloadConfig `yaml:"download" json:"download"`
	Filename FilenameConfig `yaml:"filename" json:"filename"`
	Metadata MetadataConfig `yaml:"metadata" json:"metadata"`
	Network  NetworkConfig  `yaml:"network" json:"network"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging"`
	Advanced AdvancedConfig `yaml:"advanced" json:"advanced"`
}

type APIConfig struct {
	BaseURL            string `yaml:"base_url" json:"base_url"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	SearchLimit        int    `yaml:"search_limit" json:"search_limit"`
}

type DownloadConfig struct {
	DefaultDir         string `yaml:"default_dir" json:"default_dir"`
	DefaultQuality     string `yaml:"default_quality" json:"default_quality"`
	AddIndexToFilename bool   `yaml:"add_index_to_filename" json:"add_index_to_filename"`
	IndexFormat        string `yaml:"index_format" json:"index_format"`
	SubDir             string `yaml:"sub_dir" json:"sub_dir"`
	SkipExisting       bool   `yaml:"skip_existing" json:"skip_existing"`
}

type FilenameConfig struct {
	Pattern                    string `yaml:"pattern" json:"pattern"`
	IllegalCharReplacement     string `yaml:"illegal_char_replacement" json:"illegal_char_replacement"`
	ArtistDelimiterReplacement string `yaml:"artist_delimiter_replacement" json:"artist_delimiter_replacement"`
	MaxLength                  int    `yaml:"max_length" json:"max_length"`
}

type MetadataConfig struct {
	WriteMetadata    bool `yaml:"write_metadata" json:"write_metadata"`
	WriteCover       bool `yaml:"write_cover" json:"write_cover"`
	WriteLyrics      bool `yaml:"write_lyrics" json:"write_lyrics"`
	SaveLRCFile      bool `yaml:"save_lrc_file" json:"save_lrc_file"`
	MaxLyricLength   int  `yaml:"max_lyric_length" json:"max_lyric_length"`
	CoverSize        int  `yaml:"cover_size" json:"cover_size"`
	UseNextMusicTool bool `yaml:"use_next_music_tool" json:"use_next_music_tool"`
}

type NetworkConfig struct {
	Timeout             float64 `yaml:"timeout" json:"timeout"`
	MaxRetries          int     `yaml:"max_retries" json:"max_retries"`
	RandomDelayMax      float64 `yaml:"random_delay_max" json:"random_delay_max"`
	APIDelayMin         float64 `yaml:"api_delay_min" json:"api_delay_min"`
	GetMetadataInterval float64 `yaml:"get_metadata_interval" json:"get_metadata_interval"`
	RetryBackoffMin     float64 `yaml:"retry_backoff_min" json:"retry_backoff_min"`
	RetryBackoffMax     float64 `yaml:"retry_backoff_max" json:"retry_backoff_max"`
}

type LoggingConfig struct {
	Level          string `yaml:"level" json:"level"`
	ConsoleOutput  bool   `yaml:"console_output" json:"console_output"`
	FileOutput     bool   `yaml:"file_output" json:"file_output"`
	LogDir         string `yaml:"log_dir" json:"log_dir"`
	MaxFileSizeMB  int    `yaml:"max_file_size_mb" json:"max_file_size_mb"`
	MaxBackups     int    `yaml:"max_backups" json:"max_backups"`
	MaxAgeDays     int    `yaml:"max_age_days" json:"max_age_days"`
	BufferSize     int    `yaml:"buffer_size" json:"buffer_size"`
	EventBatchSize int    `yaml:"event_batch_size" json:"event_batch_size"`
	EventFlushMS   int    `yaml:"event_flush_ms" json:"event_flush_ms"`
	ShowSource     bool   `yaml:"show_source" json:"show_source"`
}

type AdvancedConfig struct {
	NextMusicBaseURL      string `yaml:"next_music_base_url" json:"next_music_base_url"`
	NextMusicIP           string `yaml:"next_music_ip" json:"next_music_ip"`
	OOMusicBaseURL        string `yaml:"oo_music_base_url" json:"oo_music_base_url"`
	OOMusicDownloadDir    string `yaml:"oo_music_download_dir" json:"oo_music_download_dir"`
	OOMusicDefaultQuality string `yaml:"oo_music_default_quality" json:"oo_music_default_quality"`
	CoverMinResolution    int    `yaml:"cover_min_resolution" json:"cover_min_resolution"`
	MaxConcurrentTasks    int    `yaml:"max_concurrent_tasks" json:"max_concurrent_tasks"`
}

var QualityLevels = []string{"standard", "exhigh", "lossless", "hires", "sky", "jyeffect", "jymaster"}
var OOQualityLevels = []string{"SQ", "HQ", "standard", "low"}
var LogLevels = []string{"debug", "info", "warn", "error"}

func Default() Config {
	return Config{
		API: APIConfig{
			BaseURL: "https://dm.jfjt.cc/", InsecureSkipVerify: true, SearchLimit: 10,
		},
		Download: DownloadConfig{
			DefaultDir: "downloads", DefaultQuality: "lossless",
			AddIndexToFilename: true, IndexFormat: "03d", SubDir: "TBD", SkipExisting: true,
		},
		Filename: FilenameConfig{
			Pattern: "{index} - {artist} - {title}", IllegalCharReplacement: "_",
			ArtistDelimiterReplacement: ", ", MaxLength: 120,
		},
		Metadata: MetadataConfig{
			WriteMetadata: true, WriteCover: true, WriteLyrics: true, SaveLRCFile: true,
			MaxLyricLength: 1000, CoverSize: 320, UseNextMusicTool: false,
		},
		Network: NetworkConfig{
			Timeout: 30, MaxRetries: 1, RandomDelayMax: 2.5, APIDelayMin: 1.0,
			GetMetadataInterval: 20, RetryBackoffMin: 10, RetryBackoffMax: 15,
		},
		Logging: LoggingConfig{
			Level: "info", ConsoleOutput: true, FileOutput: true, LogDir: "logs",
			MaxFileSizeMB: 20, MaxBackups: 7, MaxAgeDays: 14, BufferSize: 2000,
			EventBatchSize: 100, EventFlushMS: 50, ShowSource: true,
		},
		Advanced: AdvancedConfig{
			NextMusicBaseURL: "https://nextmusic.toubiec.cn", NextMusicIP: "39.172.19.238",
			OOMusicBaseURL:     "https://tang.api.s01s.cn/music_open_api.php",
			OOMusicDownloadDir: "downloads/[OOMusic]", OOMusicDefaultQuality: "SQ",
			CoverMinResolution: 400, MaxConcurrentTasks: 1,
		},
	}
}

func (c Config) SlogLevel() slog.Level { return c.Logging.SlogLevel() }

func (l LoggingConfig) SlogLevel() slog.Level {
	switch strings.ToLower(l.Level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

var indexFormatRe = regexp.MustCompile(`^0?\d+d$`)
var templateVarRe = regexp.MustCompile(`\{(\w+)\}`)

func in(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// Validate 返回字段路径到错误说明的映射，供绑定层做字段级校验。
func (c Config) Validate() map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(c.API.BaseURL) == "" {
		errs["api.base_url"] = "API 根地址不能为空"
	} else if u, err := url.Parse(c.API.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs["api.base_url"] = "必须是包含协议与主机的完整 URL"
	}
	if c.API.SearchLimit < 1 || c.API.SearchLimit > 100 {
		errs["api.search_limit"] = "取值范围 1 - 100"
	}
	if strings.TrimSpace(c.Download.DefaultDir) == "" {
		errs["download.default_dir"] = "下载目录不能为空"
	}
	if !in(QualityLevels, c.Download.DefaultQuality) {
		errs["download.default_quality"] = "不支持的音质档位"
	}
	if !indexFormatRe.MatchString(c.Download.IndexFormat) {
		errs["download.index_format"] = "格式为数字加 d，例如 03d"
	}
	if strings.TrimSpace(c.Filename.Pattern) == "" {
		errs["filename.pattern"] = "命名模板不能为空"
	} else {
		for _, m := range templateVarRe.FindAllStringSubmatch(c.Filename.Pattern, -1) {
			switch m[1] {
			case "index", "artist", "title", "album":
			default:
				errs["filename.pattern"] = "未知变量 {" + m[1] + "}，可用 index/artist/title/album"
			}
		}
	}
	if c.Filename.MaxLength < 20 || c.Filename.MaxLength > 240 {
		errs["filename.max_length"] = "取值范围 20 - 240"
	}
	if c.Metadata.MaxLyricLength < 0 {
		errs["metadata.max_lyric_length"] = "不能为负数"
	}
	if c.Metadata.CoverSize < 64 || c.Metadata.CoverSize > 2000 {
		errs["metadata.cover_size"] = "取值范围 64 - 2000"
	}
	if c.Network.Timeout <= 0 {
		errs["network.timeout"] = "必须大于 0"
	}
	if c.Network.MaxRetries < 0 || c.Network.MaxRetries > 10 {
		errs["network.max_retries"] = "取值范围 0 - 10"
	}
	if c.Network.RandomDelayMax < 0 || c.Network.APIDelayMin < 0 || c.Network.GetMetadataInterval < 0 {
		errs["network.random_delay_max"] = "延迟与间隔不能为负数"
	}
	if c.Network.RetryBackoffMin < 0 || c.Network.RetryBackoffMax < c.Network.RetryBackoffMin {
		errs["network.retry_backoff_max"] = "退避上限不能小于下限"
	}
	if !in(LogLevels, strings.ToLower(c.Logging.Level)) {
		errs["logging.level"] = "取值 debug / info / warn / error"
	}
	if strings.TrimSpace(c.Logging.LogDir) == "" {
		errs["logging.log_dir"] = "日志目录不能为空"
	}
	if c.Logging.MaxFileSizeMB < 1 || c.Logging.MaxBackups < 0 || c.Logging.MaxAgeDays < 0 {
		errs["logging.max_file_size_mb"] = "轮转参数不合法"
	}
	if c.Logging.BufferSize < 100 || c.Logging.EventBatchSize < 1 || c.Logging.EventFlushMS < 10 {
		errs["logging.buffer_size"] = "缓冲至少 100 条，批量至少 1 条，刷新间隔至少 10ms"
	}
	if !in(OOQualityLevels, c.Advanced.OOMusicDefaultQuality) {
		errs["advanced.oo_music_default_quality"] = "取值 SQ / HQ / standard / low"
	}
	if c.Advanced.CoverMinResolution < 0 {
		errs["advanced.cover_min_resolution"] = "不能为负数"
	}
	if c.Advanced.MaxConcurrentTasks < 1 {
		errs["advanced.max_concurrent_tasks"] = "至少为 1"
	}
	return errs
}
