package domain

// BiliVideoInfo 对应 videoInfo.json 或 .videoInfo 的结构
type BiliVideoInfo struct {
	Title     string `json:"title"`  // 视频标题
	Uname     string `json:"uname"`  // UP主
	Bvid      string `json:"bvid"`   // BV号
	Status    string `json:"status"` // "completed"
	CoverUrl  string `json:"coverUrl"`
	CoverPath string `json:"coverPath"`
}

// VideoTask 代表列表中的一行任务
type VideoTask struct {
	Dir        string        // 文件夹绝对路径
	FolderName string        // 文件夹名 (如 30964780940)
	Info       BiliVideoInfo // 解析出的元数据
	Status     string        // "等待", "处理中", "完成", "失败"
	ErrMessage string
}

// DisplayTitle 用于在 UI 上显示
func (v VideoTask) DisplayTitle() string {
	if v.Info.Title == "" {
		return v.FolderName // 降级显示
	}
	return v.Info.Title
}
