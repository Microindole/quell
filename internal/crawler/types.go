package crawler

// BiliVideoMeta 是搜索返回的单个视频信息。
type BiliVideoMeta struct {
	Title    string `json:"title"`
	Bvid     string `json:"bvid"`
	Aid      int    `json:"aid"`
	Pic      string `json:"pic"`
	Subtitle string `json:"subtitle"`
	Length   string `json:"length"`
	Created  int64  `json:"created"`
	Play     string `json:"play"`
	Danmaku  string `json:"danmaku"`
}

// BiliDynamicItem 是动态流统一展示结构。
type BiliDynamicItem struct {
	IDStr      string            `json:"id_str"`
	Type       string            `json:"type"`
	UserName   string            `json:"user_name"`
	UserFace   string            `json:"user_face"`
	PubTime    string            `json:"pub_time"`
	PubTs      int64             `json:"pub_ts"`
	Text       string            `json:"text"`
	RichNodes  []DynamicRichNode `json:"rich_nodes"`
	Bvid       string            `json:"bvid"`
	VideoTitle string            `json:"video_title"`
	VideoCover string            `json:"video_cover"`
	ImageURLs  []string          `json:"image_urls"`
}

type DynamicRichNode struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	JumpURL  string `json:"jump_url"`
	EmojiURL string `json:"emoji_url"`
}

type DynamicListResult struct {
	Items   []BiliDynamicItem `json:"items"`
	Offset  string            `json:"offset"`
	HasMore bool              `json:"has_more"`
}

type arcSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List struct {
			Vlist []struct {
				Title       string      `json:"title"`
				Bvid        string      `json:"bvid"`
				Aid         int         `json:"aid"`
				Pic         string      `json:"pic"`
				Subtitle    string      `json:"subtitle"`
				Length      string      `json:"length"`
				Created     int64       `json:"created"`
				Play        interface{} `json:"play"`
				VideoReview interface{} `json:"video_review"`
			} `json:"vlist"`
		} `json:"list"`
		Page struct {
			Xcount int `json:"count"`
			Pn     int `json:"pn"`
			Ps     int `json:"ps"`
		} `json:"page"`
	} `json:"data"`
}

// BiliUserMeta 是搜索到的用户信息。
type BiliUserMeta struct {
	Mid    int    `json:"mid"`
	Uname  string `json:"uname"`
	Usign  string `json:"usign"`
	Fans   int    `json:"fans"`
	Verify bool   `json:"verify"`
	Upic   string `json:"upic"`
}

type searchUserResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Result []struct {
			Mid   int    `json:"mid"`
			Uname string `json:"uname"`
			Usign string `json:"usign"`
			Fans  int    `json:"fans"`
			Upic  string `json:"upic"`
		} `json:"result"`
	} `json:"data"`
}
