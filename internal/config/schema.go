package config

import "reflect"

// Field 与 Group 是设置页动态表单的元数据。前端不需要硬编码任何字段定义。
type Field struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Value       any      `json:"value"`
	Default     any      `json:"default"`
	Description string   `json:"description"`
	Options     []string `json:"options,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Required    bool     `json:"required"`
}

type Group struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
}

type fieldDef struct {
	key, label, typ, desc string
	options               []string
	min, max              *float64
	required              bool
}

type groupDef struct {
	key, label, desc string
	fields           []fieldDef
}

func num(v float64) *float64 { return &v }

func schemaDefs() []groupDef {
	return []groupDef{
		{
			key: "api", label: "API 接口", desc: "网易云代理服务的地址与安全选项",
			fields: []fieldDef{
				{"base_url", "API 根地址", "string", "代理服务根地址，用于 /Song_V1、/Album、/Playlist", nil, nil, nil, true},
				{"insecure_skip_verify", "跳过 TLS 校验", "bool", "第三方代理证书常有问题，仅建议对该代理开启", nil, nil, nil, false},
			},
		},
		{
			key: "download", label: "下载", desc: "下载目录、默认音质与文件组织方式",
			fields: []fieldDef{
				{"default_dir", "下载目录", "path", "相对路径以应用工作目录为基准", nil, nil, nil, true},
				{"default_quality", "默认音质", "enum", "请求失败时会按可用档位自动回退", QualityLevels, nil, nil, true},
				{"add_index_to_filename", "文件名加序号", "bool", "专辑与歌单下载时在文件名前加曲目序号", nil, nil, nil, false},
				{"index_format", "序号格式", "string", "数字宽度写法，03d 生成 001、002", nil, nil, nil, true},
				{"sub_dir", "专辑子目录", "string", "专辑/歌单目录下的子目录名，留空表示直接存放", nil, nil, nil, false},
				{"skip_existing", "跳过已存在文件", "bool", "同名音频已存在时不重复下载", nil, nil, nil, false},
			},
		},
		{
			key: "filename", label: "文件名", desc: "命名模板与 Windows 文件名安全策略",
			fields: []fieldDef{
				{"pattern", "命名模板", "string", "可用变量 {index} {artist} {title} {album}", nil, nil, nil, true},
				{"illegal_char_replacement", "非法字符替换", "string", "替换 < > : \" / \\ | ? * 等 Windows 非法字符", nil, nil, nil, false},
				{"artist_delimiter_replacement", "歌手分隔符", "string", "多位歌手之间的连接符", nil, nil, nil, false},
				{"max_length", "最大长度", "int", "超过时先减少歌手数量，再截断歌手与歌名", nil, num(20), num(240), true},
			},
		},
		{
			key: "metadata", label: "元数据", desc: "标签、封面与歌词写入策略",
			fields: []fieldDef{
				{"write_metadata", "写入标签", "bool", "写入标题、艺术家、专辑、曲目号与来源 ID", nil, nil, nil, false},
				{"write_cover", "写入封面", "bool", "下载封面并把图片嵌入音频文件", nil, nil, nil, false},
				{"write_lyrics", "写入歌词标签", "bool", "把 LRC 歌词写入音频标签", nil, nil, nil, false},
				{"save_lrc_file", "另存 LRC 文件", "bool", "在音频旁保存同名 .lrc 文件", nil, nil, nil, false},
				{"max_lyric_length", "M4A 歌词截断", "int", "MP4 ©lyr 标签最大字符数，0 表示不截断", nil, num(0), num(100000), false},
				{"cover_size", "封面边长", "int", "歌曲封面请求参数，例如 320 表示 320x320", nil, num(64), num(2000), false},
				{"use_next_music_tool", "启用备用解析", "bool", "切换到 NextMusic V2 通道获取歌曲信息与地址", nil, nil, nil, false},
			},
		},
		{
			key: "network", label: "网络", desc: "超时、重试与第三方接口限流",
			fields: []fieldDef{
				{"timeout", "请求超时（秒）", "float", "单次 HTTP 请求超时", nil, num(1), num(600), true},
				{"max_retries", "失败重试次数", "int", "额外重试次数，不含首次请求", nil, num(0), num(10), false},
				{"random_delay_max", "随机延迟上限（秒）", "float", "请求之间的随机等待，降低触发风控概率", nil, num(0), num(60), false},
				{"api_delay_min", "最小请求延迟（秒）", "float", "每次 API 调用前的固定最小等待", nil, num(0), num(60), false},
				{"get_metadata_interval", "元数据最小间隔（秒）", "float", "两次元数据请求间隔，时间戳持久化到数据库", nil, num(0), num(600), false},
				{"retry_backoff_min", "429 退避下限（秒）", "float", "遇到限流时的最短等待", nil, num(0), num(120), false},
				{"retry_backoff_max", "429 退避上限（秒）", "float", "遇到限流时的最长等待", nil, num(0), num(120), false},
			},
		},
		{
			key: "logging", label: "日志", desc: "全局日志窗口与日志文件的行为",
			fields: []fieldDef{
				{"level", "日志级别", "enum", "低于该级别的日志不会记录", LogLevels, nil, nil, true},
				{"console_output", "输出到控制台", "bool", "同时把日志打印到标准输出", nil, nil, nil, false},
				{"file_output", "写入文件", "bool", "按大小与天数轮转保存日志文件", nil, nil, nil, false},
				{"log_dir", "日志目录", "path", "相对路径以应用工作目录为基准", nil, nil, nil, true},
				{"max_file_size_mb", "单文件上限（MB）", "int", "超过后自动轮转", nil, num(1), num(500), false},
				{"max_backups", "保留备份数", "int", "轮转后保留的旧日志文件数量", nil, num(0), num(100), false},
				{"max_age_days", "保留天数", "int", "超过天数的旧日志自动清理", nil, num(0), num(365), false},
				{"buffer_size", "历史回放条数", "int", "前端挂载后可回放的最近日志条数", nil, num(100), num(20000), false},
				{"event_batch_size", "事件批量条数", "int", "达到该条数立即推送给日志窗口", nil, num(1), num(1000), false},
				{"event_flush_ms", "事件刷新间隔（毫秒）", "int", "未达批量条数时的定时推送间隔", nil, num(10), num(5000), false},
				{"show_source", "显示来源", "bool", "日志中显示来源文件与行号", nil, nil, nil, false},
			},
		},
		{
			key: "advanced", label: "高级", desc: "备用通道、OO Music 与存量库工具参数",
			fields: []fieldDef{
				{"next_music_base_url", "NextMusic 地址", "string", "备用解析服务根地址", nil, nil, nil, false},
				{"next_music_ip", "NextMusic 客户端 IP", "string", "备用接口 payload 中携带的 ip 字段", nil, nil, nil, false},
				{"oo_music_base_url", "OO Music 地址", "string", "OO Music 代理接口完整 URL", nil, nil, nil, false},
				{"oo_music_download_dir", "OO Music 下载目录", "path", "OO Music 歌曲存储位置", nil, nil, nil, false},
				{"oo_music_default_quality", "OO Music 默认音质", "enum", "不可用时按 SQ → HQ → standard → low 回退", OOQualityLevels, nil, nil, false},
				{"cover_min_resolution", "封面分辨率阈值", "int", "存量库封面边长低于此值时重新下载", nil, num(0), num(4000), false},
				{"max_concurrent_tasks", "主任务并发数", "int", "当前实现为串行执行，保留配置位", nil, num(1), num(1), false},
			},
		},
	}
}

// Schema 从 Config 结构体的 yaml 标签反射取值，保证新增字段不会与前端定义漂移。
func Schema(c Config) []Group {
	def := Default()
	groups := make([]Group, 0, 7)
	for _, g := range schemaDefs() {
		fields := make([]Field, 0, len(g.fields))
		for _, f := range g.fields {
			value, _ := lookup(c, g.key, f.key)
			fallback, _ := lookup(def, g.key, f.key)
			fields = append(fields, Field{
				Key:         f.key,
				Label:       f.label,
				Type:        f.typ,
				Value:       value,
				Default:     fallback,
				Description: f.desc,
				Options:     f.options,
				Min:         f.min,
				Max:         f.max,
				Required:    f.required,
			})
		}
		groups = append(groups, Group{
			Key: g.key, Label: g.label, Description: g.desc, Fields: fields,
		})
	}
	return groups
}

func lookup(c Config, group, key string) (any, bool) {
	rv := reflect.ValueOf(c)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).Tag.Get("yaml") != group {
			continue
		}
		gv := rv.Field(i)
		gt := gv.Type()
		for j := 0; j < gt.NumField(); j++ {
			if gt.Field(j).Tag.Get("yaml") == key {
				return gv.Field(j).Interface(), true
			}
		}
	}
	return nil, false
}
