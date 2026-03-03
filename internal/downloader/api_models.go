package downloader

// viewResponse 对应 /x/web-interface/view 接口返回。
type viewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Aid   int64  `json:"aid"`
		Bvid  string `json:"bvid"`
		Title string `json:"title"`
		Pic   string `json:"pic"`
		Cid   int64  `json:"cid"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
		Pages []struct {
			Cid      int64  `json:"cid"`
			Part     string `json:"part"`
			Page     int    `json:"page"`
			Duration int64  `json:"duration"`
		} `json:"pages"`
	} `json:"data"`
}

// playURLResponse 对应 playurl 接口返回。
type playURLResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AcceptQuality []int    `json:"accept_quality"`
		AcceptDesc    []string `json:"accept_description"`
		IsPreview     int      `json:"is_preview"`
		Timelength    int64    `json:"timelength"`
		Durl          []struct {
			URL       string   `json:"url"`
			Size      int64    `json:"size"`
			Order     int      `json:"order"`
			Length    int64    `json:"length"`
			BackupURL []string `json:"backup_url"`
		} `json:"durl"`
		Dash struct {
			Video []struct {
				ID        int    `json:"id"`
				BaseURL   string `json:"base_url"`
				Bandwidth int64  `json:"bandwidth"`
				Codecs    string `json:"codecs"`
			} `json:"video"`
			Audio []struct {
				ID        int    `json:"id"`
				BaseURL   string `json:"base_url"`
				Bandwidth int64  `json:"bandwidth"`
				Codecs    string `json:"codecs"`
			} `json:"audio"`
		} `json:"dash"`
	} `json:"data"`
}
