package bindings

// AppInfo 供顶部栏与底部状态条展示运行环境信息。
type AppInfo struct {
	Version      string `json:"version"`
	ConfigPath   string `json:"config_path"`
	LogDir       string `json:"log_dir"`
	DownloadDir  string `json:"download_dir"`
	Platform     string `json:"platform"`
	DatabasePath string `json:"database_path"`
}

// LastDownloadConfig 是下载工作台的「上次偏好」，存于 SQLite preferences 表。
type LastDownloadConfig struct {
	Kind       string `json:"kind"`
	ResourceID string `json:"resource_id"`
	Quality    string `json:"quality"`
	Indexes    []int  `json:"indexes"`
}

type ActionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type DownloadRequest struct {
	Kind       string `json:"kind"`
	ResourceID string `json:"resource_id"`
	Quality    string `json:"quality"`
	Indexes    []int  `json:"indexes"`
}

type OODownloadRequest struct {
	Keyword string `json:"keyword"`
	SongMid string `json:"song_mid"`
	Quality string `json:"quality"`
}

type LibraryFixRequest struct {
	Directory       string `json:"directory"`
	Kind            string `json:"kind"`
	OnlyMissing     bool   `json:"only_missing"`
	Force           bool   `json:"force"`
	Recursive       bool   `json:"recursive"`
	ContinueOnError bool   `json:"continue_on_error"`
}
