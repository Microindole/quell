package downloader

// PageInfo 单个分P信息
type PageInfo struct {
	Cid      int64  `json:"cid"`
	Part     string `json:"part"`
	Page     int    `json:"page"`
	Duration int64  `json:"duration"` // 单位：秒
}

// VideoInfo 视频基本信息
type VideoInfo struct {
	Title string     // 标题
	Bvid  string     // BV号
	Aid   int64      // AV号
	Cid   int64      // 默认 CID（第一P）
	Pic   string     // 封面 URL
	Owner string     // UP主名称
	Pages []PageInfo // 分P列表
}

// DashStream 单个音视频流信息
type DashStream struct {
	ID        int    // 清晰度/音质 ID
	BaseURL   string // 下载地址
	Bandwidth int64  // 码率
	Codecs    string // 编码格式
}

// PlayURLSelection 是一次播放地址选择结果，包含流模式与权限信息。
type PlayURLSelection struct {
	Video        *DashStream
	Audio        *DashStream
	Mode         string // dash / durl
	IsPreview    bool
	TrialSeconds int64
	Debug        string
}

// StreamMeta 提供给前端展示用的流元信息
type StreamMeta struct {
	QualityID    int    `json:"quality_id"`
	QualityLabel string `json:"quality_label"`
	Codec        string `json:"codec"`
	CodecLabel   string `json:"codec_label"`
	Bandwidth    int64  `json:"bandwidth"`
}

// DownloadPreference 下载偏好（画质与编码）
type DownloadPreference struct {
	QualityID int    `json:"quality_id"`
	Codec     string `json:"codec"`
}

// ProgressInfo 包含详细的下载进度信息
type ProgressInfo struct {
	Downloaded int64
	Total      int64
	Percentage float64
	Speed      float64 // 字节/秒
}

// ChunkState 记录单个分段的下载进度
type ChunkState struct {
	Start      int64 `json:"start"`
	End        int64 `json:"end"`
	Downloaded int64 `json:"downloaded"`
}

// DownloadState 记录整个文件的下载状态
type DownloadState struct {
	TotalSize int64        `json:"total_size"`
	Chunks    []ChunkState `json:"chunks"`
}
