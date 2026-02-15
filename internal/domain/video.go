package domain

// BiliVideoInfo 对应 videoInfo.json 或 .videoInfo 的结构
type BiliVideoInfo struct {
	Title      string `json:"title"`      // 单集标题/视频标题
	GroupTitle string `json:"groupTitle"` // 合集标题/总标题
	Uname      string `json:"uname"`      // UP主
	Bvid       string `json:"bvid"`       // BV号
	Status     string `json:"status"`     // "completed"
	CoverUrl   string `json:"coverUrl"`   // 封面图 URL
	CoverPath  string `json:"coverPath"`
	Pubdate    int64  `json:"pubdate"`    // 发布时间戳
	CreateTime int64  `json:"createTime"` // 缓存创建时间
	P          int    `json:"p"`          // 当前分P
}

// VideoTask 代表列表中的一行任务
type VideoTask struct {
	Dir        string        // 文件夹绝对路径
	FolderName string        // 文件夹名 (如 30964780940)
	Info       BiliVideoInfo // 解析出的元数据
	Status     string        // "等待", "处理中", "完成", "失败"
	OutputPath string        // 合并后的文件路径
	ErrMessage string
}

// DisplayTitle 用于在 UI 上显示
func (v VideoTask) DisplayTitle() string {
	if v.Info.GroupTitle != "" {
		return v.Info.GroupTitle
	}
	if v.Info.Title != "" {
		return v.Info.Title
	}
	return v.FolderName // 降级显示
}
